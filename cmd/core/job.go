package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	"github.com/sath-run/engine/cmd/utils"
	pb "github.com/sath-run/engine/pkg/protobuf"
	"google.golang.org/grpc/metadata"
)

var (
	ErrNoJob = errors.New("No job")
)

func JobStatusText(enum pb.EnumExecStatus) string {
	switch enum {
	case pb.EnumExecStatus_EES_STARTED:
		return "started"
	case pb.EnumExecStatus_EES_PULLING_IMAGE:
		return "pulling-image"
	case pb.EnumExecStatus_EES_DOWNLOADING_INPUTS:
		return "downloading"
	case pb.EnumExecStatus_EES_PROCESSING_INPUTS:
		return "preprocessing"
	case pb.EnumExecStatus_EES_RUNNING:
		return "running"
	case pb.EnumExecStatus_EES_PROCESSING_OUPUTS:
		return "postprocessing"
	case pb.EnumExecStatus_EES_UPLOADING_OUTPUTS:
		return "uploading"
	case pb.EnumExecStatus_EES_SUCCESS:
		return "success"
	case pb.EnumExecStatus_EES_CANCELED:
		return "cancelled"
	case pb.EnumExecStatus_EES_ERROR:
		return "error"
	case pb.EnumExecStatus_EES_PAUSED:
		return "paused"
	default:
		return "unspecified"
	}
}

type JobContext struct {
	mu                sync.RWMutex
	status            JobStatus
	statusSubscribers []chan JobStatus
	pauseChannel      chan bool
	stream            pb.Engine_NotifyExecStatusClient
}

var jobContext = JobContext{}

func (c *JobContext) Lock() {
	// utils.LogError(errors.New("Lock"))
	c.mu.Lock()
}

func (c *JobContext) UnLock() {
	// utils.LogError(errors.New("UnLock"))
	c.mu.Unlock()
}

func (c *JobContext) RLock() {
	// utils.LogError(errors.New("RLock"))
	c.mu.RLock()
}

func (c *JobContext) RUnLock() {
	// utils.LogError(errors.New("RUnLock"))
	c.mu.RUnlock()
}

func (c *JobContext) Pause() {
	// utils.LogError(errors.New("Pause"))
	<-c.pauseChannel
}

func (c *JobContext) Resume() {
	// utils.LogError(errors.New("Resume"))
	select {
	case c.pauseChannel <- false:
	default:
		// utils.LogDebug("Fails to resume")
	}
}

type JobStatus struct {
	Id          string
	Image       string
	ContainerId string
	Status      pb.EnumExecStatus
	Progress    float64
	Message     string
	Paused      bool
	CreatedAt   time.Time
	CompletedAt time.Time
	UpdatedAt   time.Time
	GpuOpts     string
	Outputs     []*pb.ExecOutput
}

func (s JobStatus) IsNil() bool {
	return s.Id == ""
}

func init() {
	jobContext.pauseChannel = make(chan bool, 1)
	jobContext.pauseChannel <- false
}

