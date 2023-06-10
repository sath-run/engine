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

type TaskStatus struct {
	Id          string  `json:"id"`
	Message     string  `json:"message"`
	Status      string  `json:"status"`
	Progress    float64 `json:"progress"`
	CreatedAt   int64   `json:"createdAt"`
	CompletedAt int64   `json:"completedAt"`
	ContainerId string  `json:"containerId"`
	Image       string  `json:"Image"`
}

func getTaskStatusFromCore(coreStatus *core.TaskStatus) *TaskStatus {
	if coreStatus == nil {
		return nil
	}
	return &TaskStatus{
		Id:          coreStatus.Id,
		Message:     coreStatus.Message,
		Status:      core.TaskStatusText(coreStatus.Status),
		Progress:    coreStatus.Progress,
		CreatedAt:   coreStatus.CreatedAt.Unix(),
		CompletedAt: coreStatus.CompletedAt.Unix(),
		ContainerId: coreStatus.ContainerId,
		Image:       coreStatus.ImageUrl,
	}
}

func readTaskStatusFromLog() ([]*TaskStatus, error) {
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

	retval := []*TaskStatus{}
	for scanner.Scan() {
		line := scanner.Text()
		var status core.TaskStatus
		if err := json.Unmarshal([]byte(line), &status); err != nil {
			return nil, err
		}
		retval = append(retval, getTaskStatusFromCore(&status))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return retval, nil
}

func StreamTaskStatus(c *gin.Context) {
	chanStream := make(chan core.TaskStatus)
	core.SubscribeTaskStatus(chanStream)
	c.Stream(func(w io.Writer) bool {
		select {
		case status := <-chanStream:
			c.SSEvent("job-status", gin.H{
				"jobs": []*TaskStatus{getTaskStatusFromCore(&status)},
			})
			return true
		case <-c.Request.Context().Done():
			// client disconnected
			core.UnsubscribeTaskStatus(chanStream)
			return false
		}
	})
}

func GetTaskStatus(c *gin.Context) {
	status := core.GetTaskStatus()

	jobs := []*TaskStatus{}

	if status != nil {
		jobs = append(jobs, getTaskStatusFromCore(status))
	}

	if c.Query("filter") == "all" {
		completed, err := readTaskStatusFromLog()
		if fatal(c, err) {
			return
		}
		jobs = append(jobs, completed...)
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs": jobs,
	})
}
