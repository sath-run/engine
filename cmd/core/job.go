package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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

var jobContext = struct {
	mu                sync.RWMutex
	status            *JobStatus
	statusSubscribers []chan JobStatus
	pauseChannel      chan bool
	stream            pb.Engine_NotifyExecStatusClient
}{}

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
}

func init() {
	fmt.Println()
	jobContext.pauseChannel = make(chan bool, 1)
	jobContext.pauseChannel <- false
}

func processInputs(dir string, job *pb.JobGetResponse, status *JobStatus) error {
	files := job.GetInputs()
	dataDir := filepath.Join(dir, "/data")
	status.Status = pb.EnumExecStatus_EES_DOWNLOADING_INPUTS
	for _, file := range files {
		status.Message = fmt.Sprintf("start download %s", file.Name)
		populateJobStatus(status)
		filePath := filepath.Join(dataDir, file.Name)
		err := func() error {
			out, err := os.Create(filePath)
			if err != nil {
				return err
			}
			defer out.Close()

			resp, err := retryablehttp.Get(file.Url)
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

func processOutputs(dir string, job *pb.JobGetResponse) error {
	output := job.GetOutput()
	if output == nil {
		return errors.New("job output is nil")
	}
	outputDir := filepath.Join(dir, "/output")

	// tar + gzip
	var buf bytes.Buffer
	if err := utils.Compress(outputDir, &buf); err != nil {
		return err
	}

	var method string
	switch output.Method {
	case pb.EnumFileRequestMethod_EFRM_HTTP_POST:
		method = "POST"
	case pb.EnumFileRequestMethod_EFRM_HTTP_PUT:
		method = "PUT"
	default:
		method = "GET"
	}

	url := job.Output.Url
	headers := job.Output.Headers
	data := job.Output.Data

	if headers["Content-Type"] == "application/json" {
		body, err := json.Marshal(data)
		if err != nil {
			return err
		}
		req, err := retryablehttp.NewRequest(method, url, body)
		if err != nil {
			return err
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		res, err := retryablehttp.NewClient().Do(req)
		if err != nil {
			return err
		}
		body, err = io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		res.Body.Close()
		var obj struct {
			Url     string            `json:"url"`
			Method  string            `json:"method"`
			Headers map[string]string `json:"headers"`
			Data    map[string]string `json:"data"`
		}
		if err := json.Unmarshal(body, &obj); err != nil {
			return err
		}
		url = obj.Url
		headers = obj.Headers
		method = obj.Method
		data = obj.Data
	}
	if err := uploadOutput(url, method, headers, data, buf); err != nil {
		return err
	}
	return nil
}

func uploadOutput(url, method string, headers, data map[string]string, buf bytes.Buffer) error {
	if headers["Content-Type"] == "multipart/form-data" {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		fieldname := data["SATH_OUTPUT_FILED_NAME"]
		if len(fieldname) == 0 {
			fieldname = "file"
		}
		filename := data["SATH_OUTPUT_FILED_NAME"]
		if len(filename) == 0 {
			filename = "output.tar.gz"
		}
		part, err := writer.CreateFormFile(fieldname, filename)
		if err != nil {
			return err
		}
		_, err = io.Copy(part, &buf)
		if err != nil {
			return err
		}

		for key, val := range data {
			_ = writer.WriteField(key, val)
		}
		if err := writer.Close(); err != nil {
			return err
		}
		req, err := retryablehttp.NewRequest(method, url, body)
		if err != nil {
			return err
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())
		client := retryablehttp.NewClient()
		resp, err := client.Do(req)
		if err != nil {
			return err
		} else if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("fail to upload data, stats: %d, data: %s", resp.StatusCode, string(data))
		}
	} else {
		req, err := retryablehttp.NewRequest(method, url, &buf)
		if err != nil {
			return err
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := retryablehttp.NewClient().Do(req)
		if err != nil {
			return err
		} else if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("fail to upload data, stats: %d, data: %s", resp.StatusCode, string(data))
		}
	}
	return nil
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
	jobContext.mu.Lock()
	jobContext.stream = stream
	jobContext.mu.Unlock()
	status := JobStatus{
		Id:        job.ExecId,
		CreatedAt: time.Now(),
		Progress:  0,
	}
	err = RunJob(ctx, job, &status)
	status.CompletedAt = time.Now()
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

	err = populateJobStatus(&status)
	if err != nil {
		// if fail to populate job status to server, we still need to notify clients
		status.Status = pb.EnumExecStatus_EES_ERROR
		status.Message = err.Error()
		notifyJobStatusToClients(&status)
	}
	jobContext.mu.RLock()
	_, err = jobContext.stream.CloseAndRecv()
	jobContext.mu.RUnlock()
	return err
}

func RunJob(ctx context.Context, job *pb.JobGetResponse, status *JobStatus) error {
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
		return err
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			utils.LogError(err)
		}
	}()

	if err := os.MkdirAll(filepath.Join(dir, "/data"), os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "/output"), os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "/source"), os.ModePerm); err != nil {
		return err
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
		return err
	}

	status.Status = pb.EnumExecStatus_EES_PROCESSING_INPUTS
	if err = processInputs(localDataDir, job, status); err != nil {
		return err
	}

	status.Status = pb.EnumExecStatus_EES_RUNNING
	populateJobStatus(status)

	binds := []string{
		fmt.Sprintf("%s:%s", filepath.Join(hostDir, "/data"), job.Volume.Data),
		fmt.Sprintf("%s:%s", filepath.Join(hostDir, "/source"), job.Volume.Source),
		fmt.Sprintf("%s:%s", filepath.Join(hostDir, "/output"), job.Volume.Output),
	}

	containerName := fmt.Sprintf("sath-%s", job.ExecId)

	// create container
	containerId, err := CreateContainer(
		ctx, g.dockerClient, job.Cmd, job.Image.Url,
		job.GpuOpts, containerName, binds)
	if err != nil {
		return err
	}
	status.ContainerId = containerId
	if err := ExecImage(ctx, g.dockerClient, containerId, func(line string) {
		status.Status = pb.EnumExecStatus_EES_RUNNING
		status.Message = line
		populateJobStatus(status)
	}); err != nil {
		if stderr, ok := err.(StdErr); ok {
			status.Status = pb.EnumExecStatus_EES_ERROR
			status.Message = stderr.Error()
			populateJobStatus(status)
		} else {
			return err
		}
	}
	status.Status = pb.EnumExecStatus_EES_PROCESSING_OUPUTS
	populateJobStatus(status)

	if err := processOutputs(dir, job); err != nil {
		return err
	}

	return nil
}

