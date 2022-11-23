package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sath-run/engine/cmd/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Login(c *gin.Context) {
	var form struct {
		Email    string `binding:"required" form:"email" json:"email"`
		Password string `binding:"required" form:"password" json:"password"`
	}
	if err := c.Bind(&form); fatal(c, err) {
		return
	}
	if err := core.Login(form.Email, form.Password); err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.InvalidArgument {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": st.Message(),
			})
		} else {
			fatal(c, err)
		}
		return
	}
	c.Status(http.StatusOK)
}

func GetToken(c *gin.Context) {
	c.String(http.StatusOK, core.Token())
}
