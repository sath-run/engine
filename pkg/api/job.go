package api

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sath-run/engine/cmd/core"
)

type JobStatus struct {
	Id       string  `json:"id"`
	Message  string  `json:"message"`
	Status   string  `json:"status"`
	Progress float64 `json:"progress"`
}

func getJobStatusFromCore(coreStatus *core.JobStatus) *JobStatus {
	if coreStatus == nil {
		return nil
	}
	return &JobStatus{
		Id:       coreStatus.Id,
		Message:  coreStatus.Message,
		Status:   core.JobStatusText(coreStatus.Status),
		Progress: coreStatus.Progress,
	}
}

func RunSingleJob(c *gin.Context) {
	// _, err := action.RunSingleJob()
	// if fatal(c, err) {
	// 	return
	// }
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

func StreamCurrentJobStatus(c *gin.Context) {
	chanStream := make(chan core.JobStatus)
	core.SubscribeJobStatus(chanStream)
	c.Stream(func(w io.Writer) bool {
		select {
		case status := <-chanStream:
			c.SSEvent("job-status", getJobStatusFromCore(&status))
			return true
		case <-c.Request.Context().Done():
			// client disconnected
			core.UnsubscribeJobStatus(chanStream)
			return false
		}
	})
}

func GetCurrentJobStatus(c *gin.Context) {
	status := core.GetCurrentJobStatus()
	if status == nil {
		c.Status(http.StatusOK)
	} else {
		c.JSON(http.StatusOK, getJobStatusFromCore(status))
	}
}
