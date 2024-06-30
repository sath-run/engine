package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sath-run/engine/engine/core"
	"github.com/sath-run/engine/engine/logger"
)

var engine *core.Core

func fatal(c *gin.Context, err error) bool {
	if err == nil {
		return false
	} else if c.Writer.Status() == http.StatusBadRequest {
		return false
	} else if c.IsAborted() {

		logger.Error(err)
		return true
	} else {
		logger.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return true
	}
}
func Init(file string, egin *core.Core) {
	logger.Debug("initializing api")
	engine = egin
	r := gin.Default()
	// r.SetTrustedProxies([]string{"unix"})
	r.SetTrustedProxies(nil)

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.GET("/version", GetVersion)
	r.POST("/services/start", StartService)
	r.POST("/services/stop", StopService)
	r.GET("/services/status", GetServiceStatus)
	// r.GET("/jobs/stream", StreamJobStatus)
	r.GET("/jobs", GetJobStatus)
	r.POST("/jobs/pause", PauseJob)
	r.POST("/jobs/resume", ResumeJob)
	r.POST("/users/login", Login)
	r.POST("/users/logout", Logout)
	r.GET("/users/info", GetUserInfo)

	if err := r.RunUnix(file); err != nil {
		panic(err)
	}
}
