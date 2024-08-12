package scheduler

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/sath-run/engine/engine/core/conns"
	pb "github.com/sath-run/engine/engine/core/protobuf"
)

var (
	ErrUnautherized = errors.New("unautherized")
	ErrNoJob        = errors.New("no job")
	ErrActionBusy   = errors.New("action busy")
)

type Status int

const (
	StatusPaused Status = iota
	StatusRunning
	StatusPausing
)

type Action int

const (
	ActionStart Action = iota
	ActionPause
)

type Image struct {
	Url  string
	Auth string
}

type Volume struct {
	Data   string
	Source string
	Output string
}

type Scheduler struct {
	c           *conns.Connection
	docker      *client.Client
	dir         string
	status      Status
	closeChan   chan struct{}
	jobChan     chan *Job
	actionLock  sync.Mutex
	fetchLock   sync.Mutex
	pendingJobs map[*Job]bool

	// containers
	containers []*Container
}

func NewScheduler(ctx context.Context, c *conns.Connection, dir string, jobInterval time.Duration) (*Scheduler, error) {
	docker, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	if err := stopCurrentRunningContainers(ctx, docker); err != nil {
		return nil, err
	}
	scheduler := Scheduler{
		c:           c,
		docker:      docker,
		dir:         dir,
		status:      StatusPaused,
		jobChan:     make(chan *Job, 8),
		containers:  []*Container{},
		pendingJobs: map[*Job]bool{},
	}
	go scheduler.loop(jobInterval)
	return &scheduler, nil
}

func (scheduler *Scheduler) loop(jobInterval time.Duration) {
	ticker := time.NewTicker(jobInterval)
	for {
		select {
		case job := <-scheduler.jobChan:
			if job.state == pb.EnumExecState_EES_SUCCESS || job.err != nil {
				if job.err == nil {
					job.logger.Info().Msg("succeed")
				} else {
					job.logger.Info().Err(job.err).Str("state", job.state.String()).Send()
				}
				scheduler.rescheduleContainer(job.container)
				continue
			}
			switch job.state {
			case pb.EnumExecState_EES_INITIALIZED:
				go job.preprocess()
			case pb.EnumExecState_EES_QUEUING:
				if job.container == nil {
					scheduler.attachContainerForJob(job)
				}
				if job.container != nil {
					// if container successfully attached, run job
					go job.run()
				}
			default:
				job.err = errors.New("unexpected job state")
				log.Fatal().Str("job", job.metadata.ExecId).Str("state", job.state.String()).Err(job.err).Send()
				scheduler.jobChan <- job
			}
		case <-ticker.C:
			scheduler.fetchNewJob()
		case <-scheduler.closeChan:
			ticker.Stop()
			return
		}
	}
}

func (scheduler *Scheduler) performAction(action Action) error {
	if !scheduler.actionLock.TryLock() {
		return ErrActionBusy
	}
	defer scheduler.actionLock.Unlock()

	switch action {
	case ActionStart:
		scheduler.status = StatusRunning
	case ActionPause:
		scheduler.status = StatusPaused
	}

	return nil
}

func (scheduler *Scheduler) Start() {
	scheduler.performAction(ActionStart)
}

func (scheduler *Scheduler) Pause() {
	scheduler.performAction(ActionPause)
}

