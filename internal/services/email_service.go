package services

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

// EmailService wraps the Resend client for transactional emails.
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

// SendVerificationEmail sends an email verification link to the given address.
// The verifyURL should be the full URL including the token query parameter.
func (s *EmailService) SendVerificationEmail(to, verifyURL string) error {
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Verify your email</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f4f4f5; margin: 0; padding: 40px 0; }
    .container { max-width: 480px; margin: 0 auto; background: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 24px rgba(0,0,0,0.08); }
    .header { background: linear-gradient(135deg, #6366f1 0%%, #8b5cf6 100%%); padding: 32px; text-align: center; }
    .header h1 { color: #ffffff; margin: 0; font-size: 24px; font-weight: 700; }
    .body { padding: 40px 32px; }
    .body p { color: #374151; font-size: 16px; line-height: 1.6; margin: 0 0 24px; }
    .btn { display: inline-block; background: linear-gradient(135deg, #6366f1 0%%, #8b5cf6 100%%); color: #ffffff !important; text-decoration: none; padding: 14px 32px; border-radius: 8px; font-weight: 600; font-size: 16px; }
    .footer { padding: 24px 32px; border-top: 1px solid #f0f0f0; }
    .footer p { color: #9ca3af; font-size: 13px; margin: 0; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>Verify Your Email</h1>
    </div>
    <div class="body">
      <p>Thanks for signing up! Please verify your email address to get started.</p>
      <p>This link expires in <strong>24 hours</strong>.</p>
      <p style="text-align:center">
        <a href="%s" class="btn">Verify Email Address</a>
      </p>
      <p style="font-size:14px;color:#6b7280;">If the button doesn't work, copy and paste this link:<br>
        <a href="%s" style="color:#6366f1;word-break:break-all">%s</a>
      </p>
    </div>
    <div class="footer">
      <p>If you didn't create an account, you can safely ignore this email.</p>
    </div>
  </div>
</body>
</html>`, verifyURL, verifyURL, verifyURL)

	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: "Verify your email address",
		Html:    htmlBody,
	}

	_, err := s.client.Emails.SendWithContext(context.Background(), params)
	if err != nil {
		return fmt.Errorf("sending verification email to %s: %w", to, err)
	}
	return nil
}
