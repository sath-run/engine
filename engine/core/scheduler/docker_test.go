package scheduler_test

// import (
// 	"bufio"
// 	"context"
// 	"encoding/base64"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"testing"

// 	"github.com/davecgh/go-spew/spew"
// 	"github.com/distribution/reference"
// 	"github.com/docker/cli/opts"
// 	"github.com/docker/docker/api/types/container"
// 	"github.com/docker/docker/api/types/filters"
// 	"github.com/docker/docker/api/types/image"
// 	"github.com/docker/docker/api/types/registry"
// 	"github.com/docker/docker/client"
// 	"github.com/google/shlex"
// 	"github.com/pkg/errors"
// 	"github.com/sath-run/engine/engine/core"
// 	"github.com/sath-run/engine/engine/logger"
// )

// func TestDockerPull(t *testing.T) {
// 	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
// 	checkErr(err)

// 	authConfig := registry.AuthConfig{
// 		Username: "username",
// 		Password: "password",
// 	}
// 	encodedJSON, err := json.Marshal(authConfig)
// 	checkErr(err)
// 	authStr := base64.URLEncoding.EncodeToString(encodedJSON)
// 	ctx := context.Background()
// 	err = core.PullImage(ctx, dockerClient, "zengxinzhy/vina", image.PullOptions{
// 		RegistryAuth: authStr,
// 	}, func(text string) {
// 		var obj map[string]any
// 		if err := json.Unmarshal([]byte(text), &obj); err != nil {
// 			panic(err)
// 		}
// 		fmt.Println(obj)
// 	})

// 	if err != nil {
// 		fmt.Printf("%+v\n", err)
// 		panic(err)
// 	}
// }

// func TestDockerGPU(t *testing.T) {
// 	ctx := context.Background()

// 	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
// 	if err != nil {
// 		panic(err)
// 	}

// 	gpuOpts := opts.GpuOpts{}
// 	gpuOpts.Set("")

// 	cbody, err := dockerClient.ContainerCreate(ctx, &container.Config{
// 		Cmd:   []string{"echo", "hello sath"},
// 		Image: "amber-runtime",
// 		Tty:   true,
// 		Labels: map[string]string{
// 			"run.sath.starter": "",
// 		},
// 	}, &container.HostConfig{
// 		Resources: container.Resources{DeviceRequests: gpuOpts.Value()},
// 	}, nil, nil, "")
// 	if err != nil {
// 		panic(err)
// 	}
// 	if len(cbody.Warnings) > 0 {
// 		var msg []any
// 		for _, obj := range cbody.Warnings {
// 			msg = append(msg, obj)
// 		}
// 		logger.Warning(msg...)
// 	}
// 	fmt.Println(cbody.ID)

// 	if err := dockerClient.ContainerStart(ctx, cbody.ID, container.StartOptions{}); err != nil {
// 		panic(err)
// 	}

// 	out, err := dockerClient.ContainerLogs(ctx, cbody.ID, container.LogsOptions{
// 		ShowStdout: true,
// 		Follow:     true,
// 		Details:    true,
// 	})
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer out.Close()

// 	data, err := io.ReadAll(out)
// 	if err != nil {
// 		panic(err)
// 	}

// 	fmt.Println(string(data))
// }

// func TestDockerInfo(t *testing.T) {
// 	ctx := context.Background()

// 	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
// 	if err != nil {
// 		panic(err)
// 	}

// 	inspect, err := dockerClient.ContainerInspect(ctx, "5c97f54576bdc2558bf70f4a4e8c96057647542d4af71779b1098d7b4b3b419c")
// 	if err != nil {
// 		panic(err)
// 	}
// 	spew.Dump(inspect.HostConfig)
// }

// func TestDockerPrune(t *testing.T) {
// 	ctx := context.Background()

// 	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
// 	if err != nil {
// 		panic(err)
// 	}

// 	arg := filters.Arg("label", "run.sath.starter")
// 	report, err := dockerClient.ContainersPrune(ctx, filters.NewArgs(arg))
// 	if err != nil {
// 		panic(err)
// 	}
// 	spew.Dump(report)
// }

// func TestContainerList(t *testing.T) {
// 	ctx := context.Background()

// 	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
// 	if err != nil {
// 		panic(err)
// 	}

// 	filter := filters.NewArgs(filters.Arg("label", "run.sath.starter"))
// 	containers, err := dockerClient.ContainerList(ctx, container.ListOptions{
// 		Filters: filter,
// 	})
// 	if err != nil {
// 		panic(err)
// 	}
// 	spew.Dump(containers)
// }

