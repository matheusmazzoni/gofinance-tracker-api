package dto

import (
	"github.com/gin-gonic/gin"
)

func SendErrorResponse(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, ErrorResponse{
		Error: message,
	})
}

func SendSuccessResponse(c *gin.Context, code int, data interface{}) {
	c.JSON(code, data)
}
