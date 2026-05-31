package services

import (
	"sync"
	"time"
)

var (
	blocklist   = make(map[string]time.Time)
	blocklistMu sync.RWMutex
)

func init() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			blocklistMu.Lock()
			now := time.Now()
			for jti, expiresAt := range blocklist {
				if now.After(expiresAt) {
					delete(blocklist, jti)
				}
			}
			blocklistMu.Unlock()
		}
	}()
}

// IsTokenRevoked reports whether the JWT ID has been revoked via logout.
func IsTokenRevoked(jti string) bool {
	blocklistMu.RLock()
	defer blocklistMu.RUnlock()
	_, revoked := blocklist[jti]
	return revoked
}

// RevokeToken stores the JWT ID until the token naturally expires.
func RevokeToken(jti string, expiresAt time.Time) {
	blocklistMu.Lock()
	defer blocklistMu.Unlock()
	blocklist[jti] = expiresAt
}