func processInputs(dir string, job *pb.JobGetResponse, status JobStatus) error {
	files := job.GetInputs()
	dataDir := filepath.Join(dir, "/data")
	status.Status = pb.EnumExecStatus_EES_DOWNLOADING_INPUTS
	for _, file := range files {
		status.Message = fmt.Sprintf("start download %s", file.Path)
		populateJobStatus(status)
		filePath := filepath.Join(dataDir, file.Path)
		err := func() error {
			out, err := os.Create(filePath)
			if err != nil {
				return err
			}
			defer out.Close()

			resp, err := retryablehttp.Get(file.Path)
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
			return nil
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

func processOutputs(dir string, job *pb.JobGetResponse) ([]*pb.ExecOutput, error) {
	outputs := make([]*pb.ExecOutput, len(job.GetOutputs()))
	outputDir := filepath.Join(dir, "/output")

	for i, output := range job.GetOutputs() {
		outputs[i] = &pb.ExecOutput{
			Id:     output.Id,
			Status: pb.ExecOutputStatus_EOS_SUCCESS,
		}
		path := filepath.Join(outputDir, output.Path)
		data, err := os.Open(path)
		if err != nil {
			utils.LogDebug("file not found:", path)
			continue
		}
		defer data.Close()

		if output.Req == nil {
			// if output request is not specified, return file content
			fs, err := data.Stat()
			if err != nil {
				return nil, err
			}
			fileSizeInMB := float64(fs.Size()) / 1024 / 1024
			if fileSizeInMB > 1 {
				return nil, fmt.Errorf(
					"file %s is too large, size limit is 1MB, actual size is %.2fMB",
					output.Path,
					fileSizeInMB)
			}
			bytes, err := io.ReadAll(data)
			if err != nil {
				return nil, err
			}
			outputs[i].Content = bytes
		} else {
			// upload output file to url
			req, err := retryablehttp.NewRequest(output.Req.Method, output.Req.Url, data)
			if err != nil {
				return nil, err
			}
			for _, header := range output.Req.Headers {
				req.Header.Set(header.Name, header.Value)
			}
			resp, err := retryablehttp.NewClient().Do(req)
			if err != nil {
				return nil, err
			} else if resp.StatusCode < 200 || resp.StatusCode >= 400 {
				data, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				return nil, fmt.Errorf("fail to upload data, stats: %d, data: %s", resp.StatusCode, string(data))
			}
		}

	}

	return outputs, nil
}

func RunSingleJob(ctx context.Context, orgId string) error {
	job, err := g.grpcClient.GetNewJob(ctx, &pb.JobGetRequest{
		Version:        VERSION,
		OrganizationId: orgId,
	})

	if err != nil {
		return err
	}

	if job == nil || len(job.ExecId) == 0 {
		return ErrNoJob
	}
	c := g.ContextWithToken(context.TODO())
	c = metadata.AppendToOutgoingContext(c, "exec_id", job.ExecId)
	stream, err := g.grpcClient.NotifyExecStatus(c)
	if err != nil {
		return err
	}
	jobContext.Lock()
	jobContext.stream = stream
	jobContext.UnLock()
	status := JobStatus{
		Id:        job.ExecId,
		CreatedAt: time.Now(),
		Progress:  0,
		GpuOpts:   job.GpuOpt.String(),
	}
	outputs, err := RunJob(ctx, job, status)
	status.CompletedAt = time.Now()
	status.Outputs = outputs
	if err != nil {
		utils.LogError(err)
		if errors.Is(err, context.Canceled) {
			status.Status = pb.EnumExecStatus_EES_CANCELED
			status.Message = "user canceled"
		} else {
			status.Status = pb.EnumExecStatus_EES_ERROR
			status.Message = fmt.Sprintf("%+v", err)
		}
	} else {
		status.Progress = 100
		status.Status = pb.EnumExecStatus_EES_SUCCESS
	}

	err = populateJobStatus(status)
	if err != nil {
		// if fail to populate job status to server, we still need to notify clients
		status.Status = pb.EnumExecStatus_EES_ERROR
		status.Message = err.Error()
		notifyJobStatusToClients(status)
	}
	jobContext.RLock()
	_, err = jobContext.stream.CloseAndRecv()
	jobContext.RUnLock()
	return err
}

func RunJob(ctx context.Context, job *pb.JobGetResponse, status JobStatus) ([]*pb.ExecOutput, error) {
	if ref, err := reference.ParseNormalizedNamed(job.Image.Url); err == nil {
		status.Image = ref.Name()
	} else {
		status.Image = job.Image.Url
	}

	status.Status = pb.EnumExecStatus_EES_STARTED
	populateJobStatus(status)

	utils.LogDebug("RunJob: ", job)

	dir, err := os.MkdirTemp(g.localDataDir, "sath_tmp_*")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			utils.LogError(err)
		}
	}()

	if err := os.MkdirAll(filepath.Join(dir, "/data"), os.ModePerm); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "/output"), os.ModePerm); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "/source"), os.ModePerm); err != nil {
		return nil, err
	}

	localDataDir := dir
	hostDir := localDataDir
	if len(g.hostDataDir) > 0 {
		tmpDirName := filepath.Base(dir)
		hostDir = filepath.Join(g.hostDataDir, tmpDirName)
	}

	if err = PullImage(ctx, g.dockerClient, job.Image.Url, types.ImagePullOptions{
		RegistryAuth: job.Image.Auth,
	}, func(text string) {
		status.Status = pb.EnumExecStatus_EES_PULLING_IMAGE
		status.Message = text
		populateJobStatus(status)
	}); err != nil {
		return nil, err
	}

	status.Status = pb.EnumExecStatus_EES_PROCESSING_INPUTS
	if err = processInputs(localDataDir, job, status); err != nil {
		return nil, err
	}

	status.Status = pb.EnumExecStatus_EES_RUNNING
	populateJobStatus(status)

	if len(job.Volume.Data) == 0 {
		job.Volume.Data = "/data"
	}
	if len(job.Volume.Source) == 0 {
		job.Volume.Source = "/source"
	}
	if len(job.Volume.Output) == 0 {
		job.Volume.Output = "/output"
	}

	binds := []string{
		fmt.Sprintf("%s:%s", filepath.Join(hostDir, "/data"), job.Volume.Data),
		fmt.Sprintf("%s:%s", filepath.Join(hostDir, "/source"), job.Volume.Source),
		fmt.Sprintf("%s:%s", filepath.Join(hostDir, "/output"), job.Volume.Output),
	}

	containerName := fmt.Sprintf("sath-%s", job.ExecId)

	gpuOpts := ""
	if job.GpuOpt == pb.GpuOpt_EGO_REQUIRED || job.GpuOpt == pb.GpuOpt_EGO_PREFERRED {
		gpuOpts = "all"
	}

	// create container
	containerId, err := CreateContainer(
		ctx, g.dockerClient, job.Cmd, job.Image.Url,
		gpuOpts, containerName, binds)
	if err != nil {
		return nil, err
	}
	status.ContainerId = containerId
	if err := ExecImage(ctx, g.dockerClient, containerId, func(line string) {
		status.Status = pb.EnumExecStatus_EES_RUNNING
		status.Message = line
		populateJobStatus(status)
	}); err != nil {
		return nil, err
	}
	status.Status = pb.EnumExecStatus_EES_PROCESSING_OUPUTS
	populateJobStatus(status)

	outputs, err := processOutputs(dir, job)
	if err != nil {
		return nil, err
	}
	return outputs, nil
}

