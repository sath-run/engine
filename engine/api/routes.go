package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/sath-run/engine/engine/daemon"
)

var engine *daemon.Core

func fatal(c *gin.Context, err error) bool {
	if err == nil {
		return false
	} else if c.Writer.Status() == http.StatusBadRequest {
		return false
	} else if c.IsAborted() {
		log.Fatal().Err(err).Send()
		return true
	} else {
		log.Fatal().Err(err).Send()
		c.AbortWithStatus(http.StatusInternalServerError)
		return true
	}
}
func Init(file string, egin *daemon.Core) {
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
