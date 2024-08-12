package scheduler

import (
	"context"
	"errors"
	"time"

	"github.com/docker/docker/client"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sath-run/engine/engine/core/conns"
	pb "github.com/sath-run/engine/engine/core/protobuf"
)

type Job struct {
	c         *conns.Connection
	docker    *client.Client
	metadata  *pb.JobGetResponse
	container *Container

	dir       string
	stream    pb.Engine_NotifyExecStatusClient
	err       error
	queue     chan *Job
	state     pb.EnumExecState
	createdAt time.Time
	outputs   []JobOutput

	logger zerolog.Logger

	// ExecId      string
	// ContainerId string
	// Progress    float64
	// Message     string
	// Paused      bool
	// CompletedAt time.Time
	// UpdatedAt   time.Time

	// Outputs []*pb.ExecOutput
}

type JobNotification struct {
	Message  string
	Progress float64
	Flag     uint64
	Outputs  []JobOutput
}

type JobOutput struct {
	Id      string
	Status  pb.ExecOutputStatus
	Message string
	Content []byte
}

func newJob(ctx context.Context, c *conns.Connection, docker *client.Client, queue chan *Job, dir string, metadata *pb.JobGetResponse) (*Job, error) {
	stream, err := c.NotifyExecStatus(ctx)
	if err != nil {
		return nil, err
	}
	job := &Job{
		c:      c,
		docker: docker,
		// ExecId:    metadata.ExecId,
		metadata:  metadata,
		state:     pb.EnumExecState_EES_INITIALIZED,
		createdAt: time.Now(),
		stream:    stream,
		queue:     queue,
		dir:       dir,
		logger:    log.With().Str("job", metadata.ExecId).Logger(),
	}
	return job, nil
}

func (job *Job) notifyStatusToRemote(notification JobNotification) error {
	req := pb.ExecNotificationRequest{
		State:    job.state,
		Message:  notification.Message,
		Progress: notification.Progress,
		Flag:     notification.Flag,
	}
	for _, output := range notification.Outputs {
		req.Outputs = append(req.Outputs, &pb.ExecOutput{
			Id:      output.Id,
			Status:  output.Status,
			Message: output.Message,
			Content: output.Content,
		})
	}
	return job.stream.Send(&req)
}

func (job *Job) handleCompletion() {
	var notification JobNotification
	if job.err == nil {
		notification = JobNotification{
			Progress: 1.0,
			Outputs:  job.outputs,
		}
	} else {
		notification = JobNotification{
			Message: job.err.Error(),
		}
	}
	if err := job.notifyStatusToRemote(notification); err != nil {
		job.err = errors.Join(job.err, err)
	}

	// block and wait to close stream, error could be ignored
	job.stream.CloseAndRecv()
}

func (job *Job) preprocess() {
	var err error
	defer func() {
		if err != nil {
			job.err = err
			job.handleCompletion()
		}
		// notify scheduler
		job.queue <- job
	}()
	if err = job.prepareImage(); err != nil {
		return
	}
	if err = job.downloadInputs(); err != nil {
		return
	}
	if err = job.processInputs(); err != nil {
		return
	}
	job.logger.Trace().Msg("queuing")
	job.state = pb.EnumExecState_EES_QUEUING
}

func (job *Job) run() {
	var err error
	defer func() {
		job.err = err
		job.handleCompletion()
		// notify scheduler
		job.queue <- job
	}()
	if err = job.prepareContainer(); err != nil {
		return
	}
	if err = job.runTask(); err != nil {
		return
	}
	if err = job.processOutputs(); err != nil {
		return
	}
	job.state = pb.EnumExecState_EES_SUCCESS
}

func (job *Job) prepareImage() error {
	job.state = pb.EnumExecState_EES_PREPARE_IMAGE
	job.logger.Trace().Msg("prepareImage")
	return nil
}

