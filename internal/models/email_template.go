package models

import (
	"time"
)

// EmailTemplate stores per-site email templates managed from the database.
// Each record is uniquely identified by (site, key).
//
// Supported template keys:
//   - "verify_email" — sent after signup with the verification code
//
// Content supports the following placeholders (replaced at send time):
//   - {{code}}    — the numeric verification code
//   - {{email}}   — the recipient's email address
//   - {{site}}    — the site hostname (e.g. "site1.com")
type EmailTemplate struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"   json:"id"`
	Site      string    `gorm:"not null;uniqueIndex:idx_site_key;size:255" json:"site"`
	Key       string    `gorm:"not null;uniqueIndex:idx_site_key;size:100" json:"key"`
	Subject   string    `gorm:"not null;size:500"          json:"subject"`
	Content   string    `gorm:"not null;type:text"         json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
