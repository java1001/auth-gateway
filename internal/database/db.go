package database

import (
	"fmt"
	"sync"

	"github.com/auth-gateway/config"
	"github.com/auth-gateway/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	registry = make(map[string]*gorm.DB)
	mu       sync.RWMutex
)

// GetDB returns the *gorm.DB for the given site host.
// Returns an error if the site is not registered.
func GetDB(siteHost string) (*gorm.DB, error) {
	mu.RLock()
	db, ok := registry[siteHost]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no database registered for site %q", siteHost)
	}
	return db, nil
}

// InitAllDatabases opens a PostgreSQL connection for every site in the config,
// runs AutoMigrate for User and OAuthAccount, and stores the result in the registry.
func InitAllDatabases(sites map[string]config.SiteDBConfig) error {
	for host, cfg := range sites {
		db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err != nil {
			return fmt.Errorf("failed to connect to DB for site %q: %w", host, err)
		}

		if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto").Error; err != nil {
			return fmt.Errorf("failed to ensure pgcrypto extension for site %q: %w", host, err)
		}

		// Auto-migrate both models for this site's database.
		if err := db.AutoMigrate(&models.User{}, &models.OAuthAccount{}); err != nil {
			return fmt.Errorf("failed to auto-migrate DB for site %q: %w", host, err)
		}

		mu.Lock()
		registry[host] = db
		mu.Unlock()

		fmt.Printf("[DB] Connected and migrated site: %s\n", host)
	}
	return nil
}
