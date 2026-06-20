package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/auth-gateway/internal/models"
	"github.com/resend/resend-go/v2"
	"gorm.io/gorm"
)

// TemplateKey is the unique key used to look up an email template in the DB.
type TemplateKey = string

const (
	// TemplateKeyVerifyEmail is the template used for the OTP verification email.
	TemplateKeyVerifyEmail TemplateKey = "verify_email"
)

// EmailService wraps the Resend client and renders templates from the per-site DB.
type EmailService struct {
	client *resend.Client
	from   string
}

// NewEmailService creates a new EmailService using the provided Resend API key.
func NewEmailService(apiKey, from string) *EmailService {
	return &EmailService{
		client: resend.NewClient(apiKey),
		from:   from,
	}
}

// SendVerificationCode sends a numeric OTP verification code to the user.
//
// The subject and HTML body are loaded from the site's email_templates table
// using key = "verify_email". Placeholders in the template:
//
//	{{code}}  — the numeric code (e.g. "48291736")
//	{{email}} — the recipient email address
//	{{site}}  — the site hostname (e.g. "site1.com")
//
// If no template exists for this site+key, a built-in fallback template is used
// so the service degrades gracefully on first deploy.
func (s *EmailService) SendVerificationCode(db *gorm.DB, site, to, code string) error {
	subject, htmlBody, err := s.resolveTemplate(db, site, to, code)
	if err != nil {
		return fmt.Errorf("resolving email template: %w", err)
	}

	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: subject,
		Html:    htmlBody,
	}

	_, err = s.client.Emails.SendWithContext(context.Background(), params)
	if err != nil {
		return fmt.Errorf("sending verification email to %s: %w", to, err)
	}
	return nil
}

// resolveTemplate fetches the template from the DB for (site, key="verify_email").
// Falls back to the built-in default if no DB record is found.
func (s *EmailService) resolveTemplate(db *gorm.DB, site, recipientEmail, code string) (subject, html string, err error) {
	var tmpl models.EmailTemplate
	dbErr := db.Where("site = ? AND key = ?", site, TemplateKeyVerifyEmail).First(&tmpl).Error

	if dbErr != nil && !errors.Is(dbErr, gorm.ErrRecordNotFound) {
		return "", "", fmt.Errorf("querying email template: %w", dbErr)
	}

	if errors.Is(dbErr, gorm.ErrRecordNotFound) {
		// No custom template — use the built-in fallback.
		subject = "Your verification code"
		html = defaultVerifyTemplate()
	} else {
		subject = tmpl.Subject
		html = tmpl.Content
	}

	// Render placeholders.
	replacer := strings.NewReplacer(
		"{{code}}", code,
		"{{email}}", recipientEmail,
		"{{site}}", site,
	)
	subject = replacer.Replace(subject)
	html = replacer.Replace(html)
	return subject, html, nil
}

// defaultVerifyTemplate returns the built-in HTML fallback used when no DB
// template is configured for a site. It is intentionally simple so operators
// can always override it via the email_templates table.
func defaultVerifyTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Verify your email</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f4f4f5; margin: 0; padding: 40px 0; }
    .container { max-width: 480px; margin: 0 auto; background: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 24px rgba(0,0,0,0.08); }
    .header { background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%); padding: 32px; text-align: center; }
    .header h1 { color: #ffffff; margin: 0; font-size: 24px; font-weight: 700; }
    .body { padding: 40px 32px; text-align: center; }
    .body p { color: #374151; font-size: 16px; line-height: 1.6; margin: 0 0 24px; }
    .code-box { display: inline-block; background: #f3f4f6; border: 2px dashed #6366f1; border-radius: 12px; padding: 20px 40px; margin: 8px 0 24px; }
    .code-box span { font-size: 36px; font-weight: 800; letter-spacing: 8px; color: #4f46e5; font-family: 'Courier New', monospace; }
    .expiry { font-size: 14px; color: #6b7280; }
    .footer { padding: 24px 32px; border-top: 1px solid #f0f0f0; }
    .footer p { color: #9ca3af; font-size: 13px; margin: 0; text-align: center; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>Verify Your Email</h1>
    </div>
    <div class="body">
      <p>Thanks for signing up! Enter the code below to verify your email address.</p>
      <div class="code-box">
        <span>{{code}}</span>
      </div>
      <p class="expiry">This code expires in <strong>15 minutes</strong>.</p>
      <p style="font-size:14px;color:#6b7280;">Do not share this code with anyone.</p>
    </div>
    <div class="footer">
      <p>If you didn't create an account, you can safely ignore this email.</p>
    </div>
  </div>
</body>
</html>`
}
