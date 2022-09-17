package core

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
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
		return err
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
		return err
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
		return err
	}

	defer func() {
		if err = g.dockerClient.ContainerStop(ctx, cbody.ID, nil); err != nil {
			return
		}
		if err = g.dockerClient.ContainerRemove(ctx, cbody.ID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
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
		return err
	}
	defer out.Close()

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		line := scanner.Text()

		var res DockerImageResponse

		if err := json.Unmarshal([]byte(line), &res); err != nil {
			log.Printf("%+v\n", err)
			continue
		}

		if res.Format == "sath" && res.Type == "progress" {
			var progress ProgressData_V1
			data, err := json.Marshal(res.Data)
			if err != nil {
				log.Printf("%+v\n", err)
				continue
			}
			if err := json.Unmarshal(data, &progress); err != nil {
				log.Printf("%+v\n", err)
				continue
			}
			onProgress(progress.Progress)
		}

	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}