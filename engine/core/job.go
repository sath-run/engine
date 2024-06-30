package core

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/gammazero/deque"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	pb "github.com/sath-run/engine/engine/core/protobuf"
	"github.com/sath-run/engine/engine/logger"
)

var (
	ErrUnautherized = errors.New("Unautherized")
	ErrNoJob        = errors.New("No job")
)

type JobSchedulerStatus int

const (
	JobSchedulerStatusPaused   JobSchedulerStatus = 0
	JobSchedulerStatusRunning  JobSchedulerStatus = 1
	JobSchedulerStatusStopping JobSchedulerStatus = 2
	JobSchedulerStatusStopped  JobSchedulerStatus = 3
)

type JobSchedulerAction int

const (
	JobSchedulerActionStart JobSchedulerAction = 1
	JobSchedulerActionPause JobSchedulerAction = 2
)

type JobContainer struct {
	imageUrl    string
	containerId string
	dir         string

	jobQueue   deque.Deque[*Job]
	currentJob *Job
}

type JobImage struct {
	Url  string
	Auth string
}

type JobVolume struct {
	Data   string
	Source string
	Output string
}

type Job struct {
	ExecId      string
	ContainerId string
	Status      pb.EnumExecStatus
	Progress    float64
	Message     string
	Paused      bool
	CreatedAt   time.Time
	CompletedAt time.Time
	UpdatedAt   time.Time
	Res         *pb.JobGetResponse
	Outputs     []*pb.ExecOutput

	dir       string
	mu        sync.RWMutex
	stream    pb.Engine_NotifyExecStatusClient
	err       error
	container *JobContainer
	scheduler *JobScheduler
}

type JobScheduler struct {
	user        *User
	grpc        pb.EngineClient
	docker      *client.Client
	dir         string
	status      JobSchedulerStatus
	jobChan     chan *Job
	actionChan  chan JobSchedulerAction
	containers  []*JobContainer
	pendingJobs map[*Job]bool
}

func NewJobScheduler(user *User, grpc pb.EngineClient, docker *client.Client, dir string) *JobScheduler {
	scheduler := JobScheduler{
		user:       user,
		grpc:       grpc,
		docker:     docker,
		dir:        dir,
		status:     JobSchedulerStatusPaused,
		jobChan:    make(chan *Job, 8),
		actionChan: make(chan JobSchedulerAction),
		containers: []*JobContainer{},
	}
	go scheduler.run()
	return &scheduler
}

func (scheduler *JobScheduler) run() {
	ticker := time.NewTicker(time.Second * 30)
	for {
		select {
		case action := <-scheduler.actionChan:
			switch action {
			case JobSchedulerActionStart:
				scheduler.status = JobSchedulerStatusRunning
			case JobSchedulerActionPause:
				scheduler.status = JobSchedulerStatusPaused
			default:
				panic(fmt.Sprintf("unexpected action %d\n", action))
			}
		case job := <-scheduler.jobChan:
			if job.container == nil {
				// if container is nil, it means the job has just finished processing inputs
				delete(scheduler.pendingJobs, job)
				if job.Status == pb.EnumExecStatus_EES_ERROR {
					return
				}
				// try to find container for image url from existing running ones
				var container *JobContainer
				for _, c := range scheduler.containers {
					if c.imageUrl == job.Res.Image.Url {
						container = c
						break
					}
				}
				if container == nil {
					container = &JobContainer{
						imageUrl: job.Res.Image.Url,
						jobQueue: deque.Deque[*Job]{},
						dir:      filepath.Join(job.scheduler.dir, "container_"+job.ExecId),
					}
					scheduler.containers = append(scheduler.containers, container)
				}
				job.container = container
				if container.currentJob == nil {
					container.currentJob = job
				} else {
					container.jobQueue.PushBack(job)
					return
				}
			}

			// assersions
			if job.container == nil {
				panic(spew.Sprintln("job container is nil", job))
			}
			if job != job.container.currentJob {
				panic(spew.Sprintln("job is not current", job, job.container.currentJob))
			}

			container := job.container

			switch job.Status {
			case pb.EnumExecStatus_EES_WAITING:
				go job.run()
			case pb.EnumExecStatus_EES_ERROR, pb.EnumExecStatus_EES_CANCELED, pb.EnumExecStatus_EES_SUCCESS:
				// job is finished, scheduler the next job
				if container.jobQueue.Len() > 0 {
					nextJob := container.jobQueue.PopFront()
					container.currentJob = nextJob
					scheduler.jobChan <- nextJob
				} else {
					container.currentJob = nil
					// TODO: cleanup container ?
				}
				scheduler.fetchNewJob()
			default:
				panic(spew.Sprintln("unexpected job status", job))
			}
		case <-ticker.C:
			scheduler.fetchNewJob()
		}
	}
}

