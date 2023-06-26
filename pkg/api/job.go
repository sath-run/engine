package api

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/sath-run/engine/cmd/core"
	"github.com/sath-run/engine/cmd/utils"
)

type JobStatus struct {
	Id          string  `json:"id"`
	Message     string  `json:"message"`
	Status      string  `json:"status"`
	Progress    float64 `json:"progress"`
	CreatedAt   int64   `json:"createdAt"`
	CompletedAt int64   `json:"completedAt"`
	ContainerId string  `json:"containerId"`
	Image       string  `json:"Image"`
}

func getJobStatusFromCore(coreStatus *core.JobStatus) *JobStatus {
	if coreStatus == nil {
		return nil
	}
	return &JobStatus{
		Id:          coreStatus.Id,
		Message:     coreStatus.Message,
		Status:      core.JobStatusText(coreStatus.Status),
		Progress:    coreStatus.Progress,
		CreatedAt:   coreStatus.CreatedAt.Unix(),
		CompletedAt: coreStatus.CompletedAt.Unix(),
		ContainerId: coreStatus.ContainerId,
		Image:       coreStatus.Image,
	}
}

func readJobStatusFromLog() ([]*JobStatus, error) {
	dir, err := utils.GetExecutableDir()
	if err != nil {
		return nil, err
	}
	logPath := filepath.Join(dir, "log", "jobs.log")
	file, err := os.Open(logPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	retval := []*JobStatus{}
	for scanner.Scan() {
		line := scanner.Text()
		var status core.JobStatus
		if err := json.Unmarshal([]byte(line), &status); err != nil {
			return nil, err
		}
		retval = append(retval, getJobStatusFromCore(&status))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return retval, nil
}

func StreamJobStatus(c *gin.Context) {
	chanStream := make(chan core.JobStatus)
	core.SubscribeJobStatus(chanStream)
	c.Stream(func(w io.Writer) bool {
		select {
		case status := <-chanStream:
			c.SSEvent("job-status", gin.H{
				"jobs": []*JobStatus{getJobStatusFromCore(&status)},
			})
			return true
		case <-c.Request.Context().Done():
			// client disconnected
			core.UnsubscribeJobStatus(chanStream)
			return false
		}
	})
}

func GetJobStatus(c *gin.Context) {
	status := core.GetJobStatus()

	jobs := []*JobStatus{}

	if status != nil {
		jobs = append(jobs, getJobStatusFromCore(status))
	}

	if c.Query("filter") == "all" {
		completed, err := readJobStatusFromLog()
		if fatal(c, err) {
			return
		}
		jobs = append(jobs, completed...)
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs": jobs,
	})
}
