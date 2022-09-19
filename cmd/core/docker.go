package core

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
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
	if config.Digest != "" {
		image += "@" + config.Digest
	}
	return image
}

func PullImage(ctx context.Context, config *DockerImageConfig, onProgress func(text string)) error {
	// look for local images to see if any mathces given id
	images, err := g.dockerClient.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return errors.WithStack(err)
	}

	for _, image := range images {
		for _, repoDigest := range image.RepoDigests {
			if repoDigest == config.Repository+":"+config.Tag+"@"+config.Digest ||
				repoDigest == config.Image() {
				return nil
			}
		}
	}

	uri := config.Uri
	if uri == "" {
		uri = config.Image()
	}

	// pull image from remote
	reader, err := g.dockerClient.ImagePull(context.Background(), uri, types.ImagePullOptions{})
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
	cmds []string,
	image string,
	dir string,
	volumePath string,
	onProgress func(progress float64)) error {

	cbody, err := g.dockerClient.ContainerCreate(ctx, &container.Config{
		Cmd:   cmds,
		Image: image,
		Tty:   true,
	}, &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:%s", dir, volumePath),
		},
	}, nil, nil, "")
	if err != nil {
		return errors.WithStack(err)
	}

	defer func() {
		// assign a new background to ctx to make sure the following code still works
		// in case the original ctx was cancelled
		ctx = context.Background()
		if err = g.dockerClient.ContainerStop(ctx, cbody.ID, nil); err != nil {
			utils.LogError(errors.WithStack(err))
			return
		}
		if err = g.dockerClient.ContainerRemove(ctx, cbody.ID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
			utils.LogError(errors.WithStack(err))
			return
		}
	}()

	if err := g.dockerClient.ContainerStart(ctx, cbody.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	out, err := g.dockerClient.ContainerLogs(ctx, cbody.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		Follow:     true,
		Details:    true,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	defer out.Close()

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		line := scanner.Text()

		var res DockerImageResponse

		if err := json.Unmarshal([]byte(line), &res); err != nil {
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

	if err := scanner.Err(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
