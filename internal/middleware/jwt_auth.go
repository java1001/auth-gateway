package middleware

import (
	"net/http"
	"strings"

	"github.com/auth-gateway/internal/database"
	"github.com/auth-gateway/internal/services"
	"github.com/gin-gonic/gin"
)

// JWTAuth validates the Bearer token in the Authorization header and injects
// the resolved *gorm.DB (based on the site claim inside the JWT) into the context.
//
// Used on all protected routes (/auth/me, /auth/logout).
// The site claim acts as the DB routing key — no SiteResolver needed.
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing or invalid authorization header",
				"code":  "INVALID_TOKEN",
			})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := services.ValidateJWT(tokenStr, secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "token is invalid or expired",
				"code":  "INVALID_TOKEN",
			})
			return
		}

		if claims == nil || claims.Site == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "token is invalid or expired",
				"code":  "INVALID_TOKEN",
			})
			return
		}

		if services.IsTokenRevoked(claims.ID) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "token has been revoked",
				"code":  "INVALID_TOKEN",
			})
			return
		}

		// Route to the correct per-site database using the site claim.
		db, err := database.GetDB(claims.Site)
		if err != nil {
			// Site in token doesn't exist in registry — misconfiguration or stale token.
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "site not registered",
				"code":  "SITE_NOT_FOUND",
			})
			return
		}

		c.Set("db", db)
		c.Set("site", claims.Site)
		c.Set("claims", claims)
		c.Next()
	}
}
