package config

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SiteDBConfig holds the connection details for one site's isolated database.
type SiteDBConfig struct {
	Host   string
	DSN    string
	DBName string
}

// Config is the runtime application configuration loaded from environment variables.
type Config struct {
	Port                    string
	JWTSecret               string
	OAuthStateSecret        string
	OAuthTokenEncryptionKey string
	ResendAPIKey            string
	EmailFrom               string
	GatewayBaseURL          string
	AllowSites              []string
	GoogleClientID          string
	GoogleClientSecret      string
	TwitterClientID         string
	TwitterClientSecret     string
	Sites                   map[string]SiteDBConfig
	// VerifyCodeLength is the number of digits in the email verification code.
	// Configurable via VERIFY_CODE_LENGTH env var (default: 8, range: 4–10).
	VerifyCodeLength        int
	// AccessTokenExpiry is the duration the access token is valid for (default 720h = 30 days)
	AccessTokenExpiry       time.Duration
	// RefreshTokenExpiry is the duration the refresh token is valid for (default 2160h = 90 days)
	RefreshTokenExpiry      time.Duration
}

// Load parses and validates all environment variables required by the gateway.
// It returns an error instead of panicking so main() can fail fast with context.
func Load() (*Config, error) {
	allowSites, err := parseAllowSites()
	if err != nil {
		return nil, err
	}

	sites, err := parseSites()
	if err != nil {
		return nil, err
	}

	if err := validateSiteSets(allowSites, sites); err != nil {
		return nil, err
	}

	jwtSecret, err := requireEnv("JWT_SECRET")
	if err != nil {
		return nil, err
	}
	oauthStateSecret, err := requireEnv("OAUTH_STATE_SECRET")
	if err != nil {
		return nil, err
	}
	oauthTokenEncryptionKey, err := requireEnv("OAUTH_TOKEN_ENCRYPTION_KEY")
	if err != nil {
		return nil, err
	}
	resendAPIKey, err := requireEnv("RESEND_API_KEY")
	if err != nil {
		return nil, err
	}
	emailFrom, err := requireEnv("EMAIL_FROM")
	if err != nil {
		return nil, err
	}
	gatewayBaseURL, err := requireEnv("GATEWAY_BASE_URL")
	if err != nil {
		return nil, err
	}
	googleClientID, err := requireEnv("GOOGLE_CLIENT_ID")
	if err != nil {
		return nil, err
	}
	googleClientSecret, err := requireEnv("GOOGLE_CLIENT_SECRET")
	if err != nil {
		return nil, err
	}
	twitterClientID, err := requireEnv("TWITTER_CLIENT_ID")
	if err != nil {
		return nil, err
	}
	twitterClientSecret, err := requireEnv("TWITTER_CLIENT_SECRET")
	if err != nil {
		return nil, err
	}
	if jwtSecret == oauthStateSecret {
		return nil, fmt.Errorf("JWT_SECRET and OAUTH_STATE_SECRET must be different values")
	}

	verifyCodeLength, err := parseVerifyCodeLength()
	if err != nil {
		return nil, err
	}

	accessTokenExpiry, err := parseDuration("ACCESS_TOKEN_EXPIRY", 30*24*time.Hour)
	if err != nil {
		return nil, err
	}
	refreshTokenExpiry, err := parseDuration("REFRESH_TOKEN_EXPIRY", 90*24*time.Hour)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Port:                    getEnv("APP_PORT", "8080"),
		JWTSecret:               jwtSecret,
		OAuthStateSecret:        oauthStateSecret,
		OAuthTokenEncryptionKey: oauthTokenEncryptionKey,
		ResendAPIKey:            resendAPIKey,
		EmailFrom:               emailFrom,
		GatewayBaseURL:          strings.TrimRight(gatewayBaseURL, "/"),
		AllowSites:              allowSites,
		GoogleClientID:          googleClientID,
		GoogleClientSecret:      googleClientSecret,
		TwitterClientID:         twitterClientID,
		TwitterClientSecret:     twitterClientSecret,
		Sites:                   sites,
		VerifyCodeLength:        verifyCodeLength,
		AccessTokenExpiry:       accessTokenExpiry,
		RefreshTokenExpiry:      refreshTokenExpiry,
	}

	return cfg, nil
}

