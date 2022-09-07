package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sath-run/engine/cmd/action"
)

func RunSingleJob(c *gin.Context) {
	_, err := action.RunSingleJob()
	if fatal(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}