func populateJobStatus(status *JobStatus) error {
	err := notifyJobStatusToServer(status, 0, 3)
	notifyJobStatusToClients(status)
	return errors.WithStack(err)
}

func notifyJobStatusToServer(status *JobStatus, retry int, maxRetry int) error {
	status.UpdatedAt = time.Now()
	jobContext.mu.Lock()
	st := status.Status
	if status.Paused && !jobContext.status.Paused {
		st = pb.EnumExecStatus_EES_PAUSED
	}
	if err := jobContext.stream.Send(&pb.ExecNotificationRequest{
		Status:   st,
		Message:  status.Message,
		Progress: float32(status.Progress),
	}); errors.Is(err, io.EOF) {
		if retry >= maxRetry {
			jobContext.mu.Unlock()
			return errors.Wrap(err, "max retry exceeded")
		}
		// server may ternimate stream connection after a configured idle timeout
		// in this case, reconnect and try it again
		stream, err := g.grpcClient.NotifyExecStatus(jobContext.stream.Context())
		if err != nil {
			return err
		}
		jobContext.stream.CloseSend()
		jobContext.stream = stream
		jobContext.mu.Unlock()
		return notifyJobStatusToServer(status, retry+1, maxRetry)
	} else if err != nil {
		err = errors.WithStack(err)
		jobContext.mu.Unlock()
		return err
	}
	jobContext.mu.Unlock()
	return nil
}

func notifyJobStatusToClients(status *JobStatus) error {
	<-jobContext.pauseChannel
	jobContext.mu.Lock()
	defer jobContext.mu.Unlock()
	jobContext.status = status
	for _, c := range jobContext.statusSubscribers {
		c <- *status
	}
	status.Message = ""
	if !jobContext.status.Paused {
		select {
		case jobContext.pauseChannel <- false:
		default:
		}
	}
	return nil
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

	if jobContext.status == nil {
		return nil
	} else {
		var status JobStatus = *jobContext.status
		return &status
	}
}

func Pause() bool {
	var status JobStatus
	jobContext.mu.Lock()

	if jobContext.status == nil {
		jobContext.mu.Unlock()
		return false
	}
	if !jobContext.status.Paused && jobContext.status.Status == pb.EnumExecStatus_EES_RUNNING && len(jobContext.status.ContainerId) > 0 {
		if err := g.dockerClient.ContainerPause(context.TODO(), jobContext.status.ContainerId); err != nil {
			utils.LogError(err)
		}
	}
	select {
	case <-jobContext.pauseChannel:
	default:
	}
	status = *jobContext.status
	status.Paused = true
	jobContext.status.Paused = true
	jobContext.mu.Unlock()
	populateJobStatus(&status)
	return true
}

func Resume() bool {
	var status JobStatus
	jobContext.mu.Lock()
	if jobContext.status == nil {
		jobContext.mu.Unlock()
		return false
	}
	if jobContext.status.Paused && jobContext.status.Status == pb.EnumExecStatus_EES_RUNNING && len(jobContext.status.ContainerId) > 0 {
		if err := g.dockerClient.ContainerUnpause(context.TODO(), jobContext.status.ContainerId); err != nil {
			utils.LogError(err)
		}
	}
	select {
	case jobContext.pauseChannel <- false:
	default:
	}
	status = *jobContext.status
	status.Paused = false
	jobContext.mu.Unlock()
	populateJobStatus(&status)
	return true
}
