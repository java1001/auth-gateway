package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/auth-gateway/config"
	"github.com/auth-gateway/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GoogleStateClaims represents the signed CSRF state for Google callbacks.
type GoogleStateClaims struct {
	jwt.RegisteredClaims
}

// TwitterStateEntry stores PKCE verifier state for a single Twitter login attempt.
type TwitterStateEntry struct {
	CodeVerifier string
	ExpiresAt    time.Time
}

// StateStore is a thread-safe in-memory map for Twitter PKCE state.
type StateStore struct {
	mu    sync.RWMutex
	store map[string]TwitterStateEntry
}

func NewStateStore() *StateStore {
	return &StateStore{store: make(map[string]TwitterStateEntry)}
}

func (s *StateStore) Set(nonce string, entry TwitterStateEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[nonce] = entry
}

func (s *StateStore) Get(nonce string) (TwitterStateEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.store[nonce]
	return entry, ok
}

func (s *StateStore) Delete(nonce string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.store, nonce)
}

func (s *StateStore) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			s.mu.Lock()
			for nonce, entry := range s.store {
				if now.After(entry.ExpiresAt) {
					delete(s.store, nonce)
				}
			}
			s.mu.Unlock()
		}
	}()
}

// OAuthService manages Google and Twitter OAuth flows plus token encryption.
type OAuthService struct {
	GoogleConfig            *oauth2.Config
	TwitterConfig           *oauth2.Config
	StateStore              *StateStore
	oauthStateSecret        string
	oauthTokenEncryptionKey string
}

func NewOAuthService(cfg *config.Config) *OAuthService {
	svc := &OAuthService{
		GoogleConfig: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		},
		TwitterConfig: &oauth2.Config{
			ClientID:     cfg.TwitterClientID,
			ClientSecret: cfg.TwitterClientSecret,
			Scopes:       []string{"tweet.read", "users.read", "offline.access"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://twitter.com/i/oauth2/authorize",
				TokenURL: "https://api.twitter.com/2/oauth2/token",
			},
		},
		StateStore:              NewStateStore(),
		oauthStateSecret:        cfg.OAuthStateSecret,
		oauthTokenEncryptionKey: cfg.OAuthTokenEncryptionKey,
	}
	svc.StateStore.StartCleanup(5 * time.Minute)
	return svc
}

func NewGoogleState(oauthStateSecret string) (string, error) {
	claims := GoogleStateClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(oauthStateSecret))
}

func ParseGoogleState(stateToken, oauthStateSecret string) (string, error) {
	token, err := jwt.ParseWithClaims(stateToken, &GoogleStateClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(oauthStateSecret), nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*GoogleStateClaims)
	if !ok || !token.Valid || claims.ID == "" {
		return "", errors.New("invalid google state token")
	}
	return claims.ID, nil
}

