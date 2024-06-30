package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Login(c *gin.Context) {
	var form struct {
		Username string `binding:"required"`
		Password string `binding:"required"`
	}
	if err := c.Bind(&form); fatal(c, err) {
		return
	}
	if err := engine.Login(form.Username, form.Password); err != nil {
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

func GetUserInfo(c *gin.Context) {
	info := engine.GetUserInfo()
	if info == nil {
		c.JSON(http.StatusOK, gin.H{})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"email": info.Email,
			"name":  info.Name,
		})
	}
}

func Logout(c *gin.Context) {
	if err := engine.Logout(); fatal(c, err) {
		return
	}
	c.Status(http.StatusOK)
}
