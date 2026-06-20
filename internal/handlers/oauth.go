package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/auth-gateway/config"
	"github.com/auth-gateway/internal/database"
	"github.com/auth-gateway/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

func siteAllowed(site string, cfg *config.Config) bool {
	site = strings.ToLower(strings.TrimSpace(site))
	for _, allowed := range cfg.AllowSites {
		if strings.EqualFold(allowed, site) {
			return true
		}
	}
	return false
}

func rejectSite(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"error": "origin not allowed",
		"code":  "ORIGIN_NOT_ALLOWED",
	})
}

func safeRedirectURL(site, accessToken, refreshToken, provider string, cfg *config.Config) (string, error) {
	if !siteAllowed(site, cfg) {
		return "", fmt.Errorf("redirect target %q is not a registered site", site)
	}
	return fmt.Sprintf("https://%s/auth/callback#access_token=%s&refresh_token=%s&provider=%s", site, accessToken, refreshToken, provider), nil
}

func GoogleLogin(oauthSvc *services.OAuthService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		site := strings.ToLower(strings.TrimSpace(c.Param("site")))
		if !siteAllowed(site, cfg) {
			rejectSite(c)
			return
		}

		stateToken, err := services.NewGoogleState(cfg.OAuthStateSecret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create OAuth state", "code": "INTERNAL_ERROR"})
			return
		}

		redirectURI := cfg.GatewayBaseURL + "/auth/google/" + site + "/callback"
		authURL := oauthSvc.GoogleConfig.AuthCodeURL(
			stateToken,
			oauth2.SetAuthURLParam("redirect_uri", redirectURI),
			oauth2.AccessTypeOffline,
			oauth2.SetAuthURLParam("prompt", "select_account"),
		)
		c.Redirect(http.StatusFound, authURL)
	}
}

func GoogleCallback(oauthSvc *services.OAuthService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		site := strings.ToLower(strings.TrimSpace(c.Param("site")))
		if !siteAllowed(site, cfg) {
			rejectSite(c)
			return
		}

		stateToken := c.Query("state")
		code := c.Query("code")
		if stateToken == "" || code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing state or code parameter", "code": "OAUTH_STATE_INVALID"})
			return
		}

		if _, err := services.ParseGoogleState(stateToken, cfg.OAuthStateSecret); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid OAuth state", "code": "OAUTH_STATE_INVALID"})
			return
		}

		redirectURI := cfg.GatewayBaseURL + "/auth/google/" + site + "/callback"
		token, err := oauthSvc.GoogleConfig.Exchange(
			context.Background(),
			code,
			oauth2.SetAuthURLParam("redirect_uri", redirectURI),
		)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to exchange authorization code", "code": "OAUTH_TOKEN_EXCHANGE_FAILED"})
			return
		}

		userInfo, err := services.GetGoogleUserInfo(context.Background(), token, oauthSvc.GoogleConfig)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to retrieve user info from Google", "code": "OAUTH_USERINFO_FAILED"})
			return
		}
		if !userInfo.EmailVerified {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Google account email is not verified", "code": "OAUTH_UNVERIFIED_EMAIL"})
			return
		}

		db, err := database.GetDB(site)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "site not found", "code": "SITE_NOT_FOUND"})
			return
		}

		user, err := services.FindOrCreateUser(db, "google", userInfo.Sub, userInfo.Email, userInfo.Name, userInfo.Picture, token, oauthSvc.EncryptToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create or retrieve user", "code": "INTERNAL_ERROR"})
			return
		}

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

		redirectURL, err := safeRedirectURL(site, accessToken, refreshToken, "google", cfg)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid redirect target", "code": "SITE_NOT_FOUND"})
			return
		}
		c.Redirect(http.StatusFound, redirectURL)
	}
}

func TwitterLogin(oauthSvc *services.OAuthService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		site := strings.ToLower(strings.TrimSpace(c.Param("site")))
		if !siteAllowed(site, cfg) {
			rejectSite(c)
			return
		}

		codeVerifier, err := services.GenerateCodeVerifier()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate PKCE verifier", "code": "INTERNAL_ERROR"})
			return
		}
		codeChallenge := services.GenerateCodeChallenge(codeVerifier)
		nonce := uuid.NewString()
		oauthSvc.StateStore.Set(nonce, services.TwitterStateEntry{
			CodeVerifier: codeVerifier,
			ExpiresAt:    time.Now().Add(10 * time.Minute),
		})

		redirectURI := cfg.GatewayBaseURL + "/auth/twitter/" + site + "/callback"
		authURL := oauthSvc.TwitterConfig.AuthCodeURL(
			nonce,
			oauth2.SetAuthURLParam("redirect_uri", redirectURI),
			oauth2.SetAuthURLParam("code_challenge", codeChallenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		)
		c.Redirect(http.StatusFound, authURL)
	}
}

func TwitterCallback(oauthSvc *services.OAuthService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		site := strings.ToLower(strings.TrimSpace(c.Param("site")))
		if !siteAllowed(site, cfg) {
			rejectSite(c)
			return
		}

		nonce := c.Query("state")
		code := c.Query("code")
		if nonce == "" || code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing state or code parameter", "code": "OAUTH_STATE_INVALID"})
			return
		}

		entry, ok := oauthSvc.StateStore.Get(nonce)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or already used OAuth state", "code": "OAUTH_STATE_INVALID"})
			return
		}
		oauthSvc.StateStore.Delete(nonce)
		if time.Now().After(entry.ExpiresAt) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "OAuth state has expired, please try again", "code": "OAUTH_STATE_EXPIRED"})
			return
		}

		redirectURI := cfg.GatewayBaseURL + "/auth/twitter/" + site + "/callback"
		token, err := oauthSvc.TwitterConfig.Exchange(
			context.Background(),
			code,
			oauth2.SetAuthURLParam("redirect_uri", redirectURI),
			oauth2.SetAuthURLParam("code_verifier", entry.CodeVerifier),
		)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to exchange authorization code", "code": "OAUTH_TOKEN_EXCHANGE_FAILED"})
			return
		}

		userInfo, err := services.GetTwitterUserInfo(context.Background(), token)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to retrieve user info from Twitter", "code": "OAUTH_USERINFO_FAILED"})
			return
		}

		email := fmt.Sprintf("%s@twitter.noemail", userInfo.Username)

		db, err := database.GetDB(site)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "site not found", "code": "SITE_NOT_FOUND"})
			return
		}

		user, err := services.FindOrCreateUser(db, "twitter", userInfo.ID, email, userInfo.Name, userInfo.ProfileImageURL, token, oauthSvc.EncryptToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create or retrieve user", "code": "INTERNAL_ERROR"})
			return
		}

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

		redirectURL, err := safeRedirectURL(site, accessToken, refreshToken, "twitter", cfg)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid redirect target", "code": "SITE_NOT_FOUND"})
			return
		}
		c.Redirect(http.StatusFound, redirectURL)
	}
}
