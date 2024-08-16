package daemon

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	pb "github.com/sath-run/engine/daemon/protobuf"
)

var (
	ErrUnautherized = errors.New("unautherized")
	ErrNoJob        = errors.New("no job")
	ErrActionBusy   = errors.New("action busy")
	ErrNoUser       = errors.New("no user")
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
	c           *Connection
	cli         *client.Client
	dir         string
	status      Status
	closeChan   chan struct{}
	jobChan     chan *Job
	actionLock  sync.Mutex
	fetchLock   sync.Mutex
	pendingJobs map[*Job]bool
	logger      zerolog.Logger
	// containers
	containers []*Container
}

func NewScheduler(ctx context.Context, c *Connection, dir string, jobInterval time.Duration) (*Scheduler, error) {
	docker, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	if err := stopCurrentRunningContainers(ctx, docker); err != nil {
		return nil, err
	}
	scheduler := Scheduler{
		c:           c,
		cli:         docker,
		dir:         dir,
		status:      StatusPaused,
		jobChan:     make(chan *Job, 8),
		containers:  []*Container{},
		pendingJobs: map[*Job]bool{},
		logger:      log.With().Str("component", "scheduler").Logger(),
	}
	go scheduler.loop(jobInterval)
	return &scheduler, nil
}

func (scheduler *Scheduler) loop(jobInterval time.Duration) {
	ticker := time.NewTicker(jobInterval)

	// main logic of scheduler, note that each code inside the for loop should be noneblock
	for {
		select {
		case job := <-scheduler.jobChan:
			if job.err != nil {
				job.logger.Info().Err(job.err).Str("state", job.state.String()).Send()
				go job.handleCompletion()
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
			case pb.EnumExecState_EES_RUNNING:
				go job.postprocess()
				scheduler.rescheduleContainer(job.container)
			case pb.EnumExecState_EES_SUCCESS:
				job.logger.Info().Msg("succeed")
				go job.handleCompletion()
			default:
				job.err = errors.New("unexpected job state")
				job.logger.Fatal().Str("state", job.state.String()).Err(job.err).Send()
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
	runningJobs := 0
	for _, container := range scheduler.containers {
		if container.currentJob != nil {
			runningJobs++
		}
	}
	if runningJobs > 1 {
		// TODO: instead of checking number of running jobs, it's better to check system resources
		return
	}
	if len(scheduler.pendingJobs) > 0 {
		return
	}
	user := scheduler.c.user
	if user == nil {
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
		ctx = scheduler.c.AppendToOutgoingContext(context.Background(), user)
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		res, err := scheduler.c.GetNewJob(ctx, &pb.JobGetRequest{})
		if err != nil {
			scheduler.logger.Warn().Err(err).Msg("scheduler fails to get a new job")
			return
		}
		if res == nil || len(res.ExecId) == 0 {
			// no available jobs from server
			scheduler.logger.Info().Msg("no available jobs from server")
			return
		}
		dir := filepath.Join(scheduler.dir, "job_"+res.ExecId)
		ctx = scheduler.c.AppendToOutgoingContext(context.Background(), user)
		job, err := newJob(ctx, scheduler.c, scheduler.cli, scheduler.jobChan, dir, res)
		if err != nil {
			scheduler.logger.Warn().Err(err).Msg("scheduler fails to create job")
			return
		}
		scheduler.logger.Trace().Any("scheduler fetched new job", res).Send()
		scheduler.jobChan <- job
	}()
}

func (scheduler *Scheduler) attachContainerForJob(job *Job) bool {
	var container *Container

	// find container if any
	for _, c := range scheduler.containers {
		if c.imageUrl == job.metadata.Image.Url {
			container = c
			break
		}
	}

	if container == nil {
		// allocate new container
		// TODO: should check system resouces before deciding to create a new container

		// ignore mkdir error if any, it will be handled inside job.run
		dir, _ := os.MkdirTemp(scheduler.dir, "container_")
		container = newContainer(scheduler.cli, dir, job)
		scheduler.containers = append(scheduler.containers, container)
		job.logger.Debug().Str("dir", dir).Msg("container created for job")
	} else if container.currentJob == nil {
		container.currentJob = job
		scheduler.logger.Debug().Str("container", container.id).Str("job", job.metadata.ExecId).Msg("attach container for job")
	} else {
		// TODO: support multiple container for the same image
		container = nil
	}
	if container == nil {
		// if no container found nor a new container was allocated, enqueue job
		scheduler.pendingJobs[job] = true
		scheduler.logger.Debug().Int("pendingJobs", len(scheduler.pendingJobs)).Str("job", job.metadata.ExecId).Msg("job queued")
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
