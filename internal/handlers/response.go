package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

func Fail(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

func FailWithRecovery(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error":              message,
		"recovery_available": true,
	})
}
