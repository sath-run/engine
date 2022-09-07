package action

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	pb "github.com/sath-run/engine/pkg/protobuf"
)

type JobExecResult struct {
	JobId string
}

func processInputs(dir string, job *pb.JobGetResponse) error {
	files := job.GetFiles()
	for _, file := range files {
		if uri := file.GetRemote(); uri != nil {
			// TOOD
		} else if data := file.GetData(); len(data) > 0 {
			if err := os.WriteFile(filepath.Join(dir, file.Name), data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

func processOutputs(dir string, job *pb.JobGetResponse) ([]byte, error) {
	var (
		data []byte
		err  error
	)

	if len(job.Outputs) == 0 {
		// TODO
	} else if len(job.Outputs) > 1 {
		// TODO
	} else {
		data, err = os.ReadFile(filepath.Join(dir, job.Outputs[0]))
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

func RunSingleJob() (JobExecResult, error) {
	ctx := context.Background()
	res := JobExecResult{}

	// TODO
	job, err := grpcClient.GetNewJob(ctx, &pb.JobGetRequest{
		UserId:     "",
		DeviceId:   "",
		DeviceInfo: "",
	})

	if err != nil {
		return res, err
	}

	if job == nil || len(job.JobId) == 0 {
		return res, nil
	}
	res.JobId = job.JobId

	dir, err := os.MkdirTemp("", "sath_tmp_*")
	if err != nil {
		return res, err
	}
	defer func() {
		err = os.RemoveAll(dir)
	}()

	if err := PullImage(ctx, job.Image.Id, job.Image.Tag, job.Image.Uri); err != nil {
		return res, err
	}

	if err := processInputs(dir, job); err != nil {
		return res, err
	}

	if err = ExecImage(ctx, job.Cmds, job.Image.Tag, dir, job.VolumePath, func(text string) {
		fmt.Println(text)
	}); err != nil {
		return res, err
	}

	if data, err := os.ReadFile(filepath.Join(dir, "sath_stderr.log")); err == os.ErrNotExist {
		// nothing to do
	} else if err != nil {
		return res, err
	} else if len(data) > 0 {
		return res, errors.New(string(data))
	}

	data, err := processOutputs(dir, job)
	if err != nil {
		return res, err
	}

	_, err = grpcClient.PopulateJobResult(ctx, &pb.JobPopulateRequest{
		JobId:    job.JobId,
		UserId:   "",
		DeviceId: "",
		Result:   data,
	})

	if err != nil {
		return res, err
	}

	return res, nil
}
