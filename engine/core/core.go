package core

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
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
	"github.com/sath-run/engine/constants"
	pb "github.com/sath-run/engine/engine/core/protobuf"
	"github.com/sath-run/engine/engine/logger"
	"github.com/sath-run/engine/meta"
	"github.com/sath-run/engine/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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

type Global struct {
	mu                 sync.RWMutex
	status             int
	serviceDone        chan bool
	dumpDone           chan bool
	heartbeatResetChan chan bool

	userToken    string
	deviceToken  string
	deviceId     string
	userInfo     *UserInfo
	grpcConn     *grpc.ClientConn
	grpcClient   pb.EngineClient
	dockerClient *client.Client

	cancelJob    context.CancelFunc
	hostDataDir  string
	localDataDir string
}

var g = Global{
	serviceDone:        make(chan bool),
	dumpDone:           make(chan bool),
	heartbeatResetChan: make(chan bool),
	cancelJob:          nil,
}

type Config struct {
	GrpcAddress string
	SSL         bool
	DataDir     string
}

func (g *Global) ContextWithToken(ctx context.Context) context.Context {
	var token string
	if g.userToken != "" {
		token = g.userToken
	} else {
		token = g.deviceToken
	}
	return metadata.AppendToOutgoingContext(ctx,
		"authorization", token,
		"version", constants.Version)
}

func GetDockerClient() *client.Client {
	return g.dockerClient
}

func Status() string {
	switch g.status {
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

func Init(config *Config) error {
	logger.Debug("initializing core")
	// // Set up a connection to the server.
	var err error

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.status != STATUS_UNINITIALIZED {
		return ErrInitailized
	}

	var credential credentials.TransportCredentials
	if config.SSL {
		credential = credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: false,
		})
	} else {
		credential = insecure.NewCredentials()
	}

	g.grpcConn, err = grpc.Dial(config.GrpcAddress, grpc.WithTransportCredentials(credential))
	if err != nil {
		return errors.WithStack(err)
	}

	g.grpcClient = pb.NewEngineClient(g.grpcConn)

	g.dockerClient, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	deviceToken, err := meta.GetCredentialDeviceToken()
	if err == nil {
		g.deviceToken = deviceToken
	} else if !constants.IsErrNil(err) {
		return err
	}
	resp, err := g.grpcClient.HandShake(g.ContextWithToken(context.TODO()), &pb.HandShakeRequest{
		SystemInfo: GetSystemInfo(),
	})
	if err != nil {
		return err
	}
	g.deviceToken = resp.Token
	g.deviceId = resp.DeviceId
	if err := meta.SetCredentialDeviceToken(g.deviceToken); err != nil {
		return err
	}

	userToken, err := meta.GetCredentialUserToken()
	if err != nil && !constants.IsErrNil(err) {
		return err
	}
	if len(userToken) > 0 {
		// refresh login data using userToken
		g.userToken = userToken
		userLogin("", "")
	}

	g.localDataDir = filepath.Join(utils.ExecutableDir, "/data")

	if os.Getenv("SATH_ENV") == "docker" {
		g.hostDataDir = config.DataDir
	} else {
		g.hostDataDir = g.localDataDir
	}

	if err := os.MkdirAll(g.localDataDir, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	if err := StopCurrentRunningContainers(g.dockerClient); err != nil {
		panic(err)
	}
	if err := cleanup(); err != nil {
		panic(err)
	}
	setupHeartBeat()
	setupDump()

	g.status = STATUS_WAITING
	logger.Debug("core initialized")

	return nil
}

func setupHeartBeat() {
	ticker := time.NewTicker(30 * time.Second)
	stream, err := g.grpcClient.RouteCommand(g.ContextWithToken(context.TODO()))
	if err != nil {
		logger.Error(err)
	}
	var mu sync.RWMutex
	go func() {
		for {
			<-g.heartbeatResetChan
			if s, err := g.grpcClient.RouteCommand(g.ContextWithToken(context.TODO())); err != nil {
				logger.Error(err)
			} else {
				logger.Debug("RouteCommand stream reconnected")
				mu.Lock()
				stream = s
				mu.Unlock()
			}
		}
	}()
	go func() {
		for {
			<-ticker.C
			mu.RLock()
			s := stream
			mu.RUnlock()
			if s != nil {
				logger.Debug("Send Heartbeat")
				if err = s.Send(&pb.CommandResponse{}); errors.Is(err, io.EOF) {
					// if stream is disconnected, reconnect
					select {
					case g.heartbeatResetChan <- true:
					default: //
					}
				} else if err != nil {
					logger.Error(err)
				}
			} else {
				select {
				case g.heartbeatResetChan <- true:
				default: //
				}
			}
		}
	}()
	go func() {
		for {
			mu.RLock()
			s := stream
			mu.RUnlock()
			if s == nil {
				time.Sleep(time.Second * 5)
				continue
			} else {
				if err := processCmdStream(s); err != nil {
					logger.Error(err)
				}
			}
		}
	}()
}

func processCmdStream(stream pb.Engine_RouteCommandClient) error {
	in, err := stream.Recv()
	logger.Debug("received cmd:", spew.Sdump(in))
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			return errors.WithStack(err)
		} else {
			if st.Code() == codes.Unavailable {
				select {
				case g.heartbeatResetChan <- true:
				default:
				}
			}
			time.Sleep(time.Second * 1)
			return errors.WithStack(st.Err())
		}
	}
	res := pb.CommandResponse{
		Id:      in.Id,
		Command: in.Command,
		Status:  pb.EnumCommandStatus_ECS_OK,
	}
	switch in.Command {
	case pb.EnumCommand_EC_UNSPECIFIED:
		res.Status = pb.EnumCommandStatus_ECS_OK
	case pb.EnumCommand_EC_PAUSE:
		result := Pause(in.GetData()["execId"])
		if !result {
			res.Status = pb.EnumCommandStatus_ECS_INVALID_STATE
		}
	case pb.EnumCommand_EC_RESUME:
		result := Resume(in.GetData()["execId"])
		if !result {
			res.Status = pb.EnumCommandStatus_ECS_INVALID_STATE
		}
	default:
		res.Status = pb.EnumCommandStatus_ECS_NOT_IMPLEMENTED
	}
	if err := stream.Send(&res); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.status == STATUS_RUNNING {
		return ErrRunning
	}

	if g.status == STATUS_STOPPING {
		return ErrStopping
	}

	if g.status != STATUS_WAITING {
		return fmt.Errorf("invalid engine status: %s", Status())
	}

	g.status = STATUS_STARTING
	run()
	// resume paused state
	jobContext.Resume()
	g.status = STATUS_RUNNING
	return nil
}

