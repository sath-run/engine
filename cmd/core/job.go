package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"

	pb "github.com/sath-run/engine/pkg/protobuf"
)

const (
	JOB_STATUS_UNSPECIFIED = iota
	JOB_STATUS_READY
	JOB_STATUS_PULLING_IMAGE
	JOB_STATUS_RUNNING
	JOB_STATUS_POPULATING
	JOB_STATUS_SUCCESS
	JOB_STATUS_ERROR
)

var jobContext = struct {
	mu                sync.RWMutex
	jobStatus         *JobStatus
	statusSubscribers []chan JobStatus
}{}

type JobStatus struct {
	Id       string
	Status   int
	Progress float64
	Message  string
}

type JobExecResult struct {
	JobId string
}

func processInputs(dir string, job *pb.JobGetResponse) error {
	files := job.GetFiles()
	for _, file := range files {
		if uri := file.GetRemote(); uri != nil {
			// TOOD
		} else if data := file.GetData(); len(data) > 0 {
			if err := os.WriteFile(filepath.Join(dir, file.Name), data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

func processOutputs(dir string, job *pb.JobGetResponse) ([]byte, error) {
	var (
		data []byte
		err  error
	)

	if len(job.Outputs) == 0 {
		// TODO
	} else if len(job.Outputs) > 1 {
		// TODO
	} else {
		data, err = os.ReadFile(filepath.Join(dir, job.Outputs[0]))
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

func RunSingleJob() error {
	ctx := context.Background()

	job, err := g.grpcClient.GetNewJob(ctx, &pb.JobGetRequest{
		UserId:     "", // TODO
		DeviceId:   "", // TODO
		DeviceInfo: "", // TODO
	})

	if err != nil {
		return err
	}

	if job == nil || len(job.JobId) == 0 {
		return nil
	}

	status := JobStatus{
		Id:       job.JobId,
		Progress: 0,
		Status:   JOB_STATUS_READY,
	}

	populateJobStatus(&status)

	defer func() {
		if err == nil {
			status.Status = JOB_STATUS_ERROR
		} else {
			status.Status = JOB_STATUS_SUCCESS
			status.Progress = 100
		}
		populateJobStatus(&status)
	}()

	dir, err := os.MkdirTemp("", "sath_tmp_*")
	if err != nil {
		return err
	}
	defer func() {
		err = os.RemoveAll(dir)
	}()

	imageConfig := DockerImageConfig{
		Repository: job.Image.Repository,
		Digest:     job.Image.Digest,
		Tag:        job.Image.Tag,
		Uri:        job.Image.Uri,
	}

	if err := PullImage(ctx, &imageConfig, func(text string) {
		status.Status = JOB_STATUS_PULLING_IMAGE
		status.Message = text
		populateJobStatus(&status)
	}); err != nil {
		return err
	}

	if err := processInputs(dir, job); err != nil {
		return err
	}

	status.Status = JOB_STATUS_RUNNING
	populateJobStatus(&status)
	if err = ExecImage(ctx, job.Cmds, imageConfig.Image(), dir, job.VolumePath, func(progress float64) {
		status.Status = JOB_STATUS_RUNNING
		status.Progress = progress
		populateJobStatus(&status)
	}); err != nil {
		return err
	}

	if data, err := os.ReadFile(filepath.Join(dir, "sath_stderr.log")); err == os.ErrNotExist {
		// nothing to do
	} else if err != nil {
		return err
	} else if len(data) > 0 {
		return errors.New(string(data))
	}

	data, err := processOutputs(dir, job)
	if err != nil {
		return err
	}

	status.Status = JOB_STATUS_POPULATING
	populateJobStatus(&status)

	_, err = g.grpcClient.PopulateJobResult(ctx, &pb.JobPopulateRequest{
		JobId:    job.JobId,
		UserId:   "", // TODO
		DeviceId: "", // TODO
		Result:   data,
	})

	if err != nil {
		return err
	}

	return nil
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

func GetCurrentJobStatus() *JobStatus {
	jobContext.mu.RLock()
	defer jobContext.mu.RUnlock()

	if jobContext.jobStatus == nil {
		return nil
	} else {
		var status JobStatus = *jobContext.jobStatus
		return &status
	}
}
