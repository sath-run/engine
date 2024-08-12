package core

import (
	"context"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sath-run/engine/engine/core/conns"
	"github.com/sath-run/engine/engine/core/scheduler"
	"github.com/sath-run/engine/engine/logger"
	"github.com/sath-run/engine/utils"
)

const (
	STATUS_UNINITIALIZED = "uninitialized"
	STATUS_STARTING      = "starting"
	STATUS_RUNNING       = "running"
	STATUS_PAUSED        = "paused"
	STATUS_STOPPING      = "stopping"
)

var (
	ErrInitailized = errors.New("core has already been initailized")
	ErrRunning     = errors.New("engine is running")
	ErrStopping    = errors.New("engine is stopping")
)

type Core struct {
	mu                 sync.RWMutex
	status             string
	dumpDone           chan bool
	heartbeatResetChan chan bool

	c *conns.Connection

	cancelFunc   context.CancelFunc
	hostDataDir  string
	localDataDir string

	hb           *Heartbeat
	jobScheduler *scheduler.Scheduler
}

type Config struct {
	GrpcAddress string
	SSL         bool
	DataDir     string
}

func Default(config *Config) (*Core, error) {
	logger.Debug("initializing core")
	// // Set up a connection to the server.
	var err error
	var core = &Core{
		dumpDone:           make(chan bool),
		heartbeatResetChan: make(chan bool),
		cancelFunc:         nil,
	}

	core.c, err = conns.NewConnection(config.GrpcAddress, config.SSL)
	if err != nil {
		log.Fatal(err)
	}

	core.localDataDir = filepath.Join(utils.SathHome, "/data")

	if os.Getenv("SATH_ENV") == "docker" {
		core.hostDataDir = config.DataDir
	} else {
		core.hostDataDir = core.localDataDir
	}

	if err := os.MkdirAll(core.localDataDir, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	core.hb = NewHeartbeat(core.c)
	core.jobScheduler, err = scheduler.NewScheduler(context.TODO(), core.c, core.localDataDir, time.Second*30)
	if err != nil {
		return nil, err
	}
	if err := core.cleanup(); err != nil {
		log.Fatal(err)
	}

	if u := core.c.User(); u != nil {
		core.Start()
	} else {
		core.status = STATUS_PAUSED
	}
	logger.Debug("core initialized")

	return core, nil
}

func (core *Core) Start() error {
	core.mu.Lock()
	defer core.mu.Unlock()

	if core.status == STATUS_RUNNING {
		return ErrRunning
	}

	if core.status == STATUS_STOPPING {
		return ErrStopping
	}

	core.status = STATUS_STARTING

	core.jobScheduler.Start()

	core.status = STATUS_RUNNING
	return nil
}

func (core *Core) Stop(waitTillJobDone bool) error {
	core.mu.Lock()
	defer core.mu.Unlock()

	if core.status == STATUS_STOPPING && !waitTillJobDone {
		core.cancelFunc()
		return nil
	}

	if core.status != STATUS_RUNNING {
		return nil
	}
	core.status = STATUS_STOPPING
	return nil
}

func (core *Core) cleanup() error {
	if core.status == STATUS_UNINITIALIZED {
		// clean up data folder
		if err := os.RemoveAll(core.localDataDir); err != nil {
			return err
		}
		if err := os.MkdirAll(core.localDataDir, os.ModePerm); err != nil {
			return err
		}
	}

	// clean up stopped containers
	// arg := filters.Arg("label", "run.sath.starter")
	// if _, err := core.docker.ContainersPrune(context.Background(), filters.NewArgs(arg)); err != nil {
	// 	return err
	// }
	return nil
}

func dump() {
	// fmt.Printf("\n=======================================================\n")
	// fmt.Printf(
	// 	"[SATH DUMP] %v\n",
	// 	time.Now().Format("2006/01/02 - 15:04:05"),
	// )
	// fmt.Printf("SATH Engine status: %s\n", Status())
	// if jobContext.status.IsNil() {
	// 	fmt.Println("No job is running right now")
	// } else {
	// 	fmt.Println("SATH Engine current jobs:")
	// 	printJobs([]*JobStatus{&jobContext.status})
	// }

}

// func printJobs(jobs []*JobStatus) {
// 	fmt.Printf("%-10s %-14s %-10s %-30s %-16s %-16s %-16s\n",
// 		"JOB ID", "STATUS", "PROGRESS", "IMAGE", "CONTAINER ID", "CREATED", "COMPLETED")
// 	for _, job := range jobs {
// 		createdAt := job.CreatedAt
// 		completedAt := job.CompletedAt
// 		image := job.Image
// 		created := fmtDuration(time.Since(createdAt)) + " ago"
// 		completed := ""
// 		if !completedAt.IsZero() {
// 			completed = fmtDuration(time.Since(completedAt)) + " ago"
// 		}
// 		containerId := job.ContainerId
// 		if len(containerId) > 12 {
// 			containerId = containerId[:12]
// 		}
// 		fmt.Printf("%-10s %-14s %-10.2f %-30s %-16s %-16s %-16s\n",
// 			job.Id, job.Status, job.Progress, image, containerId,
// 			created,
// 			completed,
// 		)
// 	}
// }

func fmtDuration(d time.Duration) string {
	if d > time.Hour {
		amount := math.Round(d.Hours())
		if amount == 1 {
			return strconv.Itoa(int(amount)) + " hour"
		} else {
			return strconv.Itoa(int(amount)) + " hours"
		}
	} else if d > time.Minute {
		amount := math.Round(d.Minutes())
		if amount == 1 {
			return strconv.Itoa(int(amount)) + " minute"
		} else {
			return strconv.Itoa(int(amount)) + " minutes"
		}
	} else {
		amount := math.Round(d.Seconds())
		if amount == 1 {
			return strconv.Itoa(int(amount)) + " second"
		} else {
			return strconv.Itoa(int(amount)) + " seconds"
		}
	}
}

type UserInfo struct {
	Id    string
	Name  string
	Email string
}

func (core *Core) GetUserInfo() *UserInfo {
	u := core.c.User()
	if u == nil {
		return nil
	}
	return &UserInfo{
		Id:    u.Id,
		Name:  u.Name,
		Email: u.Email,
	}
}

func (core *Core) Status() string {
	return core.status
}

func (core *Core) Login(account string, password string) error {
	return core.c.Login(context.TODO(), account, password)
}

func (core *Core) Logout() error {
	return core.c.Logout()
}
