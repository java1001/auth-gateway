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

// ─── Request / Response types ─────────────────────────────────────────────────

type signupRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func normalizeInput(value string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.ContainsRune(value, '\x00') {
		return "", fmt.Errorf("input contains invalid null byte")
	}
	return value, nil
}

// ─── Signup ───────────────────────────────────────────────────────────────────

// Signup registers a new user with email + password in the site-specific DB.
// It sends a verification email via Resend before returning.
// POST /auth/signup  (requires SiteResolver middleware)
func Signup(emailSvc *services.EmailService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Enforce JSON content type.
		if !strings.Contains(c.ContentType(), "application/json") {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{
				"error": "Content-Type must be application/json",
				"code":  "VALIDATION_ERROR",
			})
			return
		}

		var req signupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
				"code":  "VALIDATION_ERROR",
			})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "password must be 72 characters or fewer", "code": "VALIDATION_ERROR"})
			return
		}

		// Check for duplicate email in this site's DB.
		var existing models.User
		if err := db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"error": "email already registered",
				"code":  "EMAIL_ALREADY_EXISTS",
			})
			return
		}

		// Hash password with bcrypt cost 12.
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error", "code": "INTERNAL_ERROR"})
			return
		}
		hashStr := string(hash)

		// Generate a one-time email verification token.
		verifyToken := uuid.NewString()
		expiry := time.Now().Add(24 * time.Hour)

		user := models.User{
			ID:                uuid.New(),
			Email:             req.Email,
			PasswordHash:      &hashStr,
			IsVerified:        false,
			VerifyToken:       &verifyToken,
			VerifyTokenExpiry: &expiry,
		}

		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user", "code": "INTERNAL_ERROR"})
			return
		}

		// Send verification email to the site's own host so SiteResolver can
		// recover the correct tenant DB from the Host header.
		verifyURL := "https://" + site + "/auth/verify-email?token=" + verifyToken
		if err := emailSvc.SendVerificationEmail(req.Email, verifyURL); err != nil {
			// Non-fatal: user was created. They can request a new verification email.
			// Log but do not surface internal email error to client.
			_ = err
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "account created — check your email to verify your address",
			"user_id": user.ID,
		})
	}
}

// ─── Login ────────────────────────────────────────────────────────────────────

// Login authenticates a user with email + password and returns a signed JWT
// containing the site claim for DB routing on subsequent requests.
// POST /auth/login  (requires SiteResolver middleware)
func Login(jwtSecret string) gin.HandlerFunc {
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
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
				"code":  "VALIDATION_ERROR",
			})
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

		// Look up user — return the same generic error whether not found or wrong password
		// to prevent user enumeration.
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
				"error": "email address not verified — check your inbox",
				"code":  "EMAIL_NOT_VERIFIED",
			})
			return
		}

		// Issue JWT with site claim baked in.
		token, err := services.GenerateJWT(user.ID, user.Email, site, jwtSecret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token", "code": "INTERNAL_ERROR"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"user": gin.H{
				"id":    user.ID,
				"email": user.Email,
			},
		})
	}
}

// ─── Verify Email ─────────────────────────────────────────────────────────────

// VerifyEmail processes the email verification link sent during signup.
// GET /auth/verify-email?token=xxx  (requires SiteResolver middleware)
func VerifyEmail() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimSpace(c.Query("token"))
		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "missing token parameter",
				"code":  "INVALID_TOKEN",
			})
			return
		}

		db := c.MustGet("db").(*gorm.DB)

		var user models.User
		if err := db.Where("verify_token = ?", token).First(&user).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid verification token",
				"code":  "INVALID_TOKEN",
			})
			return
		}

		if user.VerifyTokenExpiry != nil && time.Now().After(*user.VerifyTokenExpiry) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "verification token has expired — please sign up again",
				"code":  "TOKEN_EXPIRED",
			})
			return
		}

		// Mark user as verified and clear token fields.
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
// GET /auth/me  (requires JWTAuth middleware — db and claims come from JWT site claim)
func Me() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := c.MustGet("claims").(*services.Claims)
		db := c.MustGet("db").(*gorm.DB)

		// Check if token has been revoked via logout.
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

		var expiresAt time.Time
		if claims.ExpiresAt != nil {
			expiresAt = claims.ExpiresAt.Time
		} else {
			expiresAt = time.Now().Add(7 * 24 * time.Hour)
		}

		services.RevokeToken(claims.ID, expiresAt)

		c.JSON(http.StatusOK, gin.H{
			"message": "logged out successfully",
		})
	}
}