func (scheduler *Scheduler) fetchNewJob() {
	if scheduler.status != StatusRunning {
		return
	}
	for _, container := range scheduler.containers {
		// TODO: support multiple container
		if container.currentJob != nil {
			return
		}
	}
	baseCtx, hasUser := scheduler.c.AppendToOutgoingContext(context.Background())
	if !hasUser {
		return
	}
	go func() {
		// make sure only one goroutine of fetchNewJob is running
		if !scheduler.fetchLock.TryLock() {
			return
		}
		defer scheduler.fetchLock.Unlock()
		var (
			ctx    context.Context
			cancel context.CancelFunc
		)

		ctx, cancel = context.WithTimeout(baseCtx, 5*time.Second)
		defer cancel()
		res, err := scheduler.c.GetNewJob(ctx, &pb.JobGetRequest{})
		if err != nil {
			log.Warn().Err(err).Msg("scheduler fails to get a new job")
			return
		}
		if res == nil || len(res.ExecId) == 0 {
			// no available jobs from server
			log.Info().Msg("no available jobs from server")
			return
		}
		ctx, cancel = context.WithTimeout(baseCtx, 5*time.Second)
		defer cancel()
		job, err := newJob(ctx, scheduler.c, scheduler.docker, scheduler.jobChan, scheduler.dir, res)
		if err != nil {
			log.Warn().Err(err).Msg("scheduler fails to create job")
			return
		}
		log.Trace().Any("scheduler fetched new job", res).Send()
		scheduler.jobChan <- job
	}()
}

func (scheduler *Scheduler) attachContainerForJob(job *Job) bool {
	var container *Container

	// find idle container if any
	for _, c := range scheduler.containers {
		if c.imageUrl == job.metadata.Image.Url && c.currentJob == nil {
			container = c
			break
		}
	}
	if container == nil {
		// allocate new container
		// TODO: should check system resouces before deciding to create a new container

		// ignore mkdir error if any, it will be handled inside job.run
		dir, _ := os.MkdirTemp(scheduler.dir, "container_")

		container = &Container{
			imageUrl:   job.metadata.Image.Url,
			currentJob: job,
			dir:        dir,
		}
		scheduler.containers = append(scheduler.containers, container)

	} else {
		container.currentJob = job
	}
	if container == nil {
		// if no container found nor allocated a new one, enqueue job
		scheduler.pendingJobs[job] = true
		return false
	}
	job.container = container
	return true
}

func (scheduler *Scheduler) rescheduleContainer(container *Container) {
	container.currentJob = nil
	jobs := []*Job{}
	for job := range scheduler.pendingJobs {
		scheduler.attachContainerForJob(job)
		if job.container != nil {
			// if successfully attached
			jobs = append(jobs, job)
		}
	}
	for _, job := range jobs {
		delete(scheduler.pendingJobs, job)
		scheduler.jobChan <- job
	}
	scheduler.fetchNewJob()
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
// 			status.Status = pb.EnumExecState_EES_CANCELED
// 			status.Message = "user canceled"
// 		} else {
// 			status.Status = pb.EnumExecState_EES_ERROR
// 			status.Message = fmt.Sprintf("%+v", err)
// 		}
// 	} else {
// 		status.Progress = 100
// 		status.Status = pb.EnumExecState_EES_SUCCESS
// 	}

// 	err = populateJobStatus(status)
// 	if err != nil {
// 		// if fail to populate job status to server, we still need to notify clients
// 		status.Status = pb.EnumExecState_EES_ERROR
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

// 	status.Status = pb.EnumExecState_EES_STARTED
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
// 		status.Status = pb.EnumExecState_EES_PULLING_IMAGE
// 		status.Message = text
// 		populateJobStatus(status)
// 	}); err != nil {
// 		return nil, err
// 	}

// 	status.Status = pb.EnumExecState_EES_PROCESSING_INPUTS
// 	if err = processInputs(localDataDir, job, status); err != nil {
// 		return nil, err
// 	}

// 	status.Status = pb.EnumExecState_EES_RUNNING
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
// 		status.Status = pb.EnumExecState_EES_RUNNING
// 		status.Message = line
// 		populateJobStatus(status)
// 	}); err != nil {
// 		return nil, err
// 	}
// 	status.Status = pb.EnumExecState_EES_PROCESSING_OUPUTS
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
// 		st = pb.EnumExecState_EES_PAUSED
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
// 	if !jobContext.status.Paused && jobContext.status.Status == pb.EnumExecState_EES_RUNNING && len(jobContext.status.ContainerId) > 0 {
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
// 		jobContext.status.Status == pb.EnumExecState_EES_RUNNING &&
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
