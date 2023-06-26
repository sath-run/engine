package core_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/docker/cli/opts"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/sath-run/engine/cmd/core"
	"github.com/sath-run/engine/cmd/utils"
)

func TestDockerPull(t *testing.T) {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	checkErr(err)

	authConfig := types.AuthConfig{
		Username: "username",
		Password: "password",
	}
	encodedJSON, err := json.Marshal(authConfig)
	checkErr(err)
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)
	ctx := context.Background()
	err = core.PullImage(ctx, dockerClient, "zengxinzhy/vina", types.ImagePullOptions{
		RegistryAuth: authStr,
	}, func(text string) {
		var obj map[string]any
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
		var msg []any
		for _, obj := range cbody.Warnings {
			msg = append(msg, obj)
		}
		utils.LogWarning(msg...)
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

func TestContainerList(t *testing.T) {
	ctx := context.Background()

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	filter := filters.NewArgs(filters.Arg("label", "run.sath.starter"))
	containers, err := dockerClient.ContainerList(ctx, types.ContainerListOptions{
		Filters: filter,
	})
	if err != nil {
		panic(err)
	}
	spew.Dump(containers)
}

func TestVinaImage(t *testing.T) {
	ctx := context.Background()
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	checkErr(err)
	cmds := []string{
		"bin/qvina02",
		"--receptor", "/data/8HDO.pdbqt",
		"--ligand", "/data/SVR.pdbqt",
		"--center_x", "138.27592581289787",
		"--center_y", "127.76051810935691",
		"--center_z", "106.25829569498698",
		"--size_x", "18.5", "--size_y", "18.5", "--size_z", "18.5",
		"--out", "/output/out.txt",
		"--log", "/output/log.txt",
	}
	containerId, err := core.CreateContainer(ctx, dockerClient, cmds, "zengxinzhy/vina", "", "vina_test", []string{
		"/Users/xinzeng/Downloads/vina:/data",
		"/Users/xinzeng/Downloads/output:/output",
	})
	checkErr(err)
	fmt.Println("containerId: ", containerId)
	err = core.ExecImage(ctx, dockerClient, containerId, func(line string) {
		fmt.Println(line)
	})
	checkErr(err)
}

func TestImageName(t *testing.T) {
	ref, err := reference.ParseNormalizedNamed("10.101.12.128/ai/astronomy_algorithm_presto_algorithm:v1.0")
	checkErr(err)
	spew.Dump(reference.FamiliarName(ref))
}

// func TestMdImage(t *testing.T) {
// 	ctx := context.Background()
// 	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
// 	if err != nil {
// 		panic(err)
// 	}
// 	var containerId string
// 	var dir = "/tmp/sath/sath_tmp_1605421609"
// 	err = core.ExecImage(
// 		ctx,
// 		dockerClient,
// 		[]string{"amber"},
// 		"zengxinzhy/amber-runtime-cuda11.4.2:1.2",
// 		dir,
// 		dir,
// 		"/data",
// 		"all",
// 		"container_name_test",
// 		&containerId,
// 		func(progress float64) {
// 			log.Printf("progress: %f\n", progress)
// 		},
// 	)
// 	if err != nil {
// 		panic(err)
// 	}
// }

// func TestSminaImage(t *testing.T) {
// 	ctx := context.Background()
// 	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
// 	if err != nil {
// 		panic(err)
// 	}
// 	var containerId string
// 	var dir = "/tmp/sath/smina"
// 	err = core.ExecImage(
// 		ctx,
// 		dockerClient,
// 		[]string{"bash", "-c", "cd /data && /main --config config.txt"},
// 		"zengxinzhy/smina:1.0",
// 		dir,
// 		dir,
// 		"/data",
// 		"",
// 		"container_name_test",
// 		&containerId,
// 		func(progress float64) {
// 			log.Printf("progress: %f\n", progress)
// 		},
// 	)
// 	if err != nil {
// 		panic(err)
// 	}
// }
