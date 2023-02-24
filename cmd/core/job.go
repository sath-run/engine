package core

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	"github.com/sath-run/engine/cmd/utils"
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

func processOutputs(dir string, job *pb.JobGetResponse) ([]*pb.File, error) {
	var files []*pb.File

	for _, output := range job.Outputs {
		data, err := os.ReadFile(filepath.Join(dir, output))
		if errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, err
		}
		file := &pb.File{
			Name: output,
			Content: &pb.File_Data{
				Data: data,
			},
		}
		files = append(files, file)
	}

	return files, nil
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
	req, err := runJob(ctx, job, &status)
	status.CompletedAt = time.Now()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			status.Status = pb.EnumJobStatus_EJS_CANCELLED
			status.Message = "user cancelled"
		} else {
			status.Status = pb.EnumJobStatus_EJS_ERROR
			status.Message = err.Error()
		}
		req = &pb.JobPopulateRequest{
			ExecId: job.ExecId,
			Status: http.StatusInternalServerError,
			Result: []byte(status.Message),
		}
	} else {
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

func runJob(ctx context.Context, job *pb.JobGetResponse, status *JobStatus) (*pb.JobPopulateRequest, error) {
	imageConfig := DockerImageConfig{
		Repository: job.Image.Repository,
		Digest:     job.Image.Digest,
		Tag:        job.Image.Tag,
		Uri:        job.Image.Uri,
	}

	status.Image = imageConfig.Image()
	status.Status = pb.EnumJobStatus_EJS_READY
	populateJobStatus(status)

	dir, err := utils.GetExecutableDir()
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(filepath.Join(dir, "data"), os.ModePerm)
	if err != nil {
		return nil, err
	}
	dir, err = os.MkdirTemp(filepath.Join(dir, "data"), "sath_tmp_*")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("%+v\n", err)
		}
	}()

	if err = PullImage(ctx, &imageConfig, func(text string) {
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

	hostDir := dir
	if len(g.hostDataDir) > 0 {
		hostDir = filepath.Join(g.hostDataDir, filepath.Base(dir))
	}

	if err := ExecImage(
		ctx, job.Cmds, imageConfig.Image(), hostDir, job.VolumePath,
		job.GpuOpts, &status.ContainerId, func(progress float64) {
			status.Status = pb.EnumJobStatus_EJS_RUNNING
			status.Progress = progress
			populateJobStatus(status)
		}); err != nil {
		return nil, err
	}

	if data, err := os.ReadFile(filepath.Join(dir, "sath.err")); err != nil && err != os.ErrNotExist {
		return nil, err
	} else if len(data) > 0 {
		return nil, err
	}

	files, err := processOutputs(dir, job)
	if err != nil {
		return nil, err
	}

	return &pb.JobPopulateRequest{
		ExecId: job.ExecId,
		Status: http.StatusOK,
		Files:  files,
	}, nil
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