// func TestVinaImage(t *testing.T) {
// 	ctx := context.Background()
// 	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
// 	checkErr(err)
// 	cmds := []string{
// 		"bin/qvina02",
// 		"--receptor", "/data/8HDO.pdbqt",
// 		"--ligand", "/data/SVR.pdbqt",
// 		"--center_x", "138.27592581289787",
// 		"--center_y", "127.76051810935691",
// 		"--center_z", "106.25829569498698",
// 		"--size_x", "18.5", "--size_y", "18.5", "--size_z", "18.5",
// 		"--out", "/output/out.txt",
// 		"--log", "/output/log.txt",
// 	}
// 	containerId, err := core.CreateContainer(ctx, dockerClient, cmds, "zengxinzhy/vina", "", "vina_test", []string{
// 		"/Users/xinzeng/Downloads/vina:/data",
// 		"/Users/xinzeng/Downloads/output:/output",
// 	})
// 	checkErr(err)
// 	fmt.Println("containerId: ", containerId)
// 	err = core.ExecImage(ctx, dockerClient, containerId, func(line string) {
// 		fmt.Println(line)
// 	})
// 	checkErr(err)
// }

// func TestImageName(t *testing.T) {
// 	ref, err := reference.ParseNormalizedNamed("10.101.12.128/ai/astronomy_algorithm_presto_algorithm:v1.0")
// 	checkErr(err)
// 	spew.Dump(reference.FamiliarName(ref))
// }

// // func TestMdImage(t *testing.T) {
// // 	ctx := context.Background()
// // 	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
// // 	if err != nil {
// // 		panic(err)
// // 	}
// // 	var containerId string
// // 	var dir = "/tmp/sath/sath_tmp_1605421609"
// // 	err = core.ExecImage(
// // 		ctx,
// // 		dockerClient,
// // 		[]string{"amber"},
// // 		"zengxinzhy/amber-runtime-cuda11.4.2:1.2",
// // 		dir,
// // 		dir,
// // 		"/data",
// // 		"all",
// // 		"container_name_test",
// // 		&containerId,
// // 		func(progress float64) {
// // 			log.Printf("progress: %f\n", progress)
// // 		},
// // 	)
// // 	if err != nil {
// // 		panic(err)
// // 	}
// // }

// // func TestSminaImage(t *testing.T) {
// // 	ctx := context.Background()
// // 	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
// // 	if err != nil {
// // 		panic(err)
// // 	}
// // 	var containerId string
// // 	var dir = "/tmp/sath/smina"
// // 	err = core.ExecImage(
// // 		ctx,
// // 		dockerClient,
// // 		[]string{"bash", "-c", "cd /data && /main --config config.txt"},
// // 		"zengxinzhy/smina:1.0",
// // 		dir,
// // 		dir,
// // 		"/data",
// // 		"",
// // 		"container_name_test",
// // 		&containerId,
// // 		func(progress float64) {
// // 			log.Printf("progress: %f\n", progress)
// // 		},
// // 	)
// // 	if err != nil {
// // 		panic(err)
// // 	}
// // }

// func TestCmd(t *testing.T) {
// 	ctx := context.Background()
// 	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
// 	checkErr(err)
// 	cmds, err := shlex.Split(`
// 		bash -c "python -c 'print(7*7)' > output/result.txt"
// 	`)
// 	checkErr(err)
// 	containerId, err := core.CreateContainer(ctx, dockerClient, cmds, "sathrun/base", "", "cmd_test", []string{
// 		"/tmp/sath/output:/output",
// 	})
// 	checkErr(err)
// 	fmt.Println("containerId: ", containerId)
// 	err = core.ExecImage(ctx, dockerClient, containerId, func(line string) {
// 		fmt.Println(line)
// 	})
// 	checkErr(err)
// }

// func TestDockerAttach(t *testing.T) {
// 	ctx := context.Background()
// 	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
// 	checkErr(err)
// 	containerId, err := core.CreateContainer(ctx, dockerClient, nil, "sathrun/base", "", "cmd_test", []string{"/tmp/data:/data"})
// 	// err = dockerClient.ContainerStart(ctx, containerId, container.StartOptions{})
// 	checkErr(err)

// 	defer func() {
// 		// assign a new background to ctx to make sure the following code still works
// 		// in case the original ctx was cancelled
// 		c := context.Background()
// 		if err := dockerClient.ContainerStop(c, containerId, container.StopOptions{}); err != nil {
// 			logger.Error(errors.WithStack(err))
// 			return
// 		}
// 		if err := dockerClient.ContainerRemove(c, containerId, container.RemoveOptions{
// 			RemoveVolumes: true,
// 			Force:         true,
// 		}); err != nil {
// 			logger.Error(errors.WithStack(err))
// 			return
// 		}
// 	}()

// 	err = dockerClient.ContainerStart(ctx, containerId, container.StartOptions{})
// 	checkErr(err)
// 	res, err := dockerClient.ContainerExecCreate(ctx, containerId, container.ExecOptions{
// 		AttachStderr: true,
// 		AttachStdout: true,
// 		Cmd:          []string{"echo", "Hi Tian"},
// 	})
// 	checkErr(err)

// 	hijack, err := dockerClient.ContainerExecAttach(ctx, res.ID, container.ExecStartOptions{
// 		Tty: true,
// 	})
// 	checkErr(err)
// 	err = dockerClient.ContainerExecStart(ctx, res.ID, container.ExecStartOptions{})
// 	checkErr(err)

// 	defer hijack.Close()
// 	scanner := bufio.NewScanner(hijack.Reader)
// 	for scanner.Scan() {
// 		fmt.Println(scanner.Text())
// 	}
// }
