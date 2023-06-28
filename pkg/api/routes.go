package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sath-run/engine/cmd/utils"
)

func fatal(c *gin.Context, err error) bool {
	if err == nil {
		return false
	} else if c.Writer.Status() == http.StatusBadRequest {
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
	utils.LogDebug("initializing api")
	r := gin.Default()
	r.SetTrustedProxies([]string{"127.0.0.1"})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.GET("/version", GetVersion)
	r.POST("/services/start", StartService)
	r.POST("/services/stop", StopService)
	r.GET("/services/status", GetServiceStatus)
	r.GET("/jobs/stream", StreamJobStatus)
	r.GET("/jobs", GetJobStatus)
	r.POST("/jobs/pause", PauseJob)
	r.POST("/jobs/resume", ResumeJob)
	r.POST("/users/login", Login)
	r.POST("/users/logout", Logout)
	r.GET("/users/credential", GetCredential)
	r.Run(addr)
}
