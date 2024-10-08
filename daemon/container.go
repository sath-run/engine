package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	pb "github.com/sath-run/engine/daemon/protobuf"
)

type Container struct {
	id         string
	cli        *client.Client
	imageUrl   string
	imageAuth  string
	dir        string
	gpuOpt     pb.GpuOpt
	vram       uint64
	currentJob *Job
	binds      []string
	logger     zerolog.Logger
	resourceId string
}

func newContainer(dockerCli *client.Client, dir string, job *Job) *Container {
	ctn := &Container{
		cli:        dockerCli,
		imageUrl:   job.metadata.Image.Url,
		imageAuth:  job.metadata.Image.Auth,
		currentJob: job,
		dir:        dir,
		gpuOpt:     job.metadata.GpuConf.Opt,
		vram:       0, // TODO
		resourceId: job.metadata.ResourceId,
	}

	for _, v := range []string{"data", "source", "output", "resource"} {
		var (
			src      string
			dst      string
			bind     string
			readonly bool
		)
		if v == "resource" {
			src = job.resourceDir
			readonly = true
		} else {
			src = filepath.Join(ctn.dir, v)
			readonly = false
		}
		dst = job.metadata.Image.Binds[v]
		if dst == "" {
			dst = "/" + v
		}
		os.Mkdir(src, os.ModePerm)
		if readonly {
			bind = fmt.Sprintf("%s:%s:ro", src, dst)
		} else {
			bind = fmt.Sprintf("%s:%s", src, dst)
		}
		ctn.binds = append(ctn.binds, bind)
	}
	return ctn
}

func (ctn *Container) init(ctx context.Context) error {
	gpuOptsVal := opts.GpuOpts{}
	if ctn.gpuOpt != pb.GpuOpt_EGO_None {
		gpuOptsVal.Set("all")
	}
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		hostname, _ = os.Hostname()
	}
	cbody, err := ctn.cli.ContainerCreate(ctx, &container.Config{
		Image: ctn.imageUrl,
		Tty:   true,
		Labels: map[string]string{
			"run.sath.starter": hostname,
		},
		AttachStdout: true,
		AttachStderr: true,
	}, &container.HostConfig{
		Binds: ctn.binds,
		Resources: container.Resources{
			DeviceRequests: gpuOptsVal.Value(),
		},
	},
		nil,
		nil,
		"",
	)
	if err != nil {
		return err
	}
	ctn.id = cbody.ID
	ctn.logger = log.With().Str("container", ctn.id).Logger()
	for _, warn := range cbody.Warnings {
		ctn.logger.Warn().Msg(warn)
	}

	if err := ctn.cli.ContainerStart(ctx, ctn.id, container.StartOptions{}); err != nil {
		return err
	}

	// TODO: download files for sources if specified

	return nil
}

func (ctn *Container) run(ctx context.Context, cmd []string) (*types.HijackedResponse, error) {
	res, err := ctn.cli.ContainerExecCreate(ctx, ctn.id, container.ExecOptions{
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          cmd,
	})
	if err != nil {
		return nil, err
	}

	hijack, err := ctn.cli.ContainerExecAttach(ctx, res.ID, container.ExecStartOptions{
		Tty: true,
	})
	if err != nil {
		return nil, err
	}
	return &hijack, nil
}

func (ctn *Container) dataDir() string {
	return filepath.Join(ctn.dir, "data")
}

func (ctn *Container) outputDir() string {
	return filepath.Join(ctn.dir, "output")
}

func stopCurrentRunningContainers(ctx context.Context, client *client.Client) error {
	filter := filters.NewArgs(filters.Arg("label", "run.sath.starter"))
	containers, err := client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filter,
	})
	if err != nil {
		return err
	}
	for _, c := range containers {
		if err := client.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true}); err != nil {
			return err
		}
	}
	return nil
}
