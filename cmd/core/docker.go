package core

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/hpcloud/tail"
	"github.com/pkg/errors"
	"github.com/sath-run/engine/cmd/utils"
)

type DockerImageResponse struct {
	Format  string                 `json:"format"`
	Version string                 `json:"version"`
	Type    string                 `json:"type"`
	Data    map[string]interface{} `json:"data"`
}

type ProgressData_V1 struct {
	Progress float64 `json:"progress"`
}

type DockerImageConfig struct {
	Repository string
	Digest     string
	Tag        string
	Uri        string
}

func (config *DockerImageConfig) Image() string {
	image := config.Repository
	if config.Tag != "" {
		image += ":" + config.Tag
	}
	if config.Digest != "" {
		image += "@" + config.Digest
	}
	return image
}

func PullImage(ctx context.Context, dockerClient *client.Client, config *DockerImageConfig, onProgress func(text string)) error {
	// look for local images to see if any mathces given id
	images, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return errors.WithStack(err)
	}

	for _, image := range images {
		// first try to find a image with full name match
		for _, repoDigest := range image.RepoDigests {
			if repoDigest == config.Repository+":"+config.Tag+"@"+config.Digest {
				return nil
			}
		}

		// if not found, find image by digest
		for _, repoDigest := range image.RepoDigests {
			if repoDigest == config.Digest {
				return nil
			}
		}

		// if still not found, find image by tag
		if len(config.Digest) == 0 {
			for _, tag := range image.RepoTags {
				if tag == config.Image() {
					return nil
				}
			}
		}
	}

	uri := config.Uri
	if uri == "" {
		uri = config.Image()
	}

	// pull image from remote
	reader, err := dockerClient.ImagePull(context.Background(), uri, types.ImagePullOptions{})
	if err != nil {
		return errors.WithStack(err)
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		onProgress(line)
	}

	return nil
}

func ExecImage(
	ctx context.Context,
	client *client.Client,
	cmds []string,
	image string,
	localDataDir string,
	hostDir string,
	volumePath string,
	gpuOpts string,
	containerId *string,
	onProgress func(progress float64)) error {

	fmt.Println(cmds, image, localDataDir, hostDir, volumePath, gpuOpts)

	gpuOptsVal := opts.GpuOpts{}
	gpuOptsVal.Set(gpuOpts)

	cbody, err := client.ContainerCreate(ctx, &container.Config{
		Cmd:   cmds,
		Image: image,
		Tty:   true,
		Labels: map[string]string{
			"run.sath.starter": os.Getenv("HOSTNAME"),
		},
	}, &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:%s", hostDir, volumePath),
		},
		Resources: container.Resources{DeviceRequests: gpuOptsVal.Value()},
	}, nil, nil, "")
	if err != nil {
		return errors.WithStack(err)
	}
	if len(cbody.Warnings) > 0 {
		utils.LogWarning(cbody.Warnings...)
	}
	*containerId = cbody.ID

	defer func() {
		// assign a new background to ctx to make sure the following code still works
		// in case the original ctx was cancelled
		ctx = context.Background()
		if err = client.ContainerStop(ctx, cbody.ID, nil); err != nil {
			utils.LogError(errors.WithStack(err))
			return
		}
		if err = client.ContainerRemove(ctx, cbody.ID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
			utils.LogError(errors.WithStack(err))
			return
		}
	}()

	if err := client.ContainerStart(ctx, cbody.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	out, err := client.ContainerLogs(ctx, cbody.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		Follow:     true,
		Details:    true,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	defer out.Close()

	tails, err := tail.TailFile(path.Join(localDataDir, "sath.log"), tail.Config{Follow: true, Logger: tail.DiscardingLogger})
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		tails.Stop()
		tails.Cleanup()
	}()
	go func() {
		for line := range tails.Lines {
			var res DockerImageResponse

			if err := json.Unmarshal([]byte(line.Text), &res); err != nil {
				utils.LogError(errors.WithStack(err))
				continue
			}

			if res.Format == "sath" && res.Type == "progress" {
				var progress ProgressData_V1
				data, err := json.Marshal(res.Data)
				if err != nil {
					utils.LogError(errors.WithStack(err))
					continue
				}
				if err := json.Unmarshal(data, &progress); err != nil {
					utils.LogError(errors.WithStack(err))
					continue
				}
				onProgress(progress.Progress)
			}
		}
	}()

	stdout, err := os.OpenFile(path.Join(localDataDir, "sath.out"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return errors.WithStack(err)
	}
	defer stdout.Close()

	_, err = io.Copy(stdout, out)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func StopCurrentRunningContainers(client *client.Client) error {
	ctx := context.Background()
	filter := filters.NewArgs(filters.Arg("label", "run.sath.starter"))
	containers, err := client.ContainerList(ctx, types.ContainerListOptions{
		Filters: filter,
	})
	if err != nil {
		return err
	}
	for _, container := range containers {
		if err := client.ContainerStop(ctx, container.ID, nil); err != nil {
			return err
		}
	}
	return nil
}
