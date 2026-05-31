package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// ipLimiter holds the rate limiter and last-seen time for a single IP.
type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiterStore is a thread-safe store of per-IP rate limiters.
type RateLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	r        rate.Limit
	b        int
}

func newRateLimiterStore(r rate.Limit, b int) *RateLimiterStore {
	store := &RateLimiterStore{
		limiters: make(map[string]*ipLimiter),
		r:        r,
		b:        b,
	}
	// Periodically remove limiters that haven't been seen in 5 minutes.
	go store.cleanupLoop()
	return store
}

func (s *RateLimiterStore) get(ip string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, exists := s.limiters[ip]
	if !exists {
		lim := rate.NewLimiter(s.r, s.b)
		s.limiters[ip] = &ipLimiter{limiter: lim, lastSeen: time.Now()}
		return lim
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

func (s *RateLimiterStore) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		cutoff := time.Now().Add(-5 * time.Minute)
		for ip, entry := range s.limiters {
			if entry.lastSeen.Before(cutoff) {
				delete(s.limiters, ip)
			}
		}
		s.mu.Unlock()
	}
}

// authLimiter: 5 requests/minute for signup and login.
var authLimiter = newRateLimiterStore(rate.Every(time.Minute/5), 5)

// oauthLimiter: 20 requests/minute for OAuth initiation endpoints.
var oauthLimiter = newRateLimiterStore(rate.Every(time.Minute/20), 20)

// RateLimit returns a middleware that enforces the given per-IP rate limit.
// Use AuthRateLimit() or OAuthRateLimit() convenience wrappers instead.
func rateLimit(store *RateLimiterStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !store.get(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, please try again later",
				"code":  "RATE_LIMITED",
			})
			return
		}
		c.Next()
	}
}

// AuthRateLimit enforces 5 req/min per IP — for signup and login.
func AuthRateLimit() gin.HandlerFunc {
	return rateLimit(authLimiter)
}

// OAuthRateLimit enforces 20 req/min per IP — for OAuth login initiation.
func OAuthRateLimit() gin.HandlerFunc {
	return rateLimit(oauthLimiter)
}
