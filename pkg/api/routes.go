package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sath-run/engine/cmd/utils"
)

func fatal(c *gin.Context, err error) bool {
	if err == nil {
		return false
	} else if c.IsAborted() {
		utils.LogError(err)
		return true
	} else {
		utils.LogError(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return true
	}
}
func Init(addr string) {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.POST("/services/start", StartService)
	r.POST("/services/stop", StopService)
	r.GET("/jobs/current/stream", StreamCurrentJobStatus)
	r.GET("/jobs/current", GetCurrentJobStatus)
	r.POST("/jobs/run", RunSingleJob)
	r.Run(addr)
}
