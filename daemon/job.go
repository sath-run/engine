package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	pb "github.com/sath-run/engine/daemon/protobuf"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/metadata"
)

type JobNotification struct {
	Id      string
	Message string
	Current uint
	Total   uint
	Flag    uint64
}

type JobOutput struct {
	Id      string
	Status  pb.ExecOutputStatus
	Message string
	Content []byte
}

type Job struct {
	c           *Connection
	cli         *client.Client
	metadata    *pb.JobGetResponse
	container   *Container
	rm          *ResourceManager
	resourceDir string

	dir       string
	stream    pb.Engine_NotifyExecStatusClient
	streamMu  sync.Mutex
	err       error
	queue     chan *Job
	state     pb.EnumExecState
	createdAt time.Time
	outputs   []JobOutput

	logger zerolog.Logger
}

func newJob(ctx context.Context, c *Connection, cli *client.Client, queue chan *Job, rm *ResourceManager, dir string, meta *pb.JobGetResponse) (*Job, error) {
	if err := os.Mkdir(dir, os.ModePerm); err != nil {
		return nil, err
	}
	ctx = metadata.AppendToOutgoingContext(ctx, "id", meta.JobId)
	stream, err := c.NotifyExecStatus(ctx)
	if err != nil {
		return nil, err
	}
	job := &Job{
		c:         c,
		cli:       cli,
		metadata:  meta,
		rm:        rm,
		state:     pb.EnumExecState_EES_INITIALIZED,
		createdAt: time.Now(),
		stream:    stream,
		queue:     queue,
		dir:       dir,
		logger:    log.With().Str("job", meta.JobId).Logger(),
	}
	if err := os.MkdirAll(job.dataDir(), os.ModePerm); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(job.outputDir(), os.ModePerm); err != nil {
		return nil, err
	}
	if dir, err := filepath.Abs(filepath.Join(job.dir, "..", "resource_"+job.metadata.ResourceId)); err != nil {
		return nil, err
	} else {
		job.resourceDir = dir
	}
	job.logger.Debug().Msg("job created")
	return job, nil
}

func (job *Job) dataDir() string {
	return filepath.Join(job.dir, "data")
}

func (job *Job) outputDir() string {
	return filepath.Join(job.dir, "output")
}

func (job *Job) notifyStatusToRemote(notification JobNotification) error {
	req := pb.ExecNotificationRequest{
		State:   job.state,
		Id:      notification.Id,
		Message: notification.Message,
		Current: uint64(notification.Current),
		Total:   uint64(notification.Total),
		Flag:    notification.Flag,
	}
	if job.err != nil {
		req.Message = job.err.Error()
		req.Flag |= uint64(pb.EnumExecFlag_EEF_ERROR)
	} else if req.State == pb.EnumExecState_EES_SUCCESS {
		for _, output := range job.outputs {
			req.Outputs = append(req.Outputs, &pb.ExecOutput{
				Id:      output.Id,
				Status:  output.Status,
				Message: output.Message,
				Content: output.Content,
			})
		}
	}

	job.logger.Trace().Str("state", job.state.String()).Any("notification", notification).Send()
	job.streamMu.Lock()
	err := job.stream.Send(&req)
	job.streamMu.Unlock()
	return err
}

func (job *Job) setState(state pb.EnumExecState) {
	job.state = state
	if state != pb.EnumExecState_EES_SUCCESS {
		job.notifyStatusToRemote(JobNotification{})
	}
}

func (job *Job) handleCompletion() {
	if err := job.notifyStatusToRemote(JobNotification{}); err != nil {
		job.err = errors.Join(job.err, err)
		job.logger.Warn().Err(err).Msg("err notify status")
	}

	// block and wait to close stream, error could be ignored
	if _, err := job.stream.CloseAndRecv(); err != nil {
		job.err = errors.Join(job.err, err)
		job.logger.Warn().Err(err).Msg("err CloseAndRecv")
	}

	if err := os.RemoveAll(job.dir); err != nil {
		job.err = errors.Join(job.err, err)
		job.logger.Warn().Err(err).Msg("err RemoveAll")
	}
}

