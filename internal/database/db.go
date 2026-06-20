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

		// Auto-migrate all models for this site's database.
		// EmailTemplate is included so templates can be seeded right after first deploy.
		if err := db.AutoMigrate(&models.User{}, &models.OAuthAccount{}, &models.EmailTemplate{}); err != nil {
			return fmt.Errorf("failed to auto-migrate DB for site %q: %w", host, err)
		}

		if err := seedEmailTemplates(db, host); err != nil {
			return fmt.Errorf("failed to seed email templates for site %q: %w", host, err)
		}

		mu.Lock()
		registry[host] = db
		mu.Unlock()

		fmt.Printf("[DB] Connected and migrated site: %s\n", host)
	}
	return nil
}

// seedEmailTemplates seeds default email templates into the database if they don't exist.
func seedEmailTemplates(db *gorm.DB, site string) error {
	var count int64
	if err := db.Model(&models.EmailTemplate{}).Where("site = ? AND key = ?", site, "verify_email").Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		defaultTemplate := models.EmailTemplate{
			Site:    site,
			Key:     "verify_email",
			Subject: "Your verification code for {{site}}",
			Content: `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Verify your email</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f4f4f5; margin: 0; padding: 40px 0; }
    .container { max-width: 480px; margin: 0 auto; background: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 24px rgba(0,0,0,0.08); }
    .header { background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%); padding: 32px; text-align: center; }
    .header h1 { color: #ffffff; margin: 0; font-size: 24px; font-weight: 700; }
    .body { padding: 40px 32px; text-align: center; }
    .body p { color: #374151; font-size: 16px; line-height: 1.6; margin: 0 0 24px; }
    .code-box { display: inline-block; background: #f3f4f6; border: 2px dashed #6366f1; border-radius: 12px; padding: 20px 40px; margin: 8px 0 24px; }
    .code-box span { font-size: 36px; font-weight: 800; letter-spacing: 8px; color: #4f46e5; font-family: 'Courier New', monospace; }
    .expiry { font-size: 14px; color: #6b7280; }
    .footer { padding: 24px 32px; border-top: 1px solid #f0f0f0; }
    .footer p { color: #9ca3af; font-size: 13px; margin: 0; text-align: center; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>Verify Your Email</h1>
    </div>
    <div class="body">
      <p>Thanks for signing up! Enter the code below to verify your email address.</p>
      <div class="code-box">
        <span>{{code}}</span>
      </div>
      <p class="expiry">This code expires in <strong>15 minutes</strong>.</p>
      <p style="font-size:14px;color:#6b7280;">Do not share this code with anyone.</p>
    </div>
    <div class="footer">
      <p>If you didn't create an account, you can safely ignore this email.</p>
    </div>
  </div>
</body>
</html>`,
		}
		if err := db.Create(&defaultTemplate).Error; err != nil {
			return err
		}
	}
	return nil
}
