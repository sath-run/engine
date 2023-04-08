package core_test

import (
	"context"
	"log"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/sath-run/engine/cmd/core"
	pb "github.com/sath-run/engine/pkg/protobuf"
)

func TestFileUpload(t *testing.T) {
	err := core.Init(&core.Config{
		GrpcAddress: "localhost:50051",
		SSL:         false,
	})
	if err != nil {
		log.Fatal(err)
	}
	files, err := core.ProcessOutputs(".", "587962", []string{"docker.go"})
	if err != nil {
		log.Fatal(err)
	}
	spew.Dump(files)
}

func TestMdJob(t *testing.T) {
	err := core.Init(&core.Config{
		GrpcAddress: "scheduler.sath.run:50051",
		SSL:         true,
		DataDir:     "/tmp/sath",
	})
	if err != nil {
		panic(err)
	}
	job := &pb.JobGetResponse{
		Image: &pb.DockerImage{
			Repository: "zengxinzhy/amber-runtime-cuda11.4.2",
			Tag:        "latest",
		},
		Cmds: []string{
			"amber",
		},
		VolumePath: "/data",
		GpuOpts:    "all",
		Files: []*pb.File{
			{
				Name: "equi.rst",
				Content: &pb.File_Remote{
					Remote: &pb.FileUri{
						Uri:         "https://sath-ligand.s3.ap-east-1.amazonaws.com/md/equi.rst",
						FetchMethod: pb.EnumFileFetchMethod_EFFM_HTTP,
					},
				},
			},
			{
				Name: "prod.in",
				Content: &pb.File_Remote{
					Remote: &pb.FileUri{
						Uri:         "https://sath-ligand.s3.ap-east-1.amazonaws.com/md/prod.in",
						FetchMethod: pb.EnumFileFetchMethod_EFFM_HTTP,
					},
				},
			},
			{
				Name: "ras-raf_solvated.prmtop",
				Content: &pb.File_Remote{
					Remote: &pb.FileUri{
						Uri:         "https://sath-ligand.s3.ap-east-1.amazonaws.com/md/ras-raf_solvated.prmtop",
						FetchMethod: pb.EnumFileFetchMethod_EFFM_HTTP,
					},
				},
			},
		},
		Outputs: []string{},
	}
	// var containerId string
	// if err := core.ExecImage(
	// 	context.Background(), core.GetDockerClient(), job.Cmds, "zengxinzhy/amber-runtime-cuda11.4.2:1.2", "/tmp/sath/sath_tmp_1040899213", "/tmp/sath/sath_tmp_1040899213", job.VolumePath,
	// 	job.GpuOpts, &containerId, func(progress float64) {
	// 		// 	status.Status = pb.EnumJobStatus_EJS_RUNNING
	// 		// 	status.Progress = progress
	// 		// 	populateJobStatus(status)
	// 	}); err != nil {
	// 	panic(err)
	// }
	var status core.JobStatus
	files, err := core.RunJob(context.Background(), job, &status)
	if err != nil {
		panic(err)
	} else {
		log.Println(files)
	}
}