func (job *Job) preprocess() {
	var err error
	defer func() {
		job.err = err
		// notify scheduler
		job.queue <- job
	}()
	if err = job.prepareImage(); err != nil {
		return
	}
	if err = job.downloadResources(); err != nil {
		return
	}
	if err = job.processResources(); err != nil {
		return
	}
	if err = job.downloadInputs(); err != nil {
		return
	}
	if err = job.processInputs(); err != nil {
		return
	}
	job.setState(pb.EnumExecState_EES_QUEUING)
}

func (job *Job) run() {
	var err error
	defer func() {
		job.err = err
		// notify scheduler
		job.queue <- job
	}()
	if err = job.prepareContainer(); err != nil {
		return
	}
	if err = job.runTask(); err != nil {
		return
	}
	// move output to job's folder
	if err = mvDir(job.container.outputDir(), job.outputDir()); err != nil {
		return
	}
}

func (job *Job) postprocess() {
	var err error
	defer func() {
		job.err = err
		// notify scheduler
		job.queue <- job
	}()
	if err = job.processOutputs(); err != nil {
		return
	}
	job.setState(pb.EnumExecState_EES_SUCCESS)
}

func (job *Job) prepareImage() error {
	job.setState(pb.EnumExecState_EES_PREPARING_IMAGE)

	reader, err := job.cli.ImagePull(context.Background(), job.metadata.Image.Url, image.PullOptions{
		RegistryAuth: job.metadata.Image.Auth,
	})

	if err != nil {
		return err
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)

	type Info struct {
		Id             string `json:"id"`
		Status         string `json:"status,omitempty"`
		ProgressDetail struct {
			Current int `json:"current"`
			Total   int `json:"total"`
		} `json:"progressDetail,omitempty"`
	}

	for scanner.Scan() {
		line := scanner.Text()
		var info Info
		if err := json.Unmarshal([]byte(line), &info); err != nil {
			job.logger.Debug().Err(err).Msg("unmarshal docker pull log")
			continue
		}
		if info.Status == "Downloading" || info.Status == "Extracting" {
			notification := JobNotification{
				Id:      info.Id,
				Message: info.Status,
				Current: uint(info.ProgressDetail.Current),
				Total:   uint(info.ProgressDetail.Total),
			}
			err := job.notifyStatusToRemote(notification)
			if err != nil {
				job.logger.Err(err).Msg("fail to notify ")
			}
		}
	}
	return scanner.Err()
}

func (job *Job) downloadResources() error {
	job.setState(pb.EnumExecState_EES_DOWNLOADING_RESOURCES)
	// sequentially download each resource file
	// TODO: batch download and rate limit
	for _, resource := range job.metadata.Resources {
		if err := job.downloadFile(context.TODO(), resource.Path, job.resourceDir, resource.Req.Url); err != nil {
			return err
		}
	}
	return nil
}

func (job *Job) downloadFile(ctx context.Context, path string, dir string, url string) error {
	id := path
	// sanitize path in case it contains relative path like: "../.."
	// which may hack the directory structure limitation in client-engine
	path, err := filepath.Abs(filepath.Join("/", path))
	if err != nil {
		return err
	}
	path = filepath.Join(dir, path)

	// make dir, error can be ignored
	os.MkdirAll(filepath.Dir(path), os.ModePerm)
	resp := job.rm.Download(ctx, path, url)
	progress := 0.0

	// check for download progress every 500 ms
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			newProgress := resp.Progress()
			if newProgress-progress > 0.01 || newProgress == 1 {
				progress = newProgress
				if err := job.notifyStatusToRemote(JobNotification{
					Id:      id,
					Current: uint(resp.Current()),
					Total:   uint(resp.Total()),
				}); err != nil {
					resp.Cancel()
					return nil
				}
			}

		case <-resp.Done:
			// check for errors
			if err := resp.Err(); err != nil {
				return err
			}
			return nil
		}
	}
}

func (job *Job) processResources() error {
	job.setState(pb.EnumExecState_EES_PROCESSING_RESOURCES)
	return nil
}

