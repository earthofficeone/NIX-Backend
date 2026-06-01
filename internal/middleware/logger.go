package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// RequestLogger logs each HTTP request (method, path, status, latency).
func RequestLogger() gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health"},
		Formatter: func(p gin.LogFormatterParams) string {
			return fmt.Sprintf(
				"[%s] %3d | %13v | %15s | %s %s\n",
				p.TimeStamp.Format("2006-01-02 15:04:05"),
				p.StatusCode,
				p.Latency,
				p.ClientIP,
				p.Method,
				p.Path,
			)
		},
	})
}
