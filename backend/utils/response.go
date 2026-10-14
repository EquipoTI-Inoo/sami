package utils

import (
	"log"

	"github.com/gin-gonic/gin"
)

func OK(c *gin.Context, data any) {
	c.JSON(200, gin.H{"data": data})
}

func Created(c *gin.Context, data any) {
	c.JSON(201, gin.H{"data": data})
}

func BadRequest(c *gin.Context, msg string) {
	c.JSON(400, gin.H{"error": msg})
}

func Unauthorized(c *gin.Context, msg string) {
	c.JSON(401, gin.H{"error": msg})
}

func Conflict(c *gin.Context, msg string) {
	c.JSON(409, gin.H{"error": msg})
}

func NotFound(c *gin.Context, msg string) {
	c.JSON(404, gin.H{"error": msg})
}

func InternalError(c *gin.Context, msg string) {
	c.JSON(500, gin.H{"error": msg})
}

// InternalErrorf logs the real error server-side (InternalError's msg is
// deliberately generic — never echoes internals to the client) before
// responding the same way InternalError does.
func InternalErrorf(c *gin.Context, msg string, err error) {
	log.Printf("%s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	InternalError(c, msg)
}