func (job *Job) downloadInputs() error {
	job.state = pb.EnumExecState_EES_DOWNLOADING_INPUTS
	job.logger.Trace().Msg("downloadInputs")
	return nil
}

func (job *Job) processInputs() error {
	job.state = pb.EnumExecState_EES_PROCESSING_INPUTS
	job.logger.Trace().Msg("precessInputs")
	return nil
}

func (job *Job) prepareContainer() error {
	job.state = pb.EnumExecState_EES_PREPARE_IMAGE
	job.logger.Trace().Msg("prepareContainer")
	return nil
}

func (job *Job) runTask() error {
	job.state = pb.EnumExecState_EES_RUNNING
	job.logger.Trace().Msg("runTask")
	return nil
}

func (job *Job) processOutputs() error {
	job.state = pb.EnumExecState_EES_PROCESSING_OUPUTS
	job.logger.Trace().Msg("processOutputs")
	return nil
}

// ******************************************************************

// func (job *Job) updateStatus(status pb.EnumExecState, message string, progress float64) {
// 	// if job.err != nil && status != pb.EnumExecState_EES_ERROR {
// 	// 	return
// 	// }

// 	job.Status = status
// 	job.Message = message
// 	job.Progress = progress
// 	job.UpdatedAt = time.Now()

// 	// notify status to server
// 	req := &pb.ExecNotificationRequest{
// 		Status:   job.Status,
// 		Message:  job.Message,
// 		Progress: job.Progress,
// 	}
// 	if status == pb.EnumExecState_EES_SUCCESS {
// 		req.Outputs = job.Outputs
// 	}
// 	// if status.GpuOpts != "" {
// 	// 	req.GpuStats = []*pb.GpuStats{
// 	// 		{Id: 0},
// 	// 	}
// 	// }
// 	var err error
// 	if err = job.stream.Send(req); err != nil {
// 		job.handleError(err)
// 		return
// 	}
// }

// func (job *Job) handleError(err error) {
// 	if job.err != nil {
// 		return
// 	}
// 	job.err = err
// 	job.updateStatus(pb.EnumExecState_EES_ERROR, err.Error(), 0)
// }

// func (job *Job) notifyScheduler() {
// 	job.queue <- job
// }

// func (job *Job) processInputs() {
// 	defer job.notifyScheduler()

// 	files := job.metadata.Inputs
// 	for i, file := range files {
// 		job.updateStatus(
// 			pb.EnumExecState_EES_DOWNLOADING_INPUTS,
// 			fmt.Sprintf("start download %s", file.Path),
// 			float64(i+1)/float64(len(files)),
// 		)
// 		if job.err != nil {
// 			break
// 		}
// 		filePath := filepath.Join(job.dir, file.Path)
// 		err := func() error {
// 			out, err := os.Create(filePath)
// 			if err != nil {
// 				return err
// 			}
// 			defer out.Close()

// 			resp, err := retryablehttp.Get(file.Path)
// 			if err != nil {
// 				return err
// 			}
// 			defer resp.Body.Close()

// 			_, err = io.Copy(out, resp.Body)
// 			if err != nil {
// 				return err
// 			}
// 			return nil
// 		}()
// 		if err != nil {
// 			job.handleError(err)
// 			break
// 		}
// 	}
// 	if job.err == nil {
// 		job.updateStatus(pb.EnumExecState_EES_QUEUING, "", 0)
// 	}
// }

// func (job *Job) run() {
// 	job.Status = pb.EnumExecState_EES_PREPARE_IMAGE
// 	defer job.notifyScheduler()

// 	// if len(ctn.containerId) == 0 {
// 	// 	// prepare and start docker container
// 	// 	dir := ctn.dir
// 	// 	if err := os.MkdirAll(filepath.Join(dir, "/data"), os.ModePerm); err != nil {
// 	// 		job.handleError(err)
// 	// 		return
// 	// 	}
// 	// 	if err := os.MkdirAll(filepath.Join(dir, "/output"), os.ModePerm); err != nil {
// 	// 		job.handleError(err)
// 	// 		return
// 	// 	}
// 	// 	if err := os.MkdirAll(filepath.Join(dir, "/source"), os.ModePerm); err != nil {
// 	// 		job.handleError(err)
// 	// 		return
// 	// 	}