func (scheduler *JobScheduler) Start() {
	scheduler.actionChan <- JobSchedulerActionStart
}

func (scheduler *JobScheduler) Pause() {
	scheduler.actionChan <- JobSchedulerActionPause
}

func (scheduler *JobScheduler) fetchNewJob() {
	if scheduler.status != JobSchedulerStatusRunning {
		return
	}
	ctx := scheduler.user.ContextWithToken(context.TODO())
	res, err := scheduler.grpc.GetNewJob(ctx, &pb.JobGetRequest{})
	if err != nil {
		logger.Error(err)
		return
	}
	if res == nil || len(res.ExecId) == 0 {
		// no available jobs from server
		return
	}
	stream, err := scheduler.grpc.NotifyExecStatus(ctx)
	if err != nil {
		logger.Error(err)
		return
	}
	job := Job{
		ExecId:    res.ExecId,
		Res:       res,
		Status:    pb.EnumExecStatus_EES_DEFAULT,
		CreatedAt: time.Now(),
		stream:    stream,
		scheduler: scheduler,
		dir:       filepath.Join(scheduler.dir, "job_"+res.ExecId),
	}
	scheduler.pendingJobs[&job] = true

	// process job inputs, when it finishes, it will signal scheduler
	go job.processInputs()
}

func (job *Job) updateStatus(status pb.EnumExecStatus, message string, progress float64) {
	if job.err != nil && status != pb.EnumExecStatus_EES_ERROR {
		return
	}

	job.Status = status
	job.Message = message
	job.Progress = progress
	job.UpdatedAt = time.Now()

	// notify status to server
	req := &pb.ExecNotificationRequest{
		Status:   job.Status,
		Message:  job.Message,
		Progress: job.Progress,
	}
	if status == pb.EnumExecStatus_EES_SUCCESS {
		req.Outputs = job.Outputs
	}
	// if status.GpuOpts != "" {
	// 	req.GpuStats = []*pb.GpuStats{
	// 		{Id: 0},
	// 	}
	// }
	var err error
	if err = job.stream.Send(req); errors.Is(err, io.EOF) {
		// server may ternimate stream connection after a configured idle timeout
		// in this case, reconnect and try it again
		ctx := job.scheduler.user.ContextWithToken(context.TODO())
		stream, err2 := job.scheduler.grpc.NotifyExecStatus(ctx)
		if err2 != nil {
			job.handleError(err2)
			return
		} else {
			job.stream.CloseSend()
			job.stream = stream
			err = job.stream.Send(req)
		}
	}
	if err != nil {
		job.handleError(err)
	}
}

func (job *Job) handleError(err error) {
	if job.err != nil {
		return
	}
	job.err = err
	job.updateStatus(pb.EnumExecStatus_EES_ERROR, err.Error(), 0)
}

func (job *Job) notifyScheduler() {
	job.scheduler.jobChan <- job
}

func (job *Job) processInputs() {
	defer job.notifyScheduler()
	job.mu.Lock()
	defer job.mu.Unlock()

	files := job.Res.Inputs
	for i, file := range files {
		job.updateStatus(
			pb.EnumExecStatus_EES_DOWNLOADING_INPUTS,
			fmt.Sprintf("start download %s", file.Path),
			float64(i+1)/float64(len(files)),
		)
		if job.err != nil {
			break
		}
		filePath := filepath.Join(job.dir, file.Path)
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

			_, err = io.Copy(out, resp.Body)
			if err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			job.handleError(err)
			break
		}
	}
	if job.err == nil {
		job.updateStatus(pb.EnumExecStatus_EES_WAITING, "", 0)
	}
}

