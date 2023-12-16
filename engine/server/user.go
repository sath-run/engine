package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sath-run/engine/engine/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Login(c *gin.Context) {
	var form struct {
		Username     string `binding:"required"`
		Password     string `binding:"required"`
		Organization string ``
	}
	if err := c.Bind(&form); fatal(c, err) {
		return
	}
	if err := core.Login(form.Username, form.Password, form.Organization); err != nil {
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.InvalidArgument {
				c.JSON(http.StatusBadRequest, gin.H{
					"message": st.Message(),
				})
			} else if st.Code() == codes.Unauthenticated {
				c.JSON(http.StatusUnauthorized, gin.H{
					"message": "Your username and password are not recognized",
				})
			}
			return
		}
		fatal(c, err)
		return
	}
	c.Status(http.StatusOK)
}

func GetCredential(c *gin.Context) {
	credential := core.Credential()
	c.JSON(http.StatusOK, gin.H{
		"username":     credential.Username,
		"organization": credential.Organization,
	})
}

func Logout(c *gin.Context) {
	if err := core.Logout(); fatal(c, err) {
		return
	}
	c.Status(http.StatusOK)
}
