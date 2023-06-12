package core_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/sath-run/engine/cmd/utils"
)

// func TestFileUpload(t *testing.T) {
// 	err := core.Init(&core.Config{
// 		GrpcAddress: "localhost:50051",
// 		SSL:         false,
// 	})
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	files, err := core.ProcessOutputs(".", "587962", []string{"docker.go"})
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	spew.Dump(files)
// }

// func TestMdJob(t *testing.T) {
// 	err := core.Init(&core.Config{
// 		GrpcAddress: "scheduler.sath.run:50051",
// 		SSL:         true,
// 		DataDir:     "/tmp/sath",
// 	})
// 	if err != nil {
// 		panic(err)
// 	}
// 	job := &pb.JobGetResponse{
// 		Image: &pb.DockerImage{
// 			Repository: "zengxinzhy/amber-runtime-cuda11.4.2",
// 			Tag:        "latest",
// 		},
// 		Cmds: []string{
// 			"amber",
// 		},
// 		VolumePath: "/data",
// 		GpuOpts:    "all",
// 		Files: []*pb.File{
// 			{
// 				Name: "equi.rst",
// 				Content: &pb.File_Remote{
// 					Remote: &pb.FileUri{
// 						Uri:         "https://sath-ligand.s3.ap-east-1.amazonaws.com/md/equi.rst",
// 						FetchMethod: pb.EnumFileFetchMethod_EFFM_HTTP,
// 					},
// 				},
// 			},
// 			{
// 				Name: "prod.in",
// 				Content: &pb.File_Remote{
// 					Remote: &pb.FileUri{
// 						Uri:         "https://sath-ligand.s3.ap-east-1.amazonaws.com/md/prod.in",
// 						FetchMethod: pb.EnumFileFetchMethod_EFFM_HTTP,
// 					},
// 				},
// 			},
// 			{
// 				Name: "ras-raf_solvated.prmtop",
// 				Content: &pb.File_Remote{
// 					Remote: &pb.FileUri{
// 						Uri:         "https://sath-ligand.s3.ap-east-1.amazonaws.com/md/ras-raf_solvated.prmtop",
// 						FetchMethod: pb.EnumFileFetchMethod_EFFM_HTTP,
// 					},
// 				},
// 			},
// 		},
// 		Outputs: []string{},
// 	}
// 	// var containerId string
// 	// if err := core.ExecImage(
// 	// 	context.Background(), core.GetDockerClient(), job.Cmds, "zengxinzhy/amber-runtime-cuda11.4.2:1.2", "/tmp/sath/sath_tmp_1040899213", "/tmp/sath/sath_tmp_1040899213", job.VolumePath,
// 	// 	job.GpuOpts, &containerId, func(progress float64) {
// 	// 		// 	status.Status = pb.EnumExecStatus_RUNNING
// 	// 		// 	status.Progress = progress
// 	// 		// 	populateTaskStatus(status)
// 	// 	}); err != nil {
// 	// 	panic(err)
// 	// }
// 	var status core.TaskStatus
// 	files, err := core.RunJob(context.Background(), job, &status)
// 	if err != nil {
// 		panic(err)
// 	} else {
// 		log.Println(files)
// 	}
// }

func TestRequest(t *testing.T) {
	obj := map[string]any{
		"x": 123,
		"y": "fewef",
	}
	data, err := json.Marshal(obj)
	checkErr(err)
	req, err := retryablehttp.NewRequest("POST", "http://127.0.0.1:8080/outputs", data)
	checkErr(err)
	res, err := retryablehttp.NewClient().Do(req)
	checkErr(err)
	spew.Dump(res)
}

func DoUpload(folder string, method string, url string, params map[string]string) (*http.Response, error) {
	var buf bytes.Buffer
	if err := utils.Compress(folder, &buf); err != nil {
		return nil, err
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "output.tar.gz")
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, &buf)
	if err != nil {
		return nil, err
	}

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	req, err := retryablehttp.NewRequest("PUT", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := retryablehttp.NewClient()
	resp, err := client.Do(req)
	return resp, err
}

func TestUpload(t *testing.T) {
	// tar + gzip
	url := "http://localhost:8080/outputs/uploads"
	params := map[string]string{
		"title":       "My Document",
		"author":      "Matt Aimonetti",
		"description": "A document with all the Go programming language secrets",
	}
	resp, err := DoUpload("./output", "PUT", url, params)
	checkErr(err)
	data, err := io.ReadAll(resp.Body)
	checkErr(err)
	fmt.Println(string(data))
}

func TestRedirectedUpload(t *testing.T) {
	url := "http://localhost:8080/outputs"
	obj := map[string]any{
		"sathJobId":  "aaa",
		"sathTaskId": "bbb",
		"sathExecId": "ccc",
	}
	data, err := json.Marshal(obj)
	checkErr(err)
	req, err := retryablehttp.NewRequest("POST", url, data)
	checkErr(err)
	req.Header.Set("Content-Type", "application/json")
	res, err := retryablehttp.NewClient().Do(req)
	checkErr(err)
	data, err = io.ReadAll(res.Body)
	checkErr(err)
	var response struct {
		Method string `json:"method"`
		Url    string `json:"url"`
	}
	err = json.Unmarshal(data, &response)
	checkErr(err)
	// resp, err := DoUpload("./output", response.Method, response.Url, nil)
	var buf bytes.Buffer
	if err := utils.Compress("./output", &buf); err != nil {
		checkErr(err)
	}
	req, err = retryablehttp.NewRequest(response.Method, response.Url, &buf)
	if err != nil {
		checkErr(err)
	}
	resp, err := retryablehttp.NewClient().Do(req)
	if err != nil {
		checkErr(err)
	}
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	checkErr(err)
	fmt.Println(string(data))
}
