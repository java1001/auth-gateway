package router

import (
	"github.com/auth-gateway/config"
	"github.com/auth-gateway/internal/handlers"
	"github.com/auth-gateway/internal/middleware"
	"github.com/auth-gateway/internal/services"
	"github.com/gin-gonic/gin"
)

// SetupRouter builds and returns the configured Gin engine.
// Route groups:
//
//	/health                     — no middleware
//	/auth  + SiteResolver       — public auth routes (signup, login, verify)
//	/auth/google/:site/*        — OAuth routes with site in the URL path
//	/auth/twitter/:site/*       — OAuth routes with site in the URL path
//	/auth  + JWTAuth            — protected routes (me, logout)
func SetupRouter(cfg *config.Config, emailSvc *services.EmailService, oauthSvc *services.OAuthService) *gin.Engine {
	r := gin.Default()

	// Global security gate first, then CORS headers for allowed origins.
	r.Use(middleware.AllowedSites(cfg.AllowSites))
	r.Use(middleware.CORS(cfg.AllowSites))

	// ── Health ────────────────────────────────────────────────────────────────
	r.GET("/health", handlers.HealthCheck)

	// ── Public auth routes ────────────────────────────────────────────────────
	// SiteResolver reads the Host header → injects db + site into context.
	public := r.Group("/auth")
	public.Use(middleware.SiteResolver())
	{
		public.POST("/signup",
			middleware.AuthRateLimit(),
			handlers.Signup(emailSvc, cfg.VerifyCodeLength),
		)
		public.POST("/login",
			middleware.AuthRateLimit(),
			handlers.Login(cfg),
		)
		// POST — keeps OTP code out of server logs and browser history.
		public.POST("/verify-email",
			middleware.AuthRateLimit(),
			handlers.VerifyEmail(),
		)
		public.POST("/refresh",
			middleware.AuthRateLimit(),
			handlers.Refresh(cfg),
		)
	}

	// ── OAuth callbacks ───────────────────────────────────────────────────────
	r.GET("/auth/google/:site/login",
		middleware.OAuthRateLimit(),
		handlers.GoogleLogin(oauthSvc, cfg),
	)
	r.GET("/auth/google/:site/callback", handlers.GoogleCallback(oauthSvc, cfg))
	r.GET("/auth/twitter/:site/login",
		middleware.OAuthRateLimit(),
		handlers.TwitterLogin(oauthSvc, cfg),
	)
	r.GET("/auth/twitter/:site/callback", handlers.TwitterCallback(oauthSvc, cfg))

	// ── Protected routes ──────────────────────────────────────────────────────
	// JWTAuth decodes the token → reads site claim → resolves correct DB.
	// No Host header needed — the JWT carries its own routing context.
	protected := r.Group("/auth")
	protected.Use(middleware.JWTAuth(cfg.JWTSecret))
	{
		protected.GET("/me", handlers.Me())
		protected.POST("/logout", handlers.Logout())
	}

	return r
}