// 	// 	if err := PullImage(ctx, scheduler.docker, job.Res.Image.Url, image.PullOptions{
// 	// 		RegistryAuth: job.Res.Image.Auth,
// 	// 	}, func(text string) {
// 	// 		job.updateStatus(pb.EnumExecState_EES_PULLING_IMAGE, text, 0)
// 	// 		if job.err != nil {
// 	// 			return
// 	// 		}
// 	// 	}); err != nil {
// 	// 		job.handleError(err)
// 	// 		return
// 	// 	}

// 	// 	job.updateStatus(pb.EnumExecState_EES_START_CONTAINER, "", 0)
// 	// 	if job.err != nil {
// 	// 		return
// 	// 	}

// 	// 	// create container
// 	// 	binds := []string{
// 	// 		fmt.Sprintf("%s:%s", filepath.Join(dir, "/data"), job.Res.Volume.Data),
// 	// 		fmt.Sprintf("%s:%s", filepath.Join(dir, "/source"), job.Res.Volume.Source),
// 	// 		fmt.Sprintf("%s:%s", filepath.Join(dir, "/output"), job.Res.Volume.Output),
// 	// 	}

// 	// 	containerName := fmt.Sprintf("sath-%s", job.ExecId)

// 	// 	// TODO: better strategy for GPU allocation
// 	// 	gpuOpts := ""
// 	// 	spew.Dump(job.Res)
// 	// 	if job.Res.GpuConf.Opt == pb.GpuOpt_EGO_REQUIRED || job.Res.GpuConf.Opt == pb.GpuOpt_EGO_PREFERRED {
// 	// 		gpuOpts = "all"
// 	// 	}
// 	// 	containerId, err := CreateContainer(
// 	// 		ctx, scheduler.docker, nil, job.Res.Image.Url,
// 	// 		gpuOpts, containerName, binds)
// 	// 	if err != nil {
// 	// 		job.handleError(err)
// 	// 		return
// 	// 	}
// 	// 	ctn.containerId = containerId
// 	// 	err = scheduler.docker.ContainerStart(ctx, containerId, container.StartOptions{})
// 	// 	if err != nil {
// 	// 		job.handleError(err)
// 	// 		return
// 	// 	}
// 	// }

// 	// move input files to volume
// 	if err := exec.Command("rm", "-r", filepath.Join(ctn.dir, "/data", "*")).Run(); err != nil {
// 		job.handleError(err)
// 		return
// 	}
// 	if err := exec.Command("rm", "-r", filepath.Join(ctn.dir, "/output", "*")).Run(); err != nil {
// 		job.handleError(err)
// 		return
// 	}
// 	if err := exec.Command("mv", "-r", filepath.Join(job.dir, "*"), filepath.Join(ctn.dir, "/data")).Run(); err != nil {
// 		job.handleError(err)
// 		return
// 	}
// 	os.Remove(job.dir)

// 	// start job exec
// 	res, err := scheduler.docker.ContainerExecCreate(ctx, ctn.containerId, container.ExecOptions{
// 		AttachStderr: true,
// 		AttachStdout: true,
// 		Cmd:          job.Res.Cmd,
// 	})
// 	if err != nil {
// 		job.handleError(err)
// 		return
// 	}

// 	hijack, err := scheduler.docker.ContainerExecAttach(ctx, res.ID, container.ExecStartOptions{
// 		Tty: true,
// 	})
// 	if err != nil {
// 		job.handleError(err)
// 		return
// 	}
// 	defer hijack.Close()