func populateJobStatus(status JobStatus) error {
	err := notifyJobStatusToServer(status, 0, 3)
	notifyJobStatusToClients(status)
	return errors.WithStack(err)
}

func notifyJobStatusToServer(status JobStatus, retry int, maxRetry int) error {
	status.UpdatedAt = time.Now()
	jobContext.Lock()
	st := status.Status
	if status.Paused && !jobContext.status.Paused {
		st = pb.EnumExecStatus_EES_PAUSED
	}
	req := &pb.ExecNotificationRequest{
		Status:   st,
		Message:  status.Message,
		Progress: float32(status.Progress),
		Outputs:  status.Outputs,
	}
	if status.GpuOpts != "" {
		req.GpuStats = []*pb.GpuStats{
			{Id: 0},
		}
	}
	if err := jobContext.stream.Send(req); errors.Is(err, io.EOF) {
		if retry >= maxRetry {
			jobContext.UnLock()
			return errors.Wrap(err, "max retry exceeded")
		}
		// server may ternimate stream connection after a configured idle timeout
		// in this case, reconnect and try it again
		stream, err := g.grpcClient.NotifyExecStatus(jobContext.stream.Context())
		if err != nil {
			jobContext.UnLock()
			return err
		}
		jobContext.stream.CloseSend()
		jobContext.stream = stream
		jobContext.UnLock()
		return notifyJobStatusToServer(status, retry+1, maxRetry)
	} else if err != nil {
		err = errors.WithStack(err)
		jobContext.UnLock()
		return err
	}
	jobContext.UnLock()
	return nil
}

func notifyJobStatusToClients(status JobStatus) error {
	jobContext.Pause()
	jobContext.Lock()
	defer jobContext.UnLock()
	for _, c := range jobContext.statusSubscribers {
		c <- status
	}
	status.Message = ""
	jobContext.status = status
	if !status.Paused {
		jobContext.Resume()
	}
	return nil
}

func SubscribeJobStatus(channel chan JobStatus) {
	jobContext.Lock()
	defer jobContext.UnLock()

	jobContext.statusSubscribers = append(jobContext.statusSubscribers, channel)
}

func UnsubscribeJobStatus(channel chan JobStatus) {
	jobContext.Lock()
	defer jobContext.UnLock()

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
	jobContext.RLock()
	defer jobContext.RUnLock()

	if jobContext.status.IsNil() {
		return nil
	} else {
		var status JobStatus = jobContext.status
		return &status
	}
}

func Pause(execId string) bool {
	var status JobStatus
	jobContext.Lock()

	if jobContext.status.IsNil() {
		jobContext.UnLock()
		return false
	}
	if len(execId) > 0 && jobContext.status.Id != execId {
		jobContext.UnLock()
		return false
	}
	if !jobContext.status.Paused && jobContext.status.Status == pb.EnumExecStatus_EES_RUNNING && len(jobContext.status.ContainerId) > 0 {
		if err := g.dockerClient.ContainerPause(context.TODO(), jobContext.status.ContainerId); err != nil {
			utils.LogError(err)
		}
	}

	status = jobContext.status
	status.Paused = true
	jobContext.UnLock()
	populateJobStatus(status)
	return true
}

func Resume(execId string) bool {
	var status JobStatus
	jobContext.Lock()
	if jobContext.status.IsNil() {
		jobContext.UnLock()
		return false
	}
	if len(execId) > 0 && jobContext.status.Id != execId {
		jobContext.UnLock()
		return false
	}
	if jobContext.status.Paused &&
		jobContext.status.Status == pb.EnumExecStatus_EES_RUNNING &&
		len(jobContext.status.ContainerId) > 0 {
		if err := g.dockerClient.ContainerUnpause(context.TODO(), jobContext.status.ContainerId); err != nil {
			utils.LogError(err)
		}
	}
	jobContext.Resume()
	status = jobContext.status
	status.Paused = false
	jobContext.UnLock()
	populateJobStatus(status)
	return true
}
