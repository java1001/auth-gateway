package models

import (
	"time"

	"github.com/google/uuid"
)

// OAuthAccount stores a linked social provider account for a User.
// A single User may have multiple OAuthAccounts (one per provider).
type OAuthAccount struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;index;not null"                       json:"user_id"`
	Provider     string    `gorm:"not null;uniqueIndex:idx_provider_uid"           json:"provider"`     // "google" | "twitter"
	ProviderUID  string    `gorm:"not null;uniqueIndex:idx_provider_uid"           json:"provider_uid"` // provider's user ID
	Email        string    `gorm:"column:email"                                   json:"email"`
	DisplayName  string    `gorm:"column:display_name"                            json:"display_name"`
	AvatarURL    string    `gorm:"column:avatar_url"                              json:"avatar_url"`
	AccessToken  string    `gorm:"column:access_token"                            json:"-"` // never expose in API responses
	RefreshToken string    `gorm:"column:refresh_token"                           json:"-"`
	TokenExpiry  time.Time `gorm:"column:token_expiry"                            json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
