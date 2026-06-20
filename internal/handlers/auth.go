package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/auth-gateway/internal/models"
	"github.com/auth-gateway/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ─── Request types ────────────────────────────────────────────────────────────

type signupRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// verifyEmailRequest accepts the recipient email + OTP code submitted by the user.
// Requiring email (not just code) prevents brute-force guessing across users.
type verifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code"  binding:"required"`
}

// normalizeInput trims whitespace and rejects null bytes.
func normalizeInput(value string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.ContainsRune(value, '\x00') {
		return "", fmt.Errorf("input contains invalid null byte")
	}
	return value, nil
}

// ─── Signup ───────────────────────────────────────────────────────────────────

// Signup registers a new user with email + password.
// A numeric OTP code (length from cfg.VerifyCodeLength) is generated, stored,
// and emailed via the site's "verify_email" template from the DB.
// POST /auth/signup  (requires SiteResolver middleware)
func Signup(emailSvc *services.EmailService, codeLength int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.ContentType(), "application/json") {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{
				"error": "Content-Type must be application/json",
				"code":  "VALIDATION_ERROR",
			})
			return
		}

		var req signupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
			return
		}

		db := c.MustGet("db").(*gorm.DB)
		site := c.GetString("site")

		var err error
		req.Email, err = normalizeInput(req.Email)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
			return
		}
		req.Password, err = normalizeInput(req.Password)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
			return
		}
		if len(req.Password) > 72 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "password must be 72 characters or fewer",
				"code":  "VALIDATION_ERROR",
			})
			return
		}

		// Duplicate email check within this site's DB.
		var existing models.User
		if err := db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"error": "email already registered",
				"code":  "EMAIL_ALREADY_EXISTS",
			})
			return
		}

		// Hash password — bcrypt cost 12.
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error", "code": "INTERNAL_ERROR"})
			return
		}
		hashStr := string(hash)

		// Generate numeric OTP code of the configured length.
		code, err := services.GenerateVerifyCode(codeLength)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate verification code", "code": "INTERNAL_ERROR"})
			return
		}
		expiry := time.Now().Add(15 * time.Minute) // OTP expires in 15 minutes

		user := models.User{
			ID:                uuid.New(),
			Email:             req.Email,
			PasswordHash:      &hashStr,
			IsVerified:        false,
			VerifyToken:       &code,
			VerifyTokenExpiry: &expiry,
		}

		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user", "code": "INTERNAL_ERROR"})
			return
		}

		// Send OTP email using the site's DB template (falls back to built-in if none set).
		if err := emailSvc.SendVerificationCode(db, site, req.Email, code); err != nil {
			// Non-fatal: account is created. User can request resend (future endpoint).
			// Do not leak internal email error to client.
			_ = err
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "account created — check your email for the verification code",
			"user_id": user.ID,
		})
	}
}

// ─── Login ────────────────────────────────────────────────────────────────────

// Login authenticates with email + password and returns a signed JWT pair (access + refresh)
// containing the site claim for DB routing on subsequent requests.
// POST /auth/login  (requires SiteResolver middleware)
func Login(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.ContentType(), "application/json") {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{
				"error": "Content-Type must be application/json",
				"code":  "VALIDATION_ERROR",
			})
			return
		}

		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
			return
		}

		db := c.MustGet("db").(*gorm.DB)
		site := c.GetString("site")

		var err error
		req.Email, err = normalizeInput(req.Email)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
			return
		}
		req.Password, err = normalizeInput(req.Password)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
			return
		}

		// Generic error for not-found AND wrong password — prevents user enumeration.
		var user models.User
		if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid email or password",
				"code":  "INVALID_CREDENTIALS",
			})
			return
		}

		// Social-only accounts have no password set.
		if user.PasswordHash == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "this account uses social login — please sign in with Google or Twitter",
				"code":  "SOCIAL_ACCOUNT_ONLY",
			})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid email or password",
				"code":  "INVALID_CREDENTIALS",
			})
			return
		}

		if !user.IsVerified {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "email not verified — check your inbox for the verification code",
				"code":  "EMAIL_NOT_VERIFIED",
			})
			return
		}

		// Issue JWT token pair with site claim baked in.
		accessToken, refreshToken, err := services.GenerateTokenPair(
			user.ID,
			user.Email,
			site,
			cfg.JWTSecret,
			cfg.AccessTokenExpiry,
			cfg.RefreshTokenExpiry,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens", "code": "INTERNAL_ERROR"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"user": gin.H{
				"id":    user.ID,
				"email": user.Email,
			},
		})
	}
}

// ─── Verify Email (OTP code) ──────────────────────────────────────────────────

