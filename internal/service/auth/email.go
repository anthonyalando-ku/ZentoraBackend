package auth

import (
	"context"
	"fmt"
	"strings"

	"zentora-service/internal/service/email"

	"go.uber.org/zap"
)

type EmailHelper struct {
	sender  *email.EmailSender
	logger  *zap.Logger
	baseURL string
}

func NewEmailHelper(sender *email.EmailSender, logger *zap.Logger, baseURL string) *EmailHelper {
	return &EmailHelper{sender: sender, logger: logger, baseURL: baseURL}
}

// =============================================================================
// Password Reset
// =============================================================================

func (h *EmailHelper) PasswordResetEmail(fullName, token string) (string, string) {
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", h.baseURL, token)
	subject := "Reset Your Zentora Password"
	body := fmt.Sprintf(`
<h2>Password Reset Request</h2>
<p>Hi <strong>%s</strong>,</p>
<p>We received a request to reset the password for your Zentora account.
   Click the button below to choose a new password.</p>
<p style="text-align:center;">
  <a href="%s" class="btn-primary">Reset My Password</a>
</p>
<p>Or paste this link into your browser:</p>
<p><a href="%s" style="color:#004b8f;word-break:break-all;">%s</a></p>
<div class="box box-warning">
  <strong>Security notice</strong>
  <ul>
    <li>This link expires in <strong>1 hour</strong></li>
    <li>If you did not request a reset, simply ignore this email</li>
    <li>Never share this link with anyone</li>
  </ul>
</div>
`, fullName, resetURL, resetURL, resetURL)
	return subject, body
}

func (h *EmailHelper) SendPasswordResetEmail(ctx context.Context, toEmail, fullName, token string) {
	go func() {
		subject, body := h.PasswordResetEmail(fullName, token)
		if err := h.sender.Send(toEmail, subject, body); err != nil {
			h.logger.Error("failed to send password reset email", zap.String("email", toEmail), zap.Error(err))
		} else {
			h.logger.Info("password reset email sent", zap.String("email", toEmail))
		}
	}()
}

// =============================================================================
// Email Verification (link-based)
// =============================================================================

func (h *EmailHelper) EmailVerificationEmail(fullName, token string) (string, string) {
	verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", h.baseURL, token)
	subject := "Verify Your Zentora Email Address"
	body := fmt.Sprintf(`
<h2>Confirm Your Email Address</h2>
<p>Hi <strong>%s</strong>,</p>
<p>Thanks for signing up for Zentora! Please verify your email address to
   activate your account.</p>
<p style="text-align:center;">
  <a href="%s" class="btn-primary">Verify Email Address</a>
</p>
<p>Or paste this link into your browser:</p>
<p><a href="%s" style="color:#004b8f;word-break:break-all;">%s</a></p>
<div class="box box-info">
  This link expires in <strong>24 hours</strong>.
  If you did not create a Zentora account you can safely ignore this email.
</div>
`, fullName, verifyURL, verifyURL, verifyURL)
	return subject, body
}

func (h *EmailHelper) SendEmailVerification(ctx context.Context, toEmail, fullName, token string) {
	go func() {
		subject, body := h.EmailVerificationEmail(fullName, token)
		if err := h.sender.Send(toEmail, subject, body); err != nil {
			h.logger.Error("failed to send email verification", zap.String("email", toEmail), zap.Error(err))
		} else {
			h.logger.Info("email verification sent", zap.String("email", toEmail))
		}
	}()
}

// =============================================================================
// OTP Verification
// =============================================================================

func (h *EmailHelper) OTPEmail(otp string) (string, string) {
	subject := "Your Zentora Verification Code"
	body := fmt.Sprintf(`
<h2>Email Verification Code</h2>
<p>Use the code below to verify your email address. Do not share it with anyone.</p>
<div class="otp-wrapper">
  <div class="otp-code">%s</div>
  <p style="margin-top:12px;font-size:13px;color:#6b7280;">Valid for 10 minutes</p>
</div>
<div class="box box-warning">
  <strong>Security notice</strong>
  <ul>
    <li>This code expires in <strong>10 minutes</strong></li>
    <li>Never share this code with anyone — Zentora staff will never ask for it</li>
    <li>If you did not request this code, ignore this email</li>
  </ul>
</div>
`, otp)
	return subject, body
}

func (h *EmailHelper) SendOTPEmail(ctx context.Context, toEmail, otp string) error {
	subject, body := h.OTPEmail(otp)
	if err := h.sender.Send(toEmail, subject, body); err != nil {
		h.logger.Error("failed to send OTP email", zap.String("email", toEmail), zap.Error(err))
		return err
	}
	h.logger.Info("OTP email sent", zap.String("email", toEmail))
	return nil
}