func Stop(waitTillJobDone bool) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.status == STATUS_STOPPING && !waitTillJobDone {
		g.cancelJob()
		return nil
	}

	if g.status != STATUS_RUNNING {
		return nil
	}
	g.serviceDone <- waitTillJobDone
	g.status = STATUS_STOPPING
	return nil
}

func run() {
	ctx, cancel := context.WithCancel(context.Background())
	g.cancelJob = cancel
	stop := false
	go func() {
		waitTillJobDone := <-g.serviceDone
		stop = true
		if !waitTillJobDone {
			cancel()
		}
	}()
	go func() {
		ticker := time.NewTicker(600 * time.Second)
		for !stop {
			select {
			case <-ticker.C:
				err := cleanup()
				if err != nil {
					log.Printf("%+v\n", err)
				}
			default:
				err := RunSingleJob(g.ContextWithToken(ctx))
				if errors.Is(err, ErrNoJob) {
					logger.Warning("no job")
					time.Sleep(time.Second * 60)
				} else if errors.Is(err, context.Canceled) {
					logger.Warning("job cancelled")
				} else if err != nil {
					logger.Warning(err)
					logger.Error(err)
					time.Sleep(time.Second * 60)
				}
			}
		}
		g.mu.Lock()
		g.status = STATUS_WAITING
		g.mu.Unlock()
	}()
}

func cleanup() error {
	if g.status == STATUS_UNINITIALIZED {
		// clean up data folder
		if err := os.RemoveAll(g.localDataDir); err != nil {
			return err
		}
		if err := os.MkdirAll(g.localDataDir, os.ModePerm); err != nil {
			return err
		}
	}

	// clean up stopped containers
	arg := filters.Arg("label", "run.sath.starter")
	if _, err := g.dockerClient.ContainersPrune(context.Background(), filters.NewArgs(arg)); err != nil {
		return err
	}
	return nil
}

func setupDump() {
	if os.Getenv("SATH_ENV") != "docker" {
		return
	}
	dump()
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		for {
			select {
			case <-g.dumpDone:
				return
			case <-ticker.C:
				dump()
			}
		}
	}()
}

func dump() {
	fmt.Printf("\n=======================================================\n")
	fmt.Printf(
		"[SATH DUMP] %v\n",
		time.Now().Format("2006/01/02 - 15:04:05"),
	)
	fmt.Printf("SATH Engine status: %s\n", Status())
	if jobContext.status.IsNil() {
		fmt.Println("No job is running right now")
	} else {
		fmt.Println("SATH Engine current jobs:")
		printJobs([]*JobStatus{&jobContext.status})
	}

}

func printJobs(jobs []*JobStatus) {
	fmt.Printf("%-10s %-14s %-10s %-30s %-16s %-16s %-16s\n",
		"JOB ID", "STATUS", "PROGRESS", "IMAGE", "CONTAINER ID", "CREATED", "COMPLETED")
	for _, job := range jobs {
		createdAt := job.CreatedAt
		completedAt := job.CompletedAt
		image := job.Image
		created := fmtDuration(time.Since(createdAt)) + " ago"
		completed := ""
		if !completedAt.IsZero() {
			completed = fmtDuration(time.Since(completedAt)) + " ago"
		}
		containerId := job.ContainerId
		if len(containerId) > 12 {
			containerId = containerId[:12]
		}
		fmt.Printf("%-10s %-14s %-10.2f %-30s %-16s %-16s %-16s\n",
			job.Id, job.Status, job.Progress, image, containerId,
			created,
			completed,
		)
	}
}

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
