package core

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
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

func GetCurrentContainerId() (string, error) {
	data, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return "", err
	}
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		line := sc.Text()
		components := strings.Fields(line)
		if len(components) >= 5 && components[4] == "/etc/hostname" {
			parts := strings.Split(components[3], "/")
			if len(parts) > 4 {
				return parts[3], nil
			}
		}
	}
	return "", fmt.Errorf("can't find container id from mountinfo:\n%s", string(data))
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

func PullImage(ctx context.Context, dockerClient *client.Client, url string, onProgress func(text string)) error {
	// look for local images to see if any mathces given id
	// images, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
	// if err != nil {
	// 	return errors.WithStack(err)
	// }

	// for _, image := range images {
	// 	// first try to find a image with full name match
	// 	for _, repoDigest := range image.RepoDigests {
	// 		if repoDigest == config.Repository+":"+config.Tag+"@"+config.Digest {
	// 			return nil
	// 		}
	// 	}

	// 	// if not found, find image by digest
	// 	for _, repoDigest := range image.RepoDigests {
	// 		if repoDigest == config.Digest {
	// 			return nil
	// 		}
	// 	}

	// 	// if still not found, find image by tag
	// 	if len(config.Digest) == 0 {
	// 		for _, tag := range image.RepoTags {
	// 			if tag == config.Image() {
	// 				return nil
	// 			}
	// 		}
	// 	}
	// }

	// uri := config.Uri
	// if uri == "" {
	// 	uri = config.Image()
	// }

	// pull image from remote
	reader, err := dockerClient.ImagePull(context.Background(), url, types.ImagePullOptions{})
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

func CreateContainer(
	ctx context.Context,
	client *client.Client,
	cmd []string,
	image string,
	gpuOpts string,
	containerName string,
	binds []string) (string, error) {
	gpuOptsVal := opts.GpuOpts{}
	gpuOptsVal.Set(gpuOpts)
	cbody, err := client.ContainerCreate(ctx, &container.Config{
		Cmd:   cmd,
		Image: image,
		Tty:   true,
		Labels: map[string]string{
			"run.sath.starter": os.Getenv("HOSTNAME"),
		},
	}, &container.HostConfig{
		Binds: binds,
		Resources: container.Resources{
			CPUQuota:       64 * 100000, // 64 cores maximum
			DeviceRequests: gpuOptsVal.Value(),
		},
	},
		nil,
		nil,
		containerName,
	)
	if err != nil {
		return "", errors.WithStack(err)
	}
	if len(cbody.Warnings) > 0 {
		var msg []any
		for _, obj := range cbody.Warnings {
			msg = append(msg, obj)
		}
		utils.LogWarning(msg...)
	}
	return cbody.ID, nil
}

func ExecImage(
	ctx context.Context,
	client *client.Client,
	containerId string,
	onProgress func(line string)) error {

	defer func() {
		// assign a new background to ctx to make sure the following code still works
		// in case the original ctx was cancelled
		c := context.Background()
		if err := client.ContainerStop(c, containerId, nil); err != nil {
			utils.LogError(errors.WithStack(err))
			return
		}
		if err := client.ContainerRemove(c, containerId, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
			utils.LogError(errors.WithStack(err))
			return
		}
	}()

	if err := client.ContainerStart(ctx, containerId, types.ContainerStartOptions{}); err != nil {
		return err
	}

	out, err := client.ContainerLogs(ctx, containerId, types.ContainerLogsOptions{
		ShowStdout: true,
		Follow:     true,
		Details:    true,
	})
	if err != nil {
		return err
	}
	defer out.Close()

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		onProgress(scanner.Text())
	}
	return ctx.Err()
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