// =============================================================================
// Welcome Email
// =============================================================================

func (h *EmailHelper) WelcomeEmail(fullName, userEmail string) (string, string) {
	loginURL := fmt.Sprintf("%s/auth/login", h.baseURL)
	subject := "Welcome to Zentora!"
	body := fmt.Sprintf(`
<h2>Welcome to Zentora, %s!</h2>
<p>Your account has been verified and is ready to use.</p>
<hr class="divider" />
<div class="box box-success">
  <strong>Getting started</strong>
  <ul>
    <li>Complete your profile</li>
    <li>Explore the product catalog</li>
    <li>Check out current promotions</li>
  </ul>
</div>
<p style="text-align:center;">
  <a href="%s" class="btn-primary">Go to Zentora</a>
</p>
<p style="font-size:13px;color:#6b7280;">
  Logged in with: <strong>%s</strong>
</p>
`, fullName, loginURL, userEmail)
	return subject, body
}

func (h *EmailHelper) SendWelcomeEmail(ctx context.Context, toEmail, fullName string) {
	go func() {
		subject, body := h.WelcomeEmail(fullName, toEmail)
		if err := h.sender.Send(toEmail, subject, body); err != nil {
			h.logger.Error("failed to send welcome email", zap.String("email", toEmail), zap.Error(err))
		} else {
			h.logger.Info("welcome email sent", zap.String("email", toEmail))
		}
	}()
}

// =============================================================================
// Account Created by Admin
// =============================================================================

func (h *EmailHelper) AccountCreatedByAdminEmail(fullName, userEmail, temporaryPassword string, roles []string) (string, string) {
	loginURL := fmt.Sprintf("%s/auth/login", h.baseURL)
	rolesStr := strings.Join(roles, ", ")
	if rolesStr == "" {
		rolesStr = "User"
	}
	subject := "Your Zentora Account Has Been Created"
	body := fmt.Sprintf(`
<h2>Your Account Is Ready</h2>
<p>Hi <strong>%s</strong>,</p>
<p>An administrator has set up a Zentora account for you.
   Use the credentials below to sign in for the first time.</p>
<div class="credentials">
  <p><strong>Email:</strong> %s</p>
  <p><strong>Temporary password:</strong> <code>%s</code></p>
  <p><strong>Role(s):</strong> %s</p>
</div>
<div class="box box-danger">
  <strong>Action required</strong>
  <ol>
    <li>Log in using the temporary password above</li>
    <li>Change your password immediately on first login</li>
    <li>Do not share your credentials with anyone</li>
  </ol>
</div>
<p style="text-align:center;">
  <a href="%s" class="btn-primary">Log In Now</a>
</p>
<p style="font-size:13px;color:#6b7280;">
  If you believe this account was created by mistake, contact your administrator.
</p>
`, fullName, userEmail, temporaryPassword, rolesStr, loginURL)
	return subject, body
}

func (h *EmailHelper) SendAccountCreatedByAdmin(ctx context.Context, toEmail, fullName, temporaryPassword string, roles []string) {
	go func() {
		subject, body := h.AccountCreatedByAdminEmail(fullName, toEmail, temporaryPassword, roles)
		if err := h.sender.Send(toEmail, subject, body); err != nil {
			h.logger.Error("failed to send account created email", zap.String("email", toEmail), zap.Error(err))
		} else {
			h.logger.Info("account created email sent", zap.String("email", toEmail))
		}
	}()
}

// =============================================================================
// Password Changed Notification
// =============================================================================

func (h *EmailHelper) PasswordChangedEmail(fullName string) (string, string) {
	subject := "Your Zentora Password Was Changed"
	body := fmt.Sprintf(`
<h2>Password Changed Successfully</h2>
<p>Hi <strong>%s</strong>,</p>
<div class="box box-success">
  Your password was changed successfully.
  All existing sessions have been logged out for your security.
</div>
<div class="box box-danger">
  <strong>Did not make this change?</strong><br />
  If you did not change your password, contact us immediately at
  <a href="mailto:support@zentora.com" style="color:#004b8f;">support@zentora.com</a>
</div>
`, fullName)
	return subject, body
}

func (h *EmailHelper) SendPasswordChangedNotification(ctx context.Context, toEmail, fullName string) {
	go func() {
		subject, body := h.PasswordChangedEmail(fullName)
		if err := h.sender.Send(toEmail, subject, body); err != nil {
			h.logger.Error("failed to send password changed notification", zap.String("email", toEmail), zap.Error(err))
		} else {
			h.logger.Info("password changed notification sent", zap.String("email", toEmail))
		}
	}()
}

// =============================================================================
// Helpers
// =============================================================================

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}