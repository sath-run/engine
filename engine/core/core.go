package core

import (
	"context"
	"crypto/tls"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	pb "github.com/sath-run/engine/engine/core/protobuf"
	"github.com/sath-run/engine/engine/logger"
	"github.com/sath-run/engine/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	STATUS_UNINITIALIZED = iota
	STATUS_WAITING
	STATUS_STARTING
	STATUS_RUNNING
	STATUS_STOPPING
)

var (
	ErrInitailized = errors.New("core has already been initailized")
	ErrRunning     = errors.New("engine is running")
	ErrStopping    = errors.New("engine is stopping")
)

type Core struct {
	mu                 sync.RWMutex
	status             int
	dumpDone           chan bool
	heartbeatResetChan chan bool

	user       *User
	grpcClient pb.EngineClient

	cancelFunc   context.CancelFunc
	hostDataDir  string
	localDataDir string

	hb           *Heartbeat
	jobScheduler *JobScheduler
}

type Config struct {
	GrpcAddress string
	SSL         bool
	DataDir     string
}

func (core *Core) Status() string {
	switch core.status {
	case STATUS_UNINITIALIZED:
		return "UNINITIALIZED"
	case STATUS_STARTING:
		return "STARTING"
	case STATUS_WAITING:
		return "WAITING"
	case STATUS_RUNNING:
		return "RUNNING"
	case STATUS_STOPPING:
		return "STOPPING"
	default:
		return "UNKNOWN"
	}
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

	var credential credentials.TransportCredentials
	if config.SSL {
		credential = credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: false,
		})
	} else {
		credential = insecure.NewCredentials()
	}

	grpcConn, err := grpc.NewClient(config.GrpcAddress, grpc.WithTransportCredentials(credential))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	core.grpcClient = pb.NewEngineClient(grpcConn)

	core.user, err = NewUser(core.grpcClient)
	if err != nil {
		log.Fatal(err)
	}

	core.localDataDir = filepath.Join(utils.ExecutableDir, "/data")

	if os.Getenv("SATH_ENV") == "docker" {
		core.hostDataDir = config.DataDir
	} else {
		core.hostDataDir = core.localDataDir
	}

	if err := os.MkdirAll(core.localDataDir, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	spew.Dump(dockerClient)
	if err := StopCurrentRunningContainers(dockerClient); err != nil {
		log.Fatal(err)
	}
	if err := core.cleanup(); err != nil {
		log.Fatal(err)
	}

	core.hb = NewHeartbeat(core.user.ContextWithToken(context.TODO()), core.grpcClient)
	core.jobScheduler = NewJobScheduler(core.user, core.grpcClient, dockerClient, core.localDataDir)
	core.status = STATUS_WAITING
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
	arg := filters.Arg("label", "run.sath.starter")
	if _, err := core.jobScheduler.docker.ContainersPrune(context.Background(), filters.NewArgs(arg)); err != nil {
		return err
	}
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
	if core.user == nil || core.user.Id == "" {
		return nil
	}
	core.user.mu.RLock()
	defer core.user.mu.RUnlock()

	return &UserInfo{
		Id:    core.user.Id,
		Name:  core.user.Name,
		Email: core.user.Email,
	}
}

func (core *Core) Login(account string, password string) error {
	return core.user.Login(core.grpcClient, account, password)
}

func (core *Core) Logout() error {
	return core.user.Logout()
}
