package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	"github.com/sath-run/engine/cmd/utils"
	pb "github.com/sath-run/engine/pkg/protobuf"
	"google.golang.org/grpc/metadata"
)

var (
	ErrNoJob = errors.New("No job")
)

func TaskStatusText(enum pb.EnumExecStatus) string {
	switch enum {
	case pb.EnumExecStatus_READY:
		return "ready"
	case pb.EnumExecStatus_PULLING_IMAGE:
		return "pulling-image"
	case pb.EnumExecStatus_DOWNLOADING_INPUTS:
		return "downloading"
	case pb.EnumExecStatus_PROCESSING_INPUTS:
		return "preprocessing"
	case pb.EnumExecStatus_RUNNING:
		return "running"
	case pb.EnumExecStatus_PROCESSING_OUPUTS:
		return "postprocessing"
	case pb.EnumExecStatus_UPLOADING_OUTPUTS:
		return "uploading"
	case pb.EnumExecStatus_SUCCESS:
		return "success"
	case pb.EnumExecStatus_CANCELLED:
		return "cancelled"
	case pb.EnumExecStatus_ERROR:
		return "error"
	default:
		return "unspecified"
	}
}

var taskContext = struct {
	mu                sync.RWMutex
	status            *TaskStatus
	statusSubscribers []chan TaskStatus
}{}

type TaskStatus struct {
	Id          string
	ImageUrl    string
	ContainerId string
	Status      pb.EnumExecStatus
	Progress    float64
	Message     string
	CreatedAt   time.Time
	CompletedAt time.Time
	UpdatedAt   time.Time
	stream      pb.Engine_NotifyExecStatusClient
}

func processInputs(dir string, task *pb.TaskGetResponse) error {
	files := task.GetInputs()
	dataDir := filepath.Join(dir, "/data")
	for _, file := range files {
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

func processOutputs(dir string, task *pb.TaskGetResponse) error {
	output := task.GetOutput()
	if output == nil {
		return errors.New("task output is nil")
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

	url := task.Output.Url
	headers := task.Output.Headers
	data := task.Output.Data

	if headers["Accept"] == "application/json" {
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
		req.Header.Set("Content-Type", "application/json")
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
		}
		if err := json.Unmarshal(body, &obj); err != nil {
			return err
		}
		url = obj.Url
		headers = obj.Headers
		method = obj.Method
	}

	req, err := retryablehttp.NewRequest(method, url, &buf)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if _, err := retryablehttp.NewClient().Do(req); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func RunSingleJob(ctx context.Context) error {
	task, err := g.grpcClient.GetNewTask(ctx, &pb.TaskGetRequest{
		Version: VERSION,
	})

	if err != nil {
		return err
	}

	if task == nil || len(task.ExecId) == 0 {
		return ErrNoJob
	}
	c := g.ContextWithToken(context.TODO())
	metadata.AppendToOutgoingContext(c,
		"execId", task.ExecId)
	stream, err := g.grpcClient.NotifyExecStatus(c)
	if err != nil {
		return err
	}
	status := TaskStatus{
		Id:        task.ExecId,
		CreatedAt: time.Now(),
		Progress:  0,
		stream:    stream,
	}
	err = RunTask(ctx, task, &status)
	status.CompletedAt = time.Now()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			status.Status = pb.EnumExecStatus_CANCELLED
			status.Message = "user cancelled"
		} else {
			status.Status = pb.EnumExecStatus_ERROR
			status.Message = fmt.Sprintf("%+v", err)
		}
	} else {
		status.Progress = 100
		status.Status = pb.EnumExecStatus_SUCCESS
	}

	populateTaskStatus(&status)
	_, err = stream.CloseAndRecv()
	return err
}

func RunTask(ctx context.Context, task *pb.TaskGetResponse, status *TaskStatus) error {
	status.ImageUrl = task.ImageUrl
	status.Status = pb.EnumExecStatus_READY
	populateTaskStatus(status)

	dir, err := os.MkdirTemp(g.localDataDir, "sath_tmp_*")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("%+v\n", err)
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

	if err = PullImage(ctx, g.dockerClient, task.ImageUrl, func(text string) {
		status.Status = pb.EnumExecStatus_PULLING_IMAGE
		status.Message = text
		populateTaskStatus(status)
	}); err != nil {
		return err
	}

	status.Status = pb.EnumExecStatus_PROCESSING_INPUTS
	populateTaskStatus(status)

	if err = processInputs(localDataDir, task); err != nil {
		return err
	}

	status.Status = pb.EnumExecStatus_RUNNING
	populateTaskStatus(status)

	binds := []string{
		fmt.Sprintf("%s:%s", filepath.Join(hostDir, "/data"), task.Volume.Data),
		fmt.Sprintf("%s:%s", filepath.Join(hostDir, "/source"), task.Volume.Source),
		fmt.Sprintf("%s:%s", filepath.Join(hostDir, "/output"), task.Volume.Output),
	}

	containerName := fmt.Sprintf("sath-%s", task.ExecId)

	// create container
	containerId, err := CreateContainer(
		ctx, g.dockerClient, task.Cmd, task.ImageUrl,
		task.GpuOpts, containerName, binds)
	if err != nil {
		return err
	}
	if err := ExecImage(ctx, g.dockerClient, containerId, func(line string) {
		status.Status = pb.EnumExecStatus_RUNNING
		status.Message = line
		populateTaskStatus(status)
	}); err != nil {
		return err
	}
	status.Status = pb.EnumExecStatus_PROCESSING_OUPUTS
	populateTaskStatus(status)

	if err := processOutputs(dir, task); err != nil {
		return err
	}

	return nil
}

func populateTaskStatus(status *TaskStatus) error {
	status.UpdatedAt = time.Now()
	if err := status.stream.Send(&pb.ExecNotificationRequest{
		Status:   status.Status,
		Message:  status.Message,
		Progress: int32(status.Progress),
	}); err != nil {
		utils.LogDebug("populateTaskStatus", err)
		return err
	}

	taskContext.mu.Lock()
	defer taskContext.mu.Unlock()
	taskContext.status = status

	for _, c := range taskContext.statusSubscribers {
		c <- *status
	}
	status.Message = ""
	return nil
}

func SubscribeTaskStatus(channel chan TaskStatus) {
	taskContext.mu.Lock()
	defer taskContext.mu.Unlock()

	taskContext.statusSubscribers = append(taskContext.statusSubscribers, channel)
}

func UnsubscribeTaskStatus(channel chan TaskStatus) {
	taskContext.mu.Lock()
	defer taskContext.mu.Unlock()

	subscribers := make([]chan TaskStatus, 0)
	for _, c := range taskContext.statusSubscribers {
		if c != channel {
			subscribers = append(subscribers, c)
		}
	}
	taskContext.statusSubscribers = subscribers
	close(channel)
}

func GetTaskStatus() *TaskStatus {
	taskContext.mu.RLock()
	defer taskContext.mu.RUnlock()

	if taskContext.status == nil {
		return nil
	} else {
		var status TaskStatus = *taskContext.status
		return &status
	}
}
