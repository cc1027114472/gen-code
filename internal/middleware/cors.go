package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// LocalDevCORS allows localhost browser-based development clients to call the app server.
func LocalDevCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin != "" && isAllowedLocalOrigin(origin) {
			header := c.Writer.Header()
			header.Set("Access-Control-Allow-Origin", origin)
			header.Set("Access-Control-Allow-Credentials", "true")
			header.Set("Access-Control-Allow-Headers", "Content-Type, Accept, Cache-Control, X-Requested-With")
			header.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			header.Set("Vary", "Origin")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func isAllowedLocalOrigin(origin string) bool {
	return strings.HasPrefix(origin, "http://127.0.0.1:") ||
		strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "https://127.0.0.1:") ||
		strings.HasPrefix(origin, "https://localhost:")
}
