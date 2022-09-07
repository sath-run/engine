package action

import (
	"bufio"
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

func PullImage(ctx context.Context, id string, tag string, uri string) error {

	// look for local images to see if any mathces given id
	images, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return err
	}
	for _, image := range images {
		if image.ID == id {
			return nil
		}
	}
	for _, image := range images {
		for _, repoTag := range image.RepoTags {
			if repoTag == tag {
				return nil
			}
		}
	}

	// pull image from remote
	reader, err := dockerClient.ImagePull(context.Background(), uri, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf[:])
		if n > 0 {
			fmt.Println(string(buf[:n]))
		}
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			} else {
				return err
			}
			break
		}
	}

	return nil
}

func ExecImage(
	ctx context.Context,
	cmds []string,
	image string,
	dir string,
	volumePath string,
	onProgress func(text string)) error {

	cbody, err := dockerClient.ContainerCreate(ctx, &container.Config{
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
		if err = dockerClient.ContainerStop(ctx, cbody.ID, nil); err != nil {
			return
		}
		if err = dockerClient.ContainerRemove(ctx, cbody.ID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
			return
		}
	}()

	if err := dockerClient.ContainerStart(ctx, cbody.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	out, err := dockerClient.ContainerLogs(ctx, cbody.ID, types.ContainerLogsOptions{
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
		onProgress(line)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
