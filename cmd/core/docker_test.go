package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"testing"

	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/sath-run/engine/cmd/utils"
)

func TestDockerPull(t *testing.T) {
	err := Init(&Config{
		GrpcAddress: "localhost:50051",
	})

	if err != nil {
		log.Printf("%+v\n", err)
		panic(err)
	}

	ctx := context.Background()
	err = PullImage(ctx, &DockerImageConfig{
		Repository: "zengxinzhy/vinadock",
		Tag:        "latest",
		Digest:     "sha256:b5c96c44fcd3b48f30c0dfee99c97bceaa0037b86e15d0f28404d0c1f25dbcfd",
		Uri:        "",
	}, func(text string) {
		var obj gin.H
		if err := json.Unmarshal([]byte(text), &obj); err != nil {
			panic(err)
		}
		log.Println(obj)
	})

	if err != nil {
		log.Printf("%+v\n", err)
		panic(err)
	}
}

func TestDockerGPU(t *testing.T) {
	ctx := context.Background()

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	gpuOpts := opts.GpuOpts{}
	gpuOpts.Set("")

	cbody, err := dockerClient.ContainerCreate(ctx, &container.Config{
		Cmd:   []string{"echo", "hello sath"},
		Image: "amber-runtime",
		Tty:   true,
		Labels: map[string]string{
			"run.sath.starter": "",
		},
	}, &container.HostConfig{
		Resources: container.Resources{DeviceRequests: gpuOpts.Value()},
	}, nil, nil, "")
	if err != nil {
		panic(err)
	}
	if len(cbody.Warnings) > 0 {
		utils.LogWarning(cbody.Warnings...)
	}
	fmt.Println(cbody.ID)

	if err := dockerClient.ContainerStart(ctx, cbody.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	out, err := dockerClient.ContainerLogs(ctx, cbody.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		Follow:     true,
		Details:    true,
	})
	if err != nil {
		panic(err)
	}
	defer out.Close()

	data, err := io.ReadAll(out)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(data))
}
