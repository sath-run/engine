package core

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	pb "github.com/sath-run/engine/pkg/protobuf"
)

var (
	ErrNoJob = errors.New("No job")
)

func JobStatusText(enum pb.EnumJobStatus) string {
	switch enum {
	case pb.EnumJobStatus_EJS_READY:
		return "ready"
	case pb.EnumJobStatus_EJS_PULLING_IMAGE:
		return "pulling-image"
	case pb.EnumJobStatus_EJS_PROCESSING_INPUTS:
		return "preprocessing"
	case pb.EnumJobStatus_EJS_RUNNING:
		return "running"
	case pb.EnumJobStatus_EJS_PROCESSING_OUPUTS:
		return "uploading"
	case pb.EnumJobStatus_EJS_POPULATING:
		return "populating"
	case pb.EnumJobStatus_EJS_SUCCESS:
		return "success"
	case pb.EnumJobStatus_EJS_CANCELLED:
		return "cancelled"
	case pb.EnumJobStatus_EJS_ERROR:
		return "error"
	default:
		return "unspecified"
	}
}

var jobContext = struct {
	mu                sync.RWMutex
	jobStatus         *JobStatus
	statusSubscribers []chan JobStatus
}{}

type JobStatus struct {
	Id          string
	Image       string
	ContainerId string
	Status      pb.EnumJobStatus
	Progress    float64
	Message     string
	CreatedAt   time.Time
	CompletedAt time.Time
}