// 	scanner := bufio.NewScanner(hijack.Reader)
// 	for scanner.Scan() {
// 		// TODO: update progress
// 		job.updateStatus(pb.EnumExecState_EES_RUNNING, scanner.Text(), 0)
// 		if job.err != nil {
// 			return
// 		}
// 	}

// 	job.processOutputs(filepath.Join(ctn.dir, "/output", "*"))
// 	if job.err != nil {
// 		return
// 	}

// 	job.updateStatus(pb.EnumExecState_EES_SUCCESS, "", 100)
// 	if job.err != nil {
// 		return
// 	}
// }

// func (job *Job) processOutputs(outputDir string) {
// 	outputs := make([]*pb.ExecOutput, len(job.Res.Outputs))
// 	job.Outputs = outputs
// 	for i, output := range job.Res.Outputs {
// 		job.updateStatus(pb.EnumExecState_EES_PROCESSING_OUPUTS, "", float64(i)/float64(len(job.Res.Outputs)))
// 		if job.err != nil {
// 			return
// 		}
// 		outputs[i] = &pb.ExecOutput{
// 			Id:     output.Id,
// 			Status: pb.ExecOutputStatus_EOS_SUCCESS,
// 		}
// 		path := filepath.Join(outputDir, output.Path)
// 		data, err := os.Open(path)
// 		if err != nil {
// 			logger.Debug("file not found:", path)
// 			continue
// 		}
// 		defer data.Close()

// 		if output.Req == nil {
// 			// if output request is not specified, return file content
// 			fs, err := data.Stat()
// 			if err != nil {
// 				job.handleError(err)
// 				return
// 			}
// 			fileSizeInMB := float64(fs.Size()) / 1024 / 1024
// 			if fileSizeInMB > 1 {
// 				err = fmt.Errorf(
// 					"file %s is too large, size limit is 1MB, actual size is %.2fMB",
// 					output.Path,
// 					fileSizeInMB)
// 				job.handleError(err)
// 				return
// 			}
// 			bytes, err := io.ReadAll(data)
// 			if err != nil {
// 				job.handleError(err)
// 				return
// 			}
// 			outputs[i].Content = bytes
// 		} else {
// 			// upload output file to url
// 			req, err := retryablehttp.NewRequest(output.Req.Method, output.Req.Url, data)
// 			if err != nil {
// 				job.handleError(err)
// 				return
// 			}
// 			for _, header := range output.Req.Headers {
// 				req.Header.Set(header.Name, header.Value)
// 			}
// 			resp, err := retryablehttp.NewClient().Do(req)
// 			if err != nil {
// 				job.handleError(err)
// 				return
// 			} else if resp.StatusCode < 200 || resp.StatusCode >= 400 {
// 				data, _ := io.ReadAll(resp.Body)
// 				resp.Body.Close()
// 				err = fmt.Errorf("fail to upload data, stats: %d, data: %s", resp.StatusCode, string(data))
// 				job.handleError(err)
// 				return
// 			}
// 		}
// 	}
// }

// func JobStatusText(enum pb.EnumExecState) string {
// 	switch enum {
// 	case pb.EnumExecState_EES_INITIALIZED:
// 		return "default"
// 	case pb.EnumExecState_EES_PULLING_IMAGE:
// 		return "pulling-image"
// 	case pb.EnumExecState_EES_DOWNLOADING_INPUTS:
// 		return "downloading"
// 	case pb.EnumExecState_EES_PROCESSING_INPUTS:
// 		return "preprocessing"
// 	case pb.EnumExecState_EES_RUNNING:
// 		return "running"
// 	case pb.EnumExecState_EES_PROCESSING_OUPUTS:
// 		return "postprocessing"
// 	case pb.EnumExecState_EES_SUCCESS:
// 		return "success"
// 	case pb.EnumExecState_EES_CANCELED:
// 		return "cancelled"
// 	case pb.EnumExecState_EES_ERROR:
// 		return "error"
// 	default:
// 		return "unspecified"
// 	}
// }
