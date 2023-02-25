package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sath-run/engine/cmd/core"
)

func StartService(c *gin.Context) {
	err := core.Start()
	if errors.Is(err, core.ErrRunning) {
		c.JSON(http.StatusOK, gin.H{
			"message": "engine have already been started",
		})
	} else if errors.Is(err, core.ErrStopping) {
		c.JSON(http.StatusOK, gin.H{
			"message": "engine is stopping, please wait for current job completion",
		})
	} else if fatal(c, err) {
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "successfully started sath-engine",
	})
}

func StopService(c *gin.Context) {
	var form struct {
		Wait bool `form:"wait"`
	}
	if err := c.ShouldBind(&form); fatal(c, err) {
		return
	}
	err := core.Stop(form.Wait)
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
		"version": core.VERSION,
	})
}

func GetVersion(c *gin.Context) {
	c.String(http.StatusOK, core.VERSION)
}
