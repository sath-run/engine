package core

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	pb "github.com/sath-run/engine/pkg/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	STATUS_UNINITIALIZED = iota
	STATUS_WAITING
	STATUS_STARTING
	STATUS_RUNNING
	STATUS_STOPPED
)

var (
	ErrInitailized = errors.New("core has already been initailized")
	ErrStopped     = errors.New("invalid status: STOPPED")
)

var g = struct {
	mu          sync.RWMutex
	status      int
	serviceDone chan bool

	heartBeatTicker *time.Ticker
	heartBeatDone   chan bool

	grpcConn     *grpc.ClientConn
	grpcClient   pb.EngineClient
	dockerClient *client.Client
}{
	serviceDone:   make(chan bool),
	heartBeatDone: make(chan bool),
}

type Config struct {
	GrpcAddress string
}

func Init(config *Config) error {
	// // Set up a connection to the server.
	var err error

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.status != STATUS_UNINITIALIZED {
		return ErrInitailized
	}

	g.grpcConn, err = grpc.Dial(config.GrpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return errors.WithStack(err)
	}

	g.grpcClient = pb.NewEngineClient(g.grpcConn)

	g.dockerClient, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return errors.WithStack(err)
	}

	g.heartBeatTicker = time.NewTicker(30 * time.Second)
	g.heartBeatDone = make(chan bool)
	setupHeartBeat()

	g.status = STATUS_WAITING
	return nil
}

func setupHeartBeat() {
	go func() {
		ctx := context.Background()
		for {
			select {
			case <-g.heartBeatDone:
				return
			case <-g.heartBeatTicker.C:
				_, _ = g.grpcClient.HeartBeats(ctx, &pb.HeartBeatsRequest{
					DeviceId: "", // todo
					Os:       "", // todo
					CpuInfo:  "", // todo
					MemInfo:  "", // todo
					Ip:       "", // todo
				})
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
			err := RunSingleJob(ctx)
			if err != nil {
				log.Printf("%+v\n", err)
			}
		}
	}()
}
