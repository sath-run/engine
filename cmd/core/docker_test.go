package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
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

func TestDockerInfo(t *testing.T) {
	ctx := context.Background()

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	inspect, err := dockerClient.ContainerInspect(ctx, "5c97f54576bdc2558bf70f4a4e8c96057647542d4af71779b1098d7b4b3b419c")
	if err != nil {
		panic(err)
	}
	spew.Dump(inspect.HostConfig)
}

func TestDockerPrune(t *testing.T) {
	ctx := context.Background()

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	arg := filters.Arg("label", "run.sath.starter")
	report, err := dockerClient.ContainersPrune(ctx, filters.NewArgs(arg))
	if err != nil {
		panic(err)
	}
	spew.Dump(report)
}