func GenerateCodeVerifier() (string, error) {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func GenerateCodeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (s *OAuthService) EncryptToken(plaintext string) (string, error) {
	block, err := aes.NewCipher(deriveAESKey(s.oauthTokenEncryptionKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	payload := append(nonce, sealed...)
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func (s *OAuthService) DecryptToken(ciphertext string) (string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(deriveAESKey(s.oauthTokenEncryptionKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(decoded) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce := decoded[:gcm.NonceSize()]
	data := decoded[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func deriveAESKey(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

type GoogleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func GetGoogleUserInfo(ctx context.Context, token *oauth2.Token, cfg *oauth2.Config) (*GoogleUserInfo, error) {
	client := cfg.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, fmt.Errorf("fetching Google userinfo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google userinfo returned status %d", resp.StatusCode)
	}
	var info GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decoding Google userinfo: %w", err)
	}
	return &info, nil
}

type TwitterUserInfo struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Username        string `json:"username"`
	ProfileImageURL string `json:"profile_image_url"`
}

func GetTwitterUserInfo(ctx context.Context, token *oauth2.Token) (*TwitterUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.twitter.com/2/users/me?user.fields=id,name,username,profile_image_url", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching Twitter userinfo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Twitter userinfo returned status %d", resp.StatusCode)
	}
	var wrapper struct {
		Data TwitterUserInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("decoding Twitter userinfo: %w", err)
	}
	return &wrapper.Data, nil
}

func FindOrCreateUser(
	db *gorm.DB,
	provider, providerUID, email, displayName, avatarURL string,
	token *oauth2.Token,
	encrypt func(string) (string, error),
) (*models.User, error) {
	var resultUser models.User

	err := db.Transaction(func(tx *gorm.DB) error {
		var oauthAccount models.OAuthAccount
		err := tx.Where("provider = ? AND provider_uid = ?", provider, providerUID).First(&oauthAccount).Error
		if err == nil {
			updates := map[string]interface{}{
				"access_token":  "",
				"refresh_token": "",
				"token_expiry":  token.Expiry,
				"display_name":  displayName,
				"avatar_url":    avatarURL,
			}
			if encrypt != nil {
				if token.AccessToken != "" {
					enc, err := encrypt(token.AccessToken)
					if err != nil {
						return err
					}
					updates["access_token"] = enc
				}
				if token.RefreshToken != "" {
					enc, err := encrypt(token.RefreshToken)
					if err != nil {
						return err
					}
					updates["refresh_token"] = enc
				}
			}
			if updateErr := tx.Model(&oauthAccount).Updates(updates).Error; updateErr != nil {
				return updateErr
			}
			return tx.First(&resultUser, "id = ?", oauthAccount.UserID).Error
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}

		var user models.User
		emailErr := tx.Where("email = ?", email).First(&user).Error
		if emailErr == nil {
			account, err := buildOAuthAccount(user.ID, provider, providerUID, email, displayName, avatarURL, token, encrypt)
			if err != nil {
				return err
			}
			if err := tx.Create(&account).Error; err != nil {
				return err
			}
			resultUser = user
			return nil
		}
		if emailErr != gorm.ErrRecordNotFound {
			return emailErr
		}

		newUser := models.User{
			ID:         uuid.New(),
			Email:      email,
			IsVerified: true,
		}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "email"}}, DoNothing: true}).Create(&newUser).Error; err != nil {
			return err
		}
		if err := tx.Where("email = ?", email).First(&newUser).Error; err != nil {
			return err
		}

		account, err := buildOAuthAccount(newUser.ID, provider, providerUID, email, displayName, avatarURL, token, encrypt)
		if err != nil {
			return err
		}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "provider"}, {Name: "provider_uid"}}, DoNothing: true}).Create(&account).Error; err != nil {
			return err
		}
		if err := tx.Where("provider = ? AND provider_uid = ?", provider, providerUID).First(&oauthAccount).Error; err != nil {
			return err
		}
		return tx.First(&resultUser, "id = ?", oauthAccount.UserID).Error
	})

	if err != nil {
		return nil, err
	}
	return &resultUser, nil
}

func buildOAuthAccount(userID uuid.UUID, provider, providerUID, email, displayName, avatarURL string, token *oauth2.Token, encrypt func(string) (string, error)) (models.OAuthAccount, error) {
	account := models.OAuthAccount{
		ID:          uuid.New(),
		UserID:      userID,
		Provider:    provider,
		ProviderUID: providerUID,
		Email:       email,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
		TokenExpiry: token.Expiry,
	}
	if encrypt != nil {
		if token.AccessToken != "" {
			enc, err := encrypt(token.AccessToken)
			if err != nil {
				return models.OAuthAccount{}, err
			}
			account.AccessToken = enc
		}
		if token.RefreshToken != "" {
			enc, err := encrypt(token.RefreshToken)
			if err != nil {
				return models.OAuthAccount{}, err
			}
			account.RefreshToken = enc
		}
	}
	return account, nil
}
