package core

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/sath-run/engine/cmd/utils"
	pb "github.com/sath-run/engine/pkg/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

const VERSION = "1.4.1"

const (
	STATUS_UNINITIALIZED = iota
	STATUS_WAITING
	STATUS_STARTING
	STATUS_RUNNING
)

var (
	ErrInitailized = errors.New("core has already been initailized")
	ErrRunning     = errors.New("engine is running")
	ErrStopped     = errors.New("invalid status: STOPPED")
)

type Global struct {
	mu          sync.RWMutex
	status      int
	serviceDone chan bool

	heartBeatDone chan bool
	dumpDone      chan bool

	token        string
	isUser       bool
	grpcConn     *grpc.ClientConn
	grpcClient   pb.EngineClient
	dockerClient *client.Client

	hostDataDir string
}

var g = Global{
	serviceDone:   make(chan bool),
	heartBeatDone: make(chan bool),
	dumpDone:      make(chan bool),
}

type Config struct {
	GrpcAddress string
	SSL         bool
	DataPath    string
}

func (g *Global) ContextWithToken(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", g.token)
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
	default:
		return "UNKNOWN"
	}
}

func Init(config *Config) error {
	// // Set up a connection to the server.
	var err error

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.status != STATUS_UNINITIALIZED {
		return ErrInitailized
	}

	g.hostDataDir = config.DataPath

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

	ctx := context.Background()
	token := readToken()
	sysInfo := ""
	if len(token) > 0 {
		ctx = metadata.AppendToOutgoingContext(context.Background(), "authorization", token)
	} else {
		if sysInfo, err = getSystemInfo(); err != nil {
			return err
		}
	}
	resp, err := g.grpcClient.HandShake(ctx, &pb.HandShakeRequest{
		SystemInfo: sysInfo,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	g.token = resp.Token
	g.isUser = resp.IsUser
	if len(token) == 0 {
		saveToken(resp.Token, false)
	}

	setupHeartBeat()
	setupDump()

	g.status = STATUS_WAITING
	return nil
}

func readToken() string {
	dir, err := utils.GetExecutableDir()
	if err != nil {
		return ""
	}
	bytes, err := os.ReadFile(filepath.Join(dir, ".user.token"))
	if err != nil {
		bytes, err = os.ReadFile(filepath.Join(dir, ".device.token"))
		if err != nil {
			return ""
		}
	}
	return string(bytes)
}

func saveToken(token string, isUser bool) error {
	dir, err := utils.GetExecutableDir()
	if err != nil {
		return err
	}

	if isUser {
		return os.WriteFile(filepath.Join(dir, ".user.token"), []byte(token), 0666)

	} else {
		return os.WriteFile(filepath.Join(dir, ".device.token"), []byte(token), 0666)
	}
}

func setupHeartBeat() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for {
			select {
			case <-g.heartBeatDone:
				return
			case <-ticker.C:
				ctx := g.ContextWithToken(context.Background())
				info := pb.HeartBeatsRequest{
					Version: VERSION,
				}
				status := GetJobStatus()
				if status != nil {
					info.ExecInfos = append(info.ExecInfos, &pb.HeartBeatsRequest_ExecInfo{
						ExecId:   status.Id,
						Status:   status.Status,
						Progress: float32(status.Progress),
						Message:  status.Message,
					})
				}
				_, _ = g.grpcClient.HeartBeats(ctx, &info)
			}
		}
	}()
}

func Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.status == STATUS_RUNNING {
		return nil
	}

	g.status = STATUS_STARTING
	run()
	g.status = STATUS_RUNNING
	return nil
}

func Stop() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.status != STATUS_RUNNING {
		return nil
	}

	g.serviceDone <- true
	g.status = STATUS_WAITING
	return nil
}

func run() {
	ctx, cancel := context.WithCancel(context.Background())
	stop := false
	go func() {
		<-g.serviceDone
		stop = true
		cancel()
	}()
	go func() {
		for !stop {
			err := RunSingleJob(g.ContextWithToken(ctx))
			if errors.Is(err, ErrNoJob) {
				log.Println("no job")
				time.Sleep(time.Second * 90)
			} else if errors.Is(err, context.Canceled) {
				log.Println("job cancelled")
			} else if err != nil {
				log.Printf("%+v\n", err)
				time.Sleep(time.Second * 5)
			}
		}
	}()
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
	if jobContext.jobStatus == nil {
		fmt.Println("No job is running right now")
	} else {
		fmt.Println("SATH Engine current jobs:")
		printJobs([]*JobStatus{jobContext.jobStatus})
	}

}

func printJobs(jobs []*JobStatus) {
	fmt.Printf("%-10s %-14s %-10s %-30s %-16s %-16s %-16s\n",
		"JOB ID", "STATUS", "PROGRESS", "IMAGE", "CONTAINER ID", "CREATED", "COMPLETED")
	for _, job := range jobs {
		createdAt := job.CreatedAt
		completedAt := job.CompletedAt
		image := strings.Split(job.Image, "@")[0]
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