// VerifyEmail validates the numeric OTP code submitted by the user after signup.
//
// Changed from GET ?token=xxx  →  POST {"email":"...","code":"..."}
//   - Requiring email prevents brute-force guessing of the code across all users.
//   - POST keeps the code out of server access logs and browser history.
//
// POST /auth/verify-email  (requires SiteResolver middleware)
func VerifyEmail() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.ContentType(), "application/json") {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{
				"error": "Content-Type must be application/json",
				"code":  "VALIDATION_ERROR",
			})
			return
		}

		var req verifyEmailRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
			return
		}

		var err error
		req.Email, err = normalizeInput(req.Email)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
			return
		}
		req.Code, err = normalizeInput(req.Code)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
			return
		}

		db := c.MustGet("db").(*gorm.DB)

		// Find user by email in this site's DB.
		var user models.User
		if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
			// Generic message — do not reveal whether email exists.
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid email or verification code",
				"code":  "INVALID_TOKEN",
			})
			return
		}

		if user.IsVerified {
			c.JSON(http.StatusOK, gin.H{
				"message": "email already verified — you can log in",
			})
			return
		}

		// Validate stored OTP code.
		if user.VerifyToken == nil || *user.VerifyToken != req.Code {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid email or verification code",
				"code":  "INVALID_TOKEN",
			})
			return
		}

		// Check expiry.
		if user.VerifyTokenExpiry != nil && time.Now().After(*user.VerifyTokenExpiry) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "verification code has expired — please request a new one",
				"code":  "TOKEN_EXPIRED",
			})
			return
		}

		// Mark verified and clear the OTP fields atomically.
		if err := db.Model(&user).Updates(map[string]interface{}{
			"is_verified":         true,
			"verify_token":        nil,
			"verify_token_expiry": nil,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify email", "code": "INTERNAL_ERROR"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "email verified successfully — you can now log in",
		})
	}
}

// ─── Me (Protected) ───────────────────────────────────────────────────────────

// Me returns the authenticated user's profile.
// GET /auth/me  (requires JWTAuth middleware)
func Me() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := c.MustGet("claims").(*services.Claims)
		db := c.MustGet("db").(*gorm.DB)

		if services.IsTokenRevoked(claims.ID) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "token has been revoked",
				"code":  "INVALID_TOKEN",
			})
			return
		}

		var user models.User
		if err := db.First(&user, "id = ?", claims.UserID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "user not found",
				"code":  "INVALID_TOKEN",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"site":       claims.Site,
			"verified":   user.IsVerified,
			"created_at": user.CreatedAt,
		})
	}
}

// ─── Logout (Protected) ───────────────────────────────────────────────────────

// Logout revokes the current JWT by adding its jti to the in-memory blocklist.
// POST /auth/logout  (requires JWTAuth middleware)
func Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := c.MustGet("claims").(*services.Claims)

		expiresAt := time.Now().Add(7 * 24 * time.Hour)
		if claims.ExpiresAt != nil {
			expiresAt = claims.ExpiresAt.Time
		}

		services.RevokeToken(claims.ID, expiresAt)

		c.JSON(http.StatusOK, gin.H{
			"message": "logged out successfully",
		})
	}
}

// ─── Refresh Token ────────────────────────────────────────────────────────────

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh validates a refresh token and issues a new access + refresh token pair.
// The old refresh token is then revoked to prevent reuse.
// POST /auth/refresh  (requires SiteResolver middleware)
func Refresh(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req refreshRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
			return
		}

		site := c.GetString("site")

		// Parse and validate the refresh token specifically
		claims, err := services.ValidateRefreshToken(req.RefreshToken, cfg.JWTSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token", "code": "INVALID_TOKEN"})
			return
		}

		// Ensure the token's site matches the requested site
		if claims.Site != site {
			c.JSON(http.StatusForbidden, gin.H{"error": "token does not belong to this site", "code": "SITE_MISMATCH"})
			return
		}

		if services.IsTokenRevoked(claims.ID) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token has been revoked", "code": "INVALID_TOKEN"})
			return
		}

		db := c.MustGet("db").(*gorm.DB)
		var user models.User
		if err := db.First(&user, "id = ?", claims.UserID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user no longer exists", "code": "USER_NOT_FOUND"})
			return
		}

		// Issue a new token pair
		accessToken, newRefreshToken, err := services.GenerateTokenPair(
			user.ID,
			user.Email,
			site,
			cfg.JWTSecret,
			cfg.AccessTokenExpiry,
			cfg.RefreshTokenExpiry,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens", "code": "INTERNAL_ERROR"})
			return
		}

		// Revoke the old refresh token (Refresh Token Rotation)
		expiresAt := time.Now().Add(cfg.RefreshTokenExpiry)
		if claims.ExpiresAt != nil {
			expiresAt = claims.ExpiresAt.Time
		}
		services.RevokeToken(claims.ID, expiresAt)

		c.JSON(http.StatusOK, gin.H{
			"access_token":  accessToken,
			"refresh_token": newRefreshToken,
		})
	}
}
