package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sath-run/engine/cmd/core"
)

const Version = "1.0.0"

func StartService(c *gin.Context) {
	err := core.Start()
	if err == core.ErrRunning {
		c.JSON(http.StatusOK, gin.H{
			"message": "engine have already been started",
		})
	} else if fatal(c, err) {
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "successfully started sath-engine",
	})
}

func StopService(c *gin.Context) {
	err := core.Stop()
	if fatal(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "sath-engine stopped",
	})
}

func GetServiceStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  core.Status(),
		"version": Version,
	})
}
