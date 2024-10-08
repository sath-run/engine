package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sath-run/engine/constants"
	"github.com/sath-run/engine/daemon"
)

func StartService(c *gin.Context) {
	if engine.GetUserInfo() == nil {
		// login is required
		c.AbortWithStatusJSON(http.StatusUnauthorized, "login is required")
		return
	}
	err := engine.Start()
	if errors.Is(err, daemon.ErrRunning) {
		c.JSON(http.StatusOK, gin.H{
			"message": "engine is already started",
		})
	} else if errors.Is(err, daemon.ErrStopping) {
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
	// var form struct {
	// 	Wait bool `form:"wait"`
	// }
	// if err := c.ShouldBind(&form); fatal(c, err) {
	// 	return
	// }
	// err := core.Stop(form.Wait)
	// if fatal(c, err) {
	// 	return
	// }
	c.JSON(http.StatusOK, gin.H{
		"message": "sath-engine stopped",
	})
}

func GetServiceStatus(c *gin.Context) {
	// status := core.GetJobStatus()
	jobs := []gin.H{}
	// if status != nil {
	// 	jobs = append(jobs, gin.H{
	// 		"execId": status.Id,
	// 	})
	// }
	c.JSON(http.StatusOK, gin.H{
		"status":  engine.Status(),
		"version": constants.Version,
		"jobs":    jobs,
	})
}

func GetVersion(c *gin.Context) {
	c.String(http.StatusOK, constants.Version)
}
