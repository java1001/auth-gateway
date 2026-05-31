package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// AllowedSites blocks requests whose caller host is not in the allowlist.
// It skips the health probe and OAuth callback endpoints so the gateway can
// still receive provider redirects on its own domain.
func AllowedSites(allowedHosts []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedHosts))
	for _, host := range allowedHosts {
		allowed[strings.ToLower(strings.TrimSpace(host))] = struct{}{}
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		switch {
		case path == "/health":
			c.Next()
			return
		case strings.HasPrefix(path, "/auth/google/") && strings.HasSuffix(path, "/callback"):
			c.Next()
			return
		case strings.HasPrefix(path, "/auth/twitter/") && strings.HasSuffix(path, "/callback"):
			c.Next()
			return
		}

		caller := extractHost(c.GetHeader("Origin"))
		if caller == "" {
			caller = extractHost(c.GetHeader("Host"))
		}

		if caller == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "origin not allowed",
				"code":  "ORIGIN_NOT_ALLOWED",
			})
			return
		}

		if _, ok := allowed[strings.ToLower(caller)]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "origin not allowed",
				"code":  "ORIGIN_NOT_ALLOWED",
			})
			return
		}

		c.Next()
	}
}

func extractHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if parsed, err := url.Parse(raw); err == nil && parsed.Host != "" {
		raw = parsed.Host
	}

	host := raw
	if i := strings.LastIndex(host, ":"); i != -1 {
		if strings.Count(host, ":") == 1 {
			host = host[:i]
		}
	}

	return strings.ToLower(strings.TrimSpace(host))
}
