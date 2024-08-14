package daemon_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sath-run/engine/daemon"
	pb "github.com/sath-run/engine/daemon/protobuf"
	"github.com/sath-run/engine/meta"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ClientStream struct {
}

type RouteCommandClient struct {
	*ClientStream
}

type NotifyExecStatusClient struct {
	*ClientStream
}

type EngineClient struct {
	NotifyExecStatusClient pb.Engine_NotifyExecStatusClient
	RouteCommandClient     pb.Engine_RouteCommandClient
}

func (stream *ClientStream) Header() (metadata.MD, error) {
	return nil, nil
}

func (stream *ClientStream) Trailer() metadata.MD {
	return nil
}

func (stream *ClientStream) CloseSend() error {
	return nil
}

func (stream *ClientStream) Context() context.Context {
	return nil
}

func (stream *ClientStream) SendMsg(m any) error {
	return nil
}

func (stream *ClientStream) RecvMsg(m any) error {
	log.Debug().Msg("RecvMsg")
	return nil
}

func (x *RouteCommandClient) Send(m *pb.CommandResponse) error {
	return nil
}

func (x *RouteCommandClient) Recv() (*pb.CommandRequest, error) {
	return nil, nil
}

func (x *NotifyExecStatusClient) Send(m *pb.ExecNotificationRequest) error {
	return nil
}

func (x *NotifyExecStatusClient) CloseAndRecv() (*pb.ExecNotificationResponse, error) {
	return nil, nil
}

func (client *EngineClient) HandShake(ctx context.Context, in *pb.HandShakeRequest, opts ...grpc.CallOption) (*pb.HandShakeResponse, error) {
	return &pb.HandShakeResponse{
		Token:    "TestDeviceToken",
		DeviceId: "Device001",
	}, nil
}
func (client *EngineClient) Login(ctx context.Context, in *pb.LoginRequest, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
	return &pb.LoginResponse{
		Token:     "TestUserToken",
		UserId:    "User001",
		UserName:  "Doe John",
		UserEmail: "user@test.com",
	}, nil
}

func (client *EngineClient) NotifyExecStatus(ctx context.Context, opts ...grpc.CallOption) (pb.Engine_NotifyExecStatusClient, error) {
	return client.NotifyExecStatusClient, nil
}
func (client *EngineClient) RouteCommand(ctx context.Context, opts ...grpc.CallOption) (pb.Engine_RouteCommandClient, error) {
	return client.RouteCommandClient, nil
}

var jobCount = 2

func (client *EngineClient) GetNewJob(ctx context.Context, in *pb.JobGetRequest, opts ...grpc.CallOption) (*pb.JobGetResponse, error) {
	if jobCount == 0 {
		return nil, nil
	}
	jobCount--
	return &pb.JobGetResponse{
		JobId:     fmt.Sprintf("Job%03d", jobCount),
		ProjectId: fmt.Sprintf("Project%03d", jobCount),
		ExecId:    fmt.Sprintf("Exec%03d", jobCount),
		GpuConf: &pb.GpuConf{
			Opt: pb.GpuOpt_EGO_None,
		},
		Image: &pb.Image{
			Url: "sathrun/base",
		},
		Cmd: []string{
			"bash",
			"-c",
			fmt.Sprintf("python -c 'print(%d*%d)' > /output/result.txt", jobCount, jobCount),
		},
		Inputs: []*pb.JobInput{
			{
				Path: "wiki.txt",
				Req: &pb.FileRequest{
					Url:    "https://www.wikipedia.org/",
					Method: "GET",
				},
			},
		},
	}, nil
}

func NewEngineClient() *EngineClient {
	return &EngineClient{
		NotifyExecStatusClient: &NotifyExecStatusClient{},
		RouteCommandClient:     &RouteCommandClient{},
	}
}

var c *daemon.Connection

func checkErr(err error) {
	if err != nil {
		log.Panic().Err(err).Msg("error")
	}
}

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	var err error

	err = meta.Init()
	checkErr(err)

	client := NewEngineClient()
	c, err = daemon.NewConnectionWithClient(client)
	checkErr(err)
	c.Login(context.TODO(), "", "")
	code := m.Run()
	os.Exit(code)
}

func TestScheduler(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	checkErr(err)
	log.Trace().Str("schedulerDir", dir).Send()
	log.Trace().Any("user", c.User()).Send()
	s, err := daemon.NewScheduler(context.Background(), c, dir, 5*time.Second)
	checkErr(err)
	s.Start()
	time.Sleep(time.Second * 90)
}