func (job *Job) run() {
	defer job.notifyScheduler()
	job.mu.Lock()
	defer job.mu.Unlock()

	ctn := job.container
	scheduler := job.scheduler
	ctx := scheduler.user.ContextWithToken(context.TODO())
	if len(ctn.containerId) == 0 {
		// prepare and start docker container
		dir := ctn.dir
		if err := os.MkdirAll(filepath.Join(dir, "/data"), os.ModePerm); err != nil {
			job.handleError(err)
			return
		}
		if err := os.MkdirAll(filepath.Join(dir, "/output"), os.ModePerm); err != nil {
			job.handleError(err)
			return
		}
		if err := os.MkdirAll(filepath.Join(dir, "/source"), os.ModePerm); err != nil {
			job.handleError(err)
			return
		}

		if err := PullImage(ctx, scheduler.docker, job.Res.Image.Url, image.PullOptions{
			RegistryAuth: job.Res.Image.Auth,
		}, func(text string) {
			job.updateStatus(pb.EnumExecStatus_EES_PULLING_IMAGE, text, 0)
			if job.err != nil {
				return
			}
		}); err != nil {
			job.handleError(err)
			return
		}

		job.updateStatus(pb.EnumExecStatus_EES_INIT_CONTAINER, "", 0)
		if job.err != nil {
			return
		}

		// create container
		binds := []string{
			fmt.Sprintf("%s:%s", filepath.Join(dir, "/data"), job.Res.Volume.Data),
			fmt.Sprintf("%s:%s", filepath.Join(dir, "/source"), job.Res.Volume.Source),
			fmt.Sprintf("%s:%s", filepath.Join(dir, "/output"), job.Res.Volume.Output),
		}

		containerName := fmt.Sprintf("sath-%s", job.ExecId)

		// TODO: better strategy for GPU allocation
		gpuOpts := ""
		if job.Res.GpuConf.Opt == pb.GpuOpt_EGO_REQUIRED || job.Res.GpuConf.Opt == pb.GpuOpt_EGO_PREFERRED {
			gpuOpts = "all"
		}
		containerId, err := CreateContainer(
			ctx, scheduler.docker, nil, job.Res.Image.Url,
			gpuOpts, containerName, binds)
		if err != nil {
			job.handleError(err)
			return
		}
		ctn.containerId = containerId
		err = scheduler.docker.ContainerStart(ctx, containerId, container.StartOptions{})
		if err != nil {
			job.handleError(err)
			return
		}
	}

	// move input files to volume
	if err := exec.Command("rm", "-r", filepath.Join(ctn.dir, "/data", "*")).Run(); err != nil {
		job.handleError(err)
		return
	}
	if err := exec.Command("rm", "-r", filepath.Join(ctn.dir, "/output", "*")).Run(); err != nil {
		job.handleError(err)
		return
	}
	if err := exec.Command("mv", "-r", filepath.Join(job.dir, "*"), filepath.Join(ctn.dir, "/data")).Run(); err != nil {
		job.handleError(err)
		return
	}
	os.Remove(job.dir)

	// start job exec
	res, err := scheduler.docker.ContainerExecCreate(ctx, ctn.containerId, container.ExecOptions{
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          job.Res.Cmd,
	})
	if err != nil {
		job.handleError(err)
		return
	}

	hijack, err := scheduler.docker.ContainerExecAttach(ctx, res.ID, container.ExecStartOptions{
		Tty: true,
	})
	if err != nil {
		job.handleError(err)
		return
	}
	defer hijack.Close()

	scanner := bufio.NewScanner(hijack.Reader)
	for scanner.Scan() {
		// TODO: update progress
		job.updateStatus(pb.EnumExecStatus_EES_RUNNING, scanner.Text(), 0)
		if job.err != nil {
			return
		}
	}

	job.processOutputs(filepath.Join(ctn.dir, "/output", "*"))
	if job.err != nil {
		return
	}

	job.updateStatus(pb.EnumExecStatus_EES_SUCCESS, "", 100)
	if job.err != nil {
		return
	}
}

