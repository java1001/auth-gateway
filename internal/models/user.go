package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents an authenticated user within a single site's database.
// Each site DB has its own independent users table.
type User struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email             string         `gorm:"uniqueIndex;not null"                           json:"email"`
	PasswordHash      *string        `gorm:"column:password_hash"                           json:"-"` // nil for social-only accounts
	IsVerified        bool           `gorm:"default:false;not null"                         json:"is_verified"`
	VerifyToken       *string        `gorm:"index"                                          json:"-"`
	VerifyTokenExpiry *time.Time     `gorm:"column:verify_token_expiry"                     json:"-"`
	OAuthAccounts     []OAuthAccount `gorm:"foreignKey:UserID"                              json:"oauth_accounts,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}
