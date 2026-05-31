package main

import (
	"log"

	"github.com/auth-gateway/config"
	"github.com/auth-gateway/internal/database"
	"github.com/auth-gateway/internal/router"
	"github.com/auth-gateway/internal/services"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	if err := database.InitAllDatabases(cfg.Sites); err != nil {
		log.Fatalf("database init failed: %v", err)
	}

	emailSvc := services.NewEmailService(cfg.ResendAPIKey, cfg.EmailFrom)
	oauthSvc := services.NewOAuthService(cfg)

	engine := router.SetupRouter(cfg, emailSvc, oauthSvc)
	if err := engine.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