func JobStatusText(enum pb.EnumExecStatus) string {
	switch enum {
	case pb.EnumExecStatus_EES_DEFAULT:
		return "default"
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

func (job *Job) processOutputs(outputDir string) {
	outputs := make([]*pb.ExecOutput, len(job.Res.Outputs))
	job.Outputs = outputs
	for i, output := range job.Res.Outputs {
		job.updateStatus(pb.EnumExecStatus_EES_PROCESSING_OUPUTS, "", float64(i)/float64(len(job.Res.Outputs)))
		if job.err != nil {
			return
		}
		outputs[i] = &pb.ExecOutput{
			Id:     output.Id,
			Status: pb.ExecOutputStatus_EOS_SUCCESS,
		}
		path := filepath.Join(outputDir, output.Path)
		data, err := os.Open(path)
		if err != nil {
			logger.Debug("file not found:", path)
			continue
		}
		defer data.Close()

		if output.Req == nil {
			// if output request is not specified, return file content
			fs, err := data.Stat()
			if err != nil {
				job.handleError(err)
				return
			}
			fileSizeInMB := float64(fs.Size()) / 1024 / 1024
			if fileSizeInMB > 1 {
				err = fmt.Errorf(
					"file %s is too large, size limit is 1MB, actual size is %.2fMB",
					output.Path,
					fileSizeInMB)
				job.handleError(err)
				return
			}
			bytes, err := io.ReadAll(data)
			if err != nil {
				job.handleError(err)
				return
			}
			outputs[i].Content = bytes
		} else {
			// upload output file to url
			req, err := retryablehttp.NewRequest(output.Req.Method, output.Req.Url, data)
			if err != nil {
				job.handleError(err)
				return
			}
			for _, header := range output.Req.Headers {
				req.Header.Set(header.Name, header.Value)
			}
			resp, err := retryablehttp.NewClient().Do(req)
			if err != nil {
				job.handleError(err)
				return
			} else if resp.StatusCode < 200 || resp.StatusCode >= 400 {
				data, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				err = fmt.Errorf("fail to upload data, stats: %d, data: %s", resp.StatusCode, string(data))
				job.handleError(err)
				return
			}
		}
	}
}

// func (jobContext *JobContext) Run(ctx context.Context, grpcClient pb.EngineClient) error {
// 	job, err := grpcClient.GetNewJob(ctx, &pb.JobGetRequest{})

// 	if err != nil {
// 		return err
// 	}

// 	if job == nil || len(job.ExecId) == 0 {
// 		return ErrNoJob
// 	}
// 	ctx = metadata.AppendToOutgoingContext(ctx, "exec_id", job.ExecId)
// 	stream, err := grpcClient.NotifyExecStatus(ctx)
// 	if err != nil {
// 		return err
// 	}
// 	status := JobStatus{
// 		Id:        job.ExecId,
// 		CreatedAt: time.Now(),
// 		Progress:  0,
// 		GpuOpts:   job.GpuOpt.String(),
// 	}
// 	outputs, err := RunJob(ctx, job, status)
// 	status.CompletedAt = time.Now()
// 	status.Outputs = outputs
// 	if err != nil {
// 		logger.Error(err)
// 		if errors.Is(err, context.Canceled) {
// 			status.Status = pb.EnumExecStatus_EES_CANCELED
// 			status.Message = "user canceled"
// 		} else {
// 			status.Status = pb.EnumExecStatus_EES_ERROR
// 			status.Message = fmt.Sprintf("%+v", err)
// 		}
// 	} else {
// 		status.Progress = 100
// 		status.Status = pb.EnumExecStatus_EES_SUCCESS
// 	}

// 	err = populateJobStatus(status)
// 	if err != nil {
// 		// if fail to populate job status to server, we still need to notify clients
// 		status.Status = pb.EnumExecStatus_EES_ERROR
// 		status.Message = err.Error()
// 		notifyJobStatusToClients(status)
// 	}
// 	jobContext.RLock()
// 	_, err = jobContext.stream.CloseAndRecv()
// 	jobContext.RUnLock()
// 	return err
// }

// func RunJob(ctx context.Context, job *pb.JobGetResponse, status JobStatus) ([]*pb.ExecOutput, error) {
// 	if ref, err := reference.ParseNormalizedNamed(job.Image.Url); err == nil {
// 		status.Image = ref.Name()
// 	} else {
// 		status.Image = job.Image.Url
// 	}

// 	status.Status = pb.EnumExecStatus_EES_STARTED
// 	populateJobStatus(status)

// 	logger.Debug("RunJob: ", job)

// 	dir, err := os.MkdirTemp(g.localDataDir, "sath_tmp_*")
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer func() {
// 		if err := os.RemoveAll(dir); err != nil {
// 			logger.Error(err)
// 		}
// 	}()

// 	if err := os.MkdirAll(filepath.Join(dir, "/data"), os.ModePerm); err != nil {
// 		return nil, err
// 	}
// 	if err := os.MkdirAll(filepath.Join(dir, "/output"), os.ModePerm); err != nil {
// 		return nil, err
// 	}
// 	if err := os.MkdirAll(filepath.Join(dir, "/source"), os.ModePerm); err != nil {
// 		return nil, err
// 	}

// 	localDataDir := dir
// 	hostDir := localDataDir
// 	if len(g.hostDataDir) > 0 {
// 		tmpDirName := filepath.Base(dir)
// 		hostDir = filepath.Join(g.hostDataDir, tmpDirName)
// 	}

// 	if err = PullImage(ctx, g.dockerClient, job.Image.Url, image.PullOptions{
// 		RegistryAuth: job.Image.Auth,
// 	}, func(text string) {
// 		status.Status = pb.EnumExecStatus_EES_PULLING_IMAGE
// 		status.Message = text
// 		populateJobStatus(status)
// 	}); err != nil {
// 		return nil, err
// 	}

// 	status.Status = pb.EnumExecStatus_EES_PROCESSING_INPUTS
// 	if err = processInputs(localDataDir, job, status); err != nil {
// 		return nil, err
// 	}

// 	status.Status = pb.EnumExecStatus_EES_RUNNING
// 	populateJobStatus(status)

// 	if len(job.Volume.Data) == 0 {
// 		job.Volume.Data = "/data"
// 	}
// 	if len(job.Volume.Source) == 0 {
// 		job.Volume.Source = "/source"
// 	}
// 	if len(job.Volume.Output) == 0 {
// 		job.Volume.Output = "/output"
// 	}

// 	binds := []string{
// 		fmt.Sprintf("%s:%s", filepath.Join(hostDir, "/data"), job.Volume.Data),
// 		fmt.Sprintf("%s:%s", filepath.Join(hostDir, "/source"), job.Volume.Source),
// 		fmt.Sprintf("%s:%s", filepath.Join(hostDir, "/output"), job.Volume.Output),
// 	}

// 	containerName := fmt.Sprintf("sath-%s", job.ExecId)

// 	gpuOpts := ""
// 	if job.GpuOpt == pb.GpuOpt_EGO_REQUIRED || job.GpuOpt == pb.GpuOpt_EGO_PREFERRED {
// 		gpuOpts = "all"
// 	}

// 	// create container
// 	containerId, err := CreateContainer(
// 		ctx, g.dockerClient, job.Cmd, job.Image.Url,
// 		gpuOpts, containerName, binds)
// 	if err != nil {
// 		return nil, err
// 	}
// 	status.ContainerId = containerId
// 	if err := ExecImage(ctx, g.dockerClient, containerId, func(line string) {
// 		status.Status = pb.EnumExecStatus_EES_RUNNING
// 		status.Message = line
// 		populateJobStatus(status)
// 	}); err != nil {
// 		return nil, err
// 	}
// 	status.Status = pb.EnumExecStatus_EES_PROCESSING_OUPUTS
// 	populateJobStatus(status)

// 	outputs, err := processOutputs(dir, job)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return outputs, nil
// }

// func populateJobStatus(status JobStatus) error {
// 	err := notifyJobStatusToServer(status, 0, 3)
// 	notifyJobStatusToClients(status)
// 	return errors.WithStack(err)
// }

// func notifyJobStatusToServer(status JobStatus, retry int, maxRetry int) error {
// 	status.UpdatedAt = time.Now()
// 	jobContext.Lock()
// 	st := status.Status
// 	if status.Paused && !jobContext.status.Paused {
// 		st = pb.EnumExecStatus_EES_PAUSED
// 	}
// 	req := &pb.ExecNotificationRequest{
// 		Status:   st,
// 		Message:  status.Message,
// 		Progress: float32(status.Progress),
// 		Outputs:  status.Outputs,
// 	}
// 	if status.GpuOpts != "" {
// 		req.GpuStats = []*pb.GpuStats{
// 			{Id: 0},
// 		}
// 	}
// 	if err := jobContext.stream.Send(req); errors.Is(err, io.EOF) {
// 		if retry >= maxRetry {
// 			jobContext.UnLock()
// 			return errors.Wrap(err, "max retry exceeded")
// 		}
// 		// server may ternimate stream connection after a configured idle timeout
// 		// in this case, reconnect and try it again
// 		stream, err := g.grpcClient.NotifyExecStatus(jobContext.stream.Context())
// 		if err != nil {
// 			jobContext.UnLock()
// 			return err
// 		}
// 		jobContext.stream.CloseSend()
// 		jobContext.stream = stream
// 		jobContext.UnLock()
// 		return notifyJobStatusToServer(status, retry+1, maxRetry)
// 	} else if err != nil {
// 		err = errors.WithStack(err)
// 		jobContext.UnLock()
// 		return err
// 	}
// 	jobContext.UnLock()
// 	return nil
// }

// func notifyJobStatusToClients(status JobStatus) error {
// 	jobContext.Pause()
// 	jobContext.Lock()
// 	defer jobContext.UnLock()
// 	for _, c := range jobContext.statusSubscribers {
// 		c <- status
// 	}
// 	status.Message = ""
// 	jobContext.status = status
// 	if !status.Paused {
// 		jobContext.Resume()
// 	}
// 	return nil
// }

// func SubscribeJobStatus(channel chan JobStatus) {
// 	jobContext.Lock()
// 	defer jobContext.UnLock()

// 	jobContext.statusSubscribers = append(jobContext.statusSubscribers, channel)
// }

// func UnsubscribeJobStatus(channel chan JobStatus) {
// 	jobContext.Lock()
// 	defer jobContext.UnLock()

// 	subscribers := make([]chan JobStatus, 0)
// 	for _, c := range jobContext.statusSubscribers {
// 		if c != channel {
// 			subscribers = append(subscribers, c)
// 		}
// 	}
// 	jobContext.statusSubscribers = subscribers
// 	close(channel)
// }

// func GetJobStatus() *JobStatus {
// 	jobContext.RLock()
// 	defer jobContext.RUnLock()

// 	if jobContext.status.IsNil() {
// 		return nil
// 	} else {
// 		var status JobStatus = jobContext.status
// 		return &status
// 	}
// }

// func Pause(execId string) bool {
// 	var status JobStatus
// 	jobContext.Lock()

// 	if jobContext.status.IsNil() {
// 		jobContext.UnLock()
// 		return false
// 	}
// 	if len(execId) > 0 && jobContext.status.Id != execId {
// 		jobContext.UnLock()
// 		return false
// 	}
// 	if !jobContext.status.Paused && jobContext.status.Status == pb.EnumExecStatus_EES_RUNNING && len(jobContext.status.ContainerId) > 0 {
// 		if err := g.dockerClient.ContainerPause(context.TODO(), jobContext.status.ContainerId); err != nil {
// 			logger.Error(err)
// 		}
// 	}

// 	status = jobContext.status
// 	status.Paused = true
// 	jobContext.UnLock()
// 	populateJobStatus(status)
// 	return true
// }

// func Resume(execId string) bool {
// 	var status JobStatus
// 	jobContext.Lock()
// 	if jobContext.status.IsNil() {
// 		jobContext.UnLock()
// 		return false
// 	}
// 	if len(execId) > 0 && jobContext.status.Id != execId {
// 		jobContext.UnLock()
// 		return false
// 	}
// 	if jobContext.status.Paused &&
// 		jobContext.status.Status == pb.EnumExecStatus_EES_RUNNING &&
// 		len(jobContext.status.ContainerId) > 0 {
// 		if err := g.dockerClient.ContainerUnpause(context.TODO(), jobContext.status.ContainerId); err != nil {
// 			logger.Error(err)
// 		}
// 	}
// 	jobContext.Resume()
// 	status = jobContext.status
// 	status.Paused = false
// 	jobContext.UnLock()
// 	populateJobStatus(status)
// 	return true
// }
