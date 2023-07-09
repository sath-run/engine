package core

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/sath-run/engine/cmd/utils"
	pb "github.com/sath-run/engine/pkg/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

const VERSION = "1.6.0"

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

	credential   LoginCredential
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
	HostDir     string
}

func (g *Global) ContextWithToken(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx,
		"authorization", g.credential.Token,
		"version", VERSION)
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
	utils.LogDebug("initializing core")
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
		return errors.WithStack(err)
	}

	if cred := readCredential(); cred != nil {
		g.credential = *cred
	}
	sysInfo := ""

	if sysInfo, err = getSystemInfo(); err != nil {
		return err
	}
	resp, err := g.grpcClient.HandShake(g.ContextWithToken(context.TODO()), &pb.HandShakeRequest{
		SystemInfo: sysInfo,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	if len(resp.Token) == 0 {
		return errors.New("handshake did not get token")
	} else if g.credential.Token != resp.Token {
		g.credential.DeviceId = resp.DeviceId
		g.credential.Token = resp.Token
		g.credential.UserId = resp.UserId
		if err := saveCredential(g.credential); err != nil {
			return err
		}
	}

	if len(config.DataDir) > 0 {
		g.localDataDir = config.DataDir
	} else if dir, err := utils.GetExecutableDir(); err != nil {
		panic(err)
	} else if err := os.MkdirAll(filepath.Join(dir, "data"), os.ModePerm); err != nil {
		panic(err)
	} else {
		g.localDataDir = filepath.Join(dir, "data")
	}

	g.hostDataDir = config.HostDir

	if err := StopCurrentRunningContainers(g.dockerClient); err != nil {
		panic(err)
	}
	cleanup()
	setupHeartBeat()
	setupDump()

	g.status = STATUS_WAITING
	utils.LogDebug("core initialized")

	return nil
}

func setupHeartBeat() {
	ticker := time.NewTicker(30 * time.Second)
	stream, err := g.grpcClient.RouteCommand(g.ContextWithToken(context.TODO()))
	if err != nil {
		utils.LogError(err)
	}
	var mu sync.RWMutex
	go func() {
		for {
			<-g.heartbeatResetChan
			if s, err := g.grpcClient.RouteCommand(g.ContextWithToken(context.TODO())); err != nil {
				utils.LogError(err)
			} else {
				utils.LogDebug("RouteCommand stream reconnected")
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
				utils.LogDebug("Send Heartbeat")
				if err = s.Send(&pb.CommandResponse{}); errors.Is(err, io.EOF) {
					// if stream is disconnected, reconnect
					select {
					case g.heartbeatResetChan <- true:
					default: //
					}
				} else if err != nil {
					utils.LogError(err)
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
					utils.LogError(err)
				}
			}
		}
	}()
}

func processCmdStream(stream pb.Engine_RouteCommandClient) error {
	in, err := stream.Recv()
	utils.LogDebug("received cmd:", spew.Sdump(in))
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
				err := RunSingleJob(g.ContextWithToken(ctx), g.credential.OrganizationId)
				if errors.Is(err, ErrNoJob) {
					utils.LogWarning("no job")
					time.Sleep(time.Second * 60)
				} else if errors.Is(err, context.Canceled) {
					utils.LogWarning("job cancelled")
				} else if err != nil {
					utils.LogWarning(err)
					utils.LogError(err)
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
		dir, err := utils.GetExecutableDir()
		if err != nil {
			return err
		}
		dataDir := filepath.Join(dir, "data")
		if err := os.RemoveAll(dataDir); err != nil {
			return err
		}
		err = os.MkdirAll(dataDir, os.ModePerm)
		if err != nil {
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

func getSystemInfo() (string, error) {
	cpus, err := cpu.Info()
	if err != nil {
		return "", errors.WithStack(err)
	}
	hostInfo, err := host.Info()
	if err != nil {
		return "", errors.WithStack(err)
	}

	meminfo, err := mem.VirtualMemory()
	if err != nil {
		return "", errors.WithStack(err)
	}

	info := map[string]interface{}{
		"cpus": cpus,
		"host": map[string]interface{}{
			"os":              hostInfo.OS,
			"platform":        hostInfo.Platform,
			"platformFamily":  hostInfo.PlatformFamily,
			"platformVersion": hostInfo.PlatformVersion,
			"kernelVersion":   hostInfo.KernelVersion,
			"kernelArch":      hostInfo.KernelArch,
		},
		"memory": map[string]interface{}{
			"total": meminfo.Total,
		},
	}

	bytes, err := json.Marshal(&info)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(bytes), nil
}

func setupDump() {
	if strings.ToLower(os.Getenv("SATH_MODE")) != "docker" {
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