func (job *Job) downloadInputs() error {
	job.setState(pb.EnumExecState_EES_DOWNLOADING_INPUTS)
	files := job.metadata.Inputs
	g, ctx := errgroup.WithContext(context.TODO())

	// TODO: limit bandwidth
	for _, file := range files {
		g.Go(func() error {
			return job.downloadFile(ctx, file.Path, job.dataDir(), file.Req.Url)
		})
	}
	err := g.Wait()
	return err
}

func (job *Job) processInputs() error {
	job.setState(pb.EnumExecState_EES_PROCESSING_INPUTS)
	// TODO
	return nil
}

func (job *Job) prepareContainer() error {
	job.setState(pb.EnumExecState_EES_PREPARING_CONTAINER)
	ctn := job.container

	// if container has not been created by docker, create one
	if ctn.id == "" {
		if err := ctn.init(context.TODO()); err != nil {
			return err
		}
	}

	// cleanup previous run
	if err := emptyDir(ctn.dataDir()); err != nil {
		return err
	}
	if err := emptyDir(ctn.outputDir()); err != nil {
		return err
	}
	// move data files to container dir
	if err := mvDir(job.dataDir(), ctn.dataDir()); err != nil {
		return err
	}
	return nil
}

func (job *Job) runTask() error {
	job.setState(pb.EnumExecState_EES_RUNNING)
	hijack, err := job.container.run(context.TODO(), job.metadata.Cmd)
	if err != nil {
		return err
	}
	defer hijack.Close()

	time.Sleep(10 * time.Second)
	scanner := bufio.NewScanner(hijack.Reader)
	for scanner.Scan() {
		// TODO: update progress
		job.notifyStatusToRemote(JobNotification{
			Message: scanner.Text(),
		})
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func (job *Job) processOutputs() error {
	job.setState(pb.EnumExecState_EES_PROCESSING_OUPUTS)
	job.outputs = make([]JobOutput, len(job.metadata.Outputs))

	g, ctx := errgroup.WithContext(context.TODO())
	for i, output := range job.metadata.Outputs {
		job.outputs[i] = JobOutput{
			Id: output.Id,
		}
		g.Go(func() (err error) {
			// TODO: support optional output
			path := filepath.Join(job.outputDir(), output.Path)

			defer func() {
				if err != nil {
					job.outputs[i].Status = pb.ExecOutputStatus_EOS_ERROR
					job.outputs[i].Message = err.Error()
				}
			}()

			data, err := os.Open(path)
			if err != nil {
				return err
			}
			defer data.Close()

			if output.Req == nil {
				// if output request is not specified, return file content
				if fs, err := data.Stat(); err != nil {
					return err
				} else {
					// TODO: limit total file size
					fileSizeInKB := float64(fs.Size()) / 1024
					if fileSizeInKB > 128 {
						return fmt.Errorf(
							"file %s is too large, size limit is 128K, actual size is %.2fKB",
							output.Path,
							fileSizeInKB)
					}
				}
				if bytes, err := io.ReadAll(data); err != nil {
					return err
				} else {
					job.outputs[i].Content = bytes
				}
			} else {
				// upload output file to url
				var (
					req  *http.Request
					resp *http.Response
				)
				req, err = http.NewRequestWithContext(ctx, output.Req.Method, output.Req.Url, data)
				if err != nil {
					return
				}
				for _, header := range output.Req.Headers {
					req.Header.Set(header.Name, header.Value)
				}
				resp, err = http.DefaultClient.Do(req)
				if err != nil {
					return
				} else if resp.StatusCode < 200 || resp.StatusCode >= 400 {
					data, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					return fmt.Errorf("fail to upload data, stats: %d, data: %s", resp.StatusCode, string(data))
				}
			}
			return
		})
	}
	return g.Wait()
}

func emptyDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func mvDir(src string, dst string) error {
	d, err := os.Open(src)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.Rename(filepath.Join(src, name), filepath.Join(dst, name))
		if err != nil {
			return err
		}
	}
	return nil
}
