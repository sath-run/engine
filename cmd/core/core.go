package core

import (
	"context"
	"crypto/tls"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	pb "github.com/sath-run/engine/pkg/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
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
	ErrRunning     = errors.New("engine is running")
	ErrStopped     = errors.New("invalid status: STOPPED")
)

type Global struct {
	mu          sync.RWMutex
	status      int
	serviceDone chan bool

	heartBeatTicker *time.Ticker
	heartBeatDone   chan bool

	token        string
	grpcConn     *grpc.ClientConn
	grpcClient   pb.EngineClient
	dockerClient *client.Client
}

var g = Global{
	serviceDone:   make(chan bool),
	heartBeatDone: make(chan bool),
}

type Config struct {
	GrpcAddress string
	SSL         bool
}

func (g *Global) ContextWithToken(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", g.token)
}

func Init(config *Config) error {
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

	ctx := context.Background()
	token := readToken()
	if len(token) > 0 {
		ctx = metadata.AppendToOutgoingContext(context.Background(), "authorization", token)
	}
	resp, err := g.grpcClient.HandShake(ctx, &pb.HandShakeRequest{})
	if err != nil {
		return errors.WithStack(err)
	}
	g.token = resp.Token
	if len(token) == 0 {
		saveToken(resp.Token)
	}

	g.heartBeatTicker = time.NewTicker(30 * time.Second)
	g.heartBeatDone = make(chan bool)
	setupHeartBeat()

	g.status = STATUS_WAITING
	return nil
}

func readToken() string {
	dir, err := getExecutableDir()
	if err != nil {
		return ""
	}
	bytes, err := os.ReadFile(filepath.Join(dir, ".sath.token"))
	if err != nil {
		return ""
	}
	return string(bytes)
}

func saveToken(token string) error {
	dir, err := getExecutableDir()
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, ".sath.token"), []byte(token), 0666)
}

func getExecutableDir() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	executable, err = filepath.EvalSymlinks(executable)
	dir := filepath.Dir(executable)
	return dir, err
}

func setupHeartBeat() {
	go func() {
		for {
			select {
			case <-g.heartBeatDone:
				return
			case <-g.heartBeatTicker.C:
				ctx := g.ContextWithToken(context.Background())
				_, _ = g.grpcClient.HeartBeats(ctx, &pb.HeartBeatsRequest{
					Os:      runtime.GOOS, // todo
					CpuInfo: "",           // todo
					MemInfo: "",           // todo
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
			err := RunSingleJob(g.ContextWithToken(ctx))
			if err != nil {
				log.Printf("%+v\n", err)
				time.Sleep(time.Second * 5)
			}
		}
	}()
}
