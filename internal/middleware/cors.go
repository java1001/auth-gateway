package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS returns a middleware that allows requests only from registered site origins.
// It adds the appropriate Access-Control headers and handles OPTIONS preflight.
func CORS(registeredHosts []string) gin.HandlerFunc {
	// Build an allowlist set for O(1) lookup.
	allowed := make(map[string]struct{}, len(registeredHosts)*2)
	for _, host := range registeredHosts {
		// Allow both http and https origins.
		allowed["https://"+host] = struct{}{}
		allowed["http://"+host] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Strip trailing slash for normalisation.
		origin = strings.TrimRight(origin, "/")

		if _, ok := allowed[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Max-Age", "86400")
			c.Header("Vary", "Origin")
		}

		// Handle preflight requests.
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