// parseVerifyCodeLength reads VERIFY_CODE_LENGTH from env and validates the range.
func parseVerifyCodeLength() (int, error) {
	raw := strings.TrimSpace(os.Getenv("VERIFY_CODE_LENGTH"))
	if raw == "" {
		return 8, nil // default
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 4 || n > 10 {
		return 0, fmt.Errorf("VERIFY_CODE_LENGTH must be a number between 4 and 10 (got %q)", raw)
	}
	return n, nil
}

// parseDuration parses a time.Duration from the env.
func parseDuration(key string, defaultVal time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultVal, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration (e.g. 720h), got %q", key, raw)
	}
	return d, nil
}

func parseAllowSites() ([]string, error) {
	raw := strings.TrimSpace(os.Getenv("ALLOW_SITES"))
	if raw == "" {
		return nil, fmt.Errorf("ALLOW_SITES is required")
	}

	set := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		host := strings.ToLower(strings.TrimSpace(part))
		if host == "" {
			continue
		}
		set[host] = struct{}{}
	}

	if len(set) == 0 {
		return nil, fmt.Errorf("ALLOW_SITES must contain at least one hostname")
	}

	hosts := make([]string, 0, len(set))
	for host := range set {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)
	return hosts, nil
}

func parseSites() (map[string]SiteDBConfig, error) {
	sites := make(map[string]SiteDBConfig)

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		if !strings.HasPrefix(key, "SITE_") || !strings.HasSuffix(key, "_HOST") {
			continue
		}

		label := strings.TrimSuffix(strings.TrimPrefix(key, "SITE_"), "_HOST")
		host := strings.ToLower(strings.TrimSpace(parts[1]))
		if host == "" {
			return nil, fmt.Errorf("SITE_%s_HOST is set but empty", label)
		}

		dsnKey := fmt.Sprintf("SITE_%s_DSN", label)
		dsn := strings.TrimSpace(os.Getenv(dsnKey))
		if dsn == "" {
			return nil, fmt.Errorf("%s is required when SITE_%s_HOST is set", dsnKey, label)
		}

		if existing, exists := sites[host]; exists {
			return nil, fmt.Errorf("duplicate site host %q configured by multiple SITE_* env vars (existing DSN %q, new DSN %q)", host, existing.DSN, dsn)
		}

		dbName := extractDBName(dsn)
		sites[host] = SiteDBConfig{Host: host, DSN: dsn, DBName: dbName}
	}

	if len(sites) == 0 {
		return nil, fmt.Errorf("no SITE_*_HOST / SITE_*_DSN pairs were configured")
	}

	return sites, nil
}

func validateSiteSets(allowSites []string, sites map[string]SiteDBConfig) error {
	allowSet := make(map[string]struct{}, len(allowSites))
	for _, host := range allowSites {
		allowSet[strings.ToLower(host)] = struct{}{}
	}

	siteSet := make(map[string]struct{}, len(sites))
	for host := range sites {
		siteSet[strings.ToLower(host)] = struct{}{}
	}

	for host := range siteSet {
		if _, ok := allowSet[host]; !ok {
			return fmt.Errorf("config mismatch: site %q is registered in SITE_* but missing from ALLOW_SITES", host)
		}
	}

	for host := range allowSet {
		if _, ok := siteSet[host]; !ok {
			return fmt.Errorf("config mismatch: site %q is present in ALLOW_SITES but missing from SITE_*", host)
		}
	}

	return nil
}

func extractDBName(dsn string) string {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(parsed.Path, "/")
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func requireEnv(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("required environment variable %q is not set", key)
	}
	return value, nil
}