func processInputs(dir string, job *pb.JobGetResponse) error {
	files := job.GetFiles()
	for _, file := range files {
		filePath := filepath.Join(dir, file.Name)
		if remote := file.GetRemote(); remote != nil {
			err := func() error {
				out, err := os.Create(filePath)
				if err != nil {
					return err
				}
				defer out.Close()

				if remote.FetchMethod == pb.EnumFileFetchMethod_EFFM_HTTP {
					resp, err := retryablehttp.Get(file.GetRemote().Uri)
					if err != nil {
						return err
					}
					defer resp.Body.Close()
					if err != nil {
						return err
					}
					_, err = io.Copy(out, resp.Body)
					if err != nil {
						return err
					}
				} else if remote.FetchMethod == pb.EnumFileFetchMethod_EFFM_GRPC_STREAM {
					// TODO
				} else {
					// TODO
				}
				return nil
			}()
			if err != nil {
				return err
			}
		} else if data := file.GetData(); len(data) > 0 {
			if err := os.WriteFile(filepath.Join(dir, file.Name), data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

func ProcessOutputs(dir string, execId string, outputs []string) ([]*pb.File, error) {
	if len(outputs) == 0 {
		return nil, nil
	}
	req := &pb.FileUploadRequest{
		Operation:   pb.EnumOperation_EO_EXECUTION,
		OperationId: execId,
	}
	for _, output := range outputs {
		path := filepath.Join(dir, output)
		stat, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		base := filepath.Base(output)
		req.Files = append(req.Files, &pb.FileInfo{
			Name: base,
			Size: uint64(stat.Size()),
		})
	}
	res, err := g.grpcClient.RequestFileUpload(g.ContextWithToken(context.Background()), req)
	if err != nil {
		return nil, err
	}
	if len(res.Infos) != len(outputs) {
		return nil, fmt.Errorf("res info length does not equal: %d : %d", len(res.Infos), len(outputs))
	}

	var files []*pb.File
	for i, output := range outputs {
		info := res.Infos[i]
		path := filepath.Join(dir, output)
		file, err := processOutput(info, path)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}

	return files, nil
}

func processOutput(info *pb.FileUploadInfo, path string) (*pb.File, error) {
	var file = &pb.File{
		Name: filepath.Base(path),
	}
	switch info.GetOp() {
	case pb.EnumFileUploadOperation_EFUO_EMBED:
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		file.Content = &pb.File_Data{
			Data: data,
		}
	case pb.EnumFileUploadOperation_EFUO_HTTP_PUT, pb.EnumFileUploadOperation_EFUO_HTTP_POST:
		var method = "POST"
		if info.GetOp() == pb.EnumFileUploadOperation_EFUO_HTTP_PUT {
			method = "PUT"
		}
		f, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		req, err := retryablehttp.NewRequest(method, info.Url, f)
		if err != nil {
			return nil, err
		}
		for key, value := range info.Headers {
			req.Header.Add(key, value)
		}
		res, err := retryablehttp.NewClient().Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		if res.StatusCode < 200 || res.StatusCode >= 400 {
			body, _ := io.ReadAll(res.Body)
			return nil, errors.Errorf("file upload err, code: %d, body: %s", res.StatusCode, string(body))
		}
		file.Content = &pb.File_Remote{
			Remote: &pb.FileUri{
				Uri: info.Id,
			},
		}
	default:
		return nil, fmt.Errorf("unsupported op: %s", info.Op.String())
	}
	return file, nil
}

func RunSingleJob(ctx context.Context) error {
	job, err := g.grpcClient.GetNewJob(ctx, &pb.JobGetRequest{
		Version: VERSION,
	})

	if err != nil {
		return err
	}

	if job == nil || len(job.ExecId) == 0 {
		return ErrNoJob
	}

	status := JobStatus{
		Id:        job.ExecId,
		CreatedAt: time.Now(),
		Progress:  0,
	}
	var req = &pb.JobPopulateRequest{
		ExecId: job.ExecId,
		Status: http.StatusOK,
	}
	files, err := RunJob(ctx, job, &status)
	status.CompletedAt = time.Now()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			status.Status = pb.EnumJobStatus_EJS_CANCELLED
			status.Message = "user cancelled"
		} else {
			status.Status = pb.EnumJobStatus_EJS_ERROR
			status.Message = err.Error()
			log.Printf("%+v\n", err)
		}
		req.Status = http.StatusInternalServerError
		req.Result = []byte(status.Message)
	} else {
		req.Files = files
		status.Progress = 100
		status.Status = pb.EnumJobStatus_EJS_POPULATING
	}

	populateJobStatus(&status)

	if _, err := g.grpcClient.PopulateJobResult(ctx, req); err != nil {
		status.Status = pb.EnumJobStatus_EJS_ERROR
		status.Message = err.Error()
	} else {
		status.Status = pb.EnumJobStatus_EJS_SUCCESS
	}
	populateJobStatus(&status)

	return err
}

func RunJob(ctx context.Context, job *pb.JobGetResponse, status *JobStatus) ([]*pb.File, error) {
	imageConfig := DockerImageConfig{
		Repository: job.Image.Repository,
		Digest:     job.Image.Digest,
		Tag:        job.Image.Tag,
		Uri:        job.Image.Uri,
	}

	status.Image = imageConfig.Image()
	status.Status = pb.EnumJobStatus_EJS_READY
	populateJobStatus(status)

	dir, err := os.MkdirTemp(g.localDataDir, "sath_tmp_*")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("%+v\n", err)
		}
	}()

	if err = PullImage(ctx, g.dockerClient, &imageConfig, func(text string) {
		status.Status = pb.EnumJobStatus_EJS_PULLING_IMAGE
		status.Message = text
		populateJobStatus(status)
	}); err != nil {
		return nil, err
	}

	status.Status = pb.EnumJobStatus_EJS_PROCESSING_INPUTS
	populateJobStatus(status)

	if err = processInputs(dir, job); err != nil {
		return nil, err
	}

	status.Status = pb.EnumJobStatus_EJS_RUNNING
	populateJobStatus(status)

	localDataDir := dir
	hostDir := localDataDir
	if len(g.hostDataDir) > 0 {
		hostDir = filepath.Join(g.hostDataDir, filepath.Base(dir))
	}

	if err := ExecImage(
		ctx, g.dockerClient, job.Cmds, imageConfig.Image(), localDataDir, hostDir, job.VolumePath,
		job.GpuOpts, &status.ContainerId, func(progress float64) {
			status.Status = pb.EnumJobStatus_EJS_RUNNING
			status.Progress = progress
			populateJobStatus(status)
		}); err != nil {
		return nil, err
	}

	if data, err := os.ReadFile(filepath.Join(dir, "sath.err")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	} else if len(data) > 0 {
		return nil, err
	}

	status.Status = pb.EnumJobStatus_EJS_PROCESSING_OUPUTS
	populateJobStatus(status)

	files, err := ProcessOutputs(dir, job.ExecId, job.Outputs)
	if err != nil {
		return nil, err
	}

	return files, nil
}

func populateJobStatus(status *JobStatus) {
	jobContext.mu.Lock()
	defer jobContext.mu.Unlock()
	jobContext.jobStatus = status

	for _, c := range jobContext.statusSubscribers {
		c <- *status
	}
	status.Message = ""
}

func SubscribeJobStatus(channel chan JobStatus) {
	jobContext.mu.Lock()
	defer jobContext.mu.Unlock()

	jobContext.statusSubscribers = append(jobContext.statusSubscribers, channel)
}

func UnsubscribeJobStatus(channel chan JobStatus) {
	jobContext.mu.Lock()
	defer jobContext.mu.Unlock()

	subscribers := make([]chan JobStatus, 0)
	for _, c := range jobContext.statusSubscribers {
		if c != channel {
			subscribers = append(subscribers, c)
		}
	}
	jobContext.statusSubscribers = subscribers
	close(channel)
}

func GetJobStatus() *JobStatus {
	jobContext.mu.RLock()
	defer jobContext.mu.RUnlock()

	if jobContext.jobStatus == nil {
		return nil
	} else {
		var status JobStatus = *jobContext.jobStatus
		return &status
	}
}
