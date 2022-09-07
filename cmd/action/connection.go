package action

import (
	"context"
	"time"

	"github.com/docker/docker/client"
	pb "github.com/sath-run/engine/pkg/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	grpcConn     *grpc.ClientConn
	grpcClient   pb.EngineClient
	dockerClient *client.Client
)

func Init(addr string) error {
	// // Set up a connection to the server.
	var err error
	grpcConn, err = grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	grpcClient = pb.NewEngineClient(grpcConn)

	dockerClient, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	ctx := context.Background()

	go func(ctx context.Context) {
		for {
			_, _ = grpcClient.HeartBeats(ctx, &pb.HeartBeatsRequest{
				DeviceId: "", // todo
				Os:       "", // todo
				CpuInfo:  "", // todo
				MemInfo:  "", // todo
				Ip:       "", // todo
			})

			time.Sleep(30 * time.Second)
		}
	}(ctx)

	return nil
}

func Deinit() {
	grpcConn.Close()
}
