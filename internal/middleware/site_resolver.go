package middleware

import (
	"net/http"

	"github.com/auth-gateway/internal/database"
	"github.com/gin-gonic/gin"
)

// SiteResolver resolves the site from the Host header and injects the
// corresponding DB handle and site key into the request context.
func SiteResolver() gin.HandlerFunc {
	return func(c *gin.Context) {
		site := extractHost(c.Request.Host)
		if site == "" {
			site = extractHost(c.GetHeader("Host"))
		}

		db, err := database.GetDB(site)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error": "site not found",
				"code":  "SITE_NOT_FOUND",
			})
			return
		}

		c.Set("db", db)
		c.Set("site", site)
		c.Next()
	}
}
