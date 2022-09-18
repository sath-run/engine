package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sath-run/engine/cmd/core"
)

func StartService(c *gin.Context) {
	err := core.Start()
	if fatal(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

func StopService(c *gin.Context) {
	err := core.Stop()
	if fatal(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}
