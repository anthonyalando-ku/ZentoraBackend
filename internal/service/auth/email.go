// internal/usecase/auth/email_helpers.go
package auth

import (
	"context"
	"fmt"
	"strings"

	"diary-service/internal/service/email"

	"go.uber.org/zap"
)

// EmailHelper handles email template generation and sending
type EmailHelper struct {
	sender  *email.EmailSender
	logger  *zap.Logger
	baseURL string
}

func NewEmailHelper(sender *email.EmailSender, logger *zap.Logger, baseURL string) *EmailHelper {
	return &EmailHelper{
		sender:  sender,
		logger:  logger,
		baseURL: baseURL,
	}
}

// ========== Password Reset ==========

// PasswordResetEmail builds a password reset email
func (h *EmailHelper) PasswordResetEmail(fullName, token string) (string, string) {
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", h.baseURL, token)

	subject := "Password Reset Request - TaskaApp"
	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.button { 
					display: inline-block; 
					padding: 12px 24px; 
					background-color: #4CAF50; 
					color: white; 
					text-decoration: none; 
					border-radius: 4px; 
					margin: 20px 0;
				}
				.warning { color: #856404; background-color: #fff3cd; padding: 12px; border-radius: 4px; }
				.footer { margin-top: 30px; font-size: 12px; color: #666; }
			</style>
		</head>
		<body>
			<div class="container">
				<h2>Password Reset Request</h2>
				<p>Hello %s,</p>
				<p>We received a request to reset your password for your TaskaApp account.</p>
				<p>Click the button below to reset your password:</p>
				<a href="%s" class="button">Reset Password</a>
				<p>Or copy and paste this link into your browser:</p>
				<p><a href="%s">%s</a></p>
				<div class="warning">
					<strong>⚠️ Security Notice:</strong>
					<ul>
						<li>This link will expire in 1 hour</li>
						<li>If you didn't request this, please ignore this email</li>
						<li>Never share this link with anyone</li>
					</ul>
				</div>
				<div class="footer">
					<p>If you're having trouble clicking the button, copy and paste the URL into your web browser.</p>
					<p>This is an automated email, please do not reply.</p>
				</div>
			</div>
		</body>
		</html>
	`, fullName, resetURL, resetURL, resetURL)

	return subject, body
}

// SendPasswordResetEmail sends password reset email asynchronously
func (h *EmailHelper) SendPasswordResetEmail(ctx context.Context, email, fullName, token string) {
	go func() {
		subject, body := h.PasswordResetEmail(fullName, token)
		if err := h.sender.Send(email, subject, body); err != nil {
			h.logger.Error("failed to send password reset email",
				zap.String("email", email),
				zap.Error(err),
			)
		} else {
			h.logger.Info("password reset email sent",
				zap.String("email", email),
			)
		}
	}()
}

// ========== Email Verification ==========

// EmailVerificationEmail builds an email verification email
func (h *EmailHelper) EmailVerificationEmail(fullName, token string) (string, string) {
	verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", h.baseURL, token)

	subject := "Verify Your Email - TaskaApp"
	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.button { 
					display: inline-block; 
					padding: 12px 24px; 
					background-color: #2196F3; 
					color: white; 
					text-decoration: none; 
					border-radius: 4px; 
					margin: 20px 0;
				}
				.footer { margin-top: 30px; font-size: 12px; color: #666; }
			</style>
		</head>
		<body>
			<div class="container">
				<h2>Welcome to TaskaApp! 🎉</h2>
				<p>Hello %s,</p>
				<p>Thank you for signing up! Please verify your email address to get started.</p>
				<p>Click the button below to verify your email:</p>
				<a href="%s" class="button">Verify Email</a>
				<p>Or copy and paste this link into your browser:</p>
				<p><a href="%s">%s</a></p>
				<p><strong>This link will expire in 24 hours.</strong></p>
				<div class="footer">
					<p>If you didn't create an account, you can safely ignore this email.</p>
					<p>This is an automated email, please do not reply.</p>
				</div>
			</div>
		</body>
		</html>
	`, fullName, verifyURL, verifyURL, verifyURL)

	return subject, body
}

// SendEmailVerification sends email verification asynchronously
func (h *EmailHelper) SendEmailVerification(ctx context.Context, email, fullName, token string) {
	go func() {
		subject, body := h.EmailVerificationEmail(fullName, token)
		if err := h.sender.Send(email, subject, body); err != nil {
			h.logger.Error("failed to send email verification",
				zap.String("email", email),
				zap.Error(err),
			)
		} else {
			h.logger.Info("email verification sent",
				zap.String("email", email),
			)
		}
	}()
}

// ========== Welcome Email (After Registration) ==========

// WelcomeEmail builds a welcome email for new users
func (h *EmailHelper) WelcomeEmail(fullName, email string) (string, string) {
	loginURL := fmt.Sprintf("%s/auth/login", h.baseURL)

	subject := "Welcome to TaskaApp!"
	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.button { 
					display: inline-block; 
					padding: 12px 24px; 
					background-color: #4CAF50; 
					color: white; 
					text-decoration: none; 
					border-radius: 4px; 
					margin: 20px 0;
				}
				.features { background-color: #f5f5f5; padding: 20px; border-radius: 4px; margin: 20px 0; }
				.footer { margin-top: 30px; font-size: 12px; color: #666; }
			</style>
		</head>
		<body>
			<div class="container">
				<h2>Welcome to TaskaApp! 🚀</h2>
				<p>Hello %s,</p>
				<p>Your account has been successfully created and verified!</p>
				<div class="features">
					<h3>What's Next?</h3>
					<ul>
						<li>✅ Complete your profile</li>
						<li>✅ Explore our features</li>
						<li>✅ Connect with others</li>
					</ul>
				</div>
				<p>Get started by logging in:</p>
				<a href="%s" class="button">Login Now</a>
				<p><strong>Your login email:</strong> %s</p>
				<div class="footer">
					<p>Need help? Contact our support team at support@taskaapp.com</p>
					<p>This is an automated email, please do not reply.</p>
				</div>
			</div>
		</body>
		</html>
	`, fullName, loginURL, email)

	return subject, body
}

// SendWelcomeEmail sends welcome email asynchronously
func (h *EmailHelper) SendWelcomeEmail(ctx context.Context, email, fullName string) {
	go func() {
		subject, body := h.WelcomeEmail(fullName, email)
		if err := h.sender.Send(email, subject, body); err != nil {
			h.logger.Error("failed to send welcome email",
				zap.String("email", email),
				zap.Error(err),
			)
		} else {
			h.logger.Info("welcome email sent",
				zap.String("email", email),
			)
		}
	}()
}

// ========== Account Created by Admin ==========

// AccountCreatedByAdminEmail builds email for accounts created by admin
func (h *EmailHelper) AccountCreatedByAdminEmail(fullName, email, temporaryPassword string, roles []string) (string, string) {
	loginURL := fmt.Sprintf("%s/auth/login", h.baseURL)
	rolesStr := strings.Join(roles, ", ")

	subject := "Your TaskaApp Account Has Been Created"
	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.credentials { 
					background-color: #fff3cd; 
					padding: 15px; 
					border-radius: 4px; 
					margin: 20px 0;
					border-left: 4px solid #ffc107;
				}
				.button { 
					display: inline-block; 
					padding: 12px 24px; 
					background-color: #4CAF50; 
					color: white; 
					text-decoration: none; 
					border-radius: 4px; 
					margin: 20px 0;
				}
				.warning { color: #721c24; background-color: #f8d7da; padding: 12px; border-radius: 4px; }
				.footer { margin-top: 30px; font-size: 12px; color: #666; }
			</style>
		</head>
		<body>
			<div class="container">
				<h2>Your Account Has Been Created</h2>
				<p>Hello %s,</p>
				<p>An administrator has created an account for you on TaskaApp.</p>
				<div class="credentials">
					<h3>Your Login Credentials:</h3>
					<p><strong>Email:</strong> %s</p>
					<p><strong>Temporary Password:</strong> <code>%s</code></p>
					<p><strong>Your Role(s):</strong> %s</p>
				</div>
				<div class="warning">
					<strong>⚠️ Important Security Steps:</strong>
					<ol>
						<li>Login using the temporary password above</li>
						<li><strong>Change your password immediately</strong> after first login</li>
						<li>Never share your password with anyone</li>
					</ol>
				</div>
				<a href="%s" class="button">Login Now</a>
				<div class="footer">
					<p>If you believe you received this email by mistake, please contact your administrator.</p>
					<p>This is an automated email, please do not reply.</p>
				</div>
			</div>
		</body>
		</html>
	`, fullName, email, temporaryPassword, rolesStr, loginURL)

	return subject, body
}

// SendAccountCreatedByAdmin sends account creation email asynchronously
func (h *EmailHelper) SendAccountCreatedByAdmin(ctx context.Context, email, fullName, temporaryPassword string, roles []string) {
	go func() {
		subject, body := h.AccountCreatedByAdminEmail(fullName, email, temporaryPassword, roles)
		if err := h.sender.Send(email, subject, body); err != nil {
			h.logger.Error("failed to send account created email",
				zap.String("email", email),
				zap.Error(err),
			)
		} else {
			h.logger.Info("account created email sent",
				zap.String("email", email),
			)
		}
	}()
}

// ========== Password Changed Notification ==========

// PasswordChangedEmail notifies user of password change
func (h *EmailHelper) PasswordChangedEmail(fullName string) (string, string) {
	subject := "Password Changed - TaskaApp"
	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.alert { 
					background-color: #d4edda; 
					padding: 15px; 
					border-radius: 4px; 
					border-left: 4px solid #28a745;
					margin: 20px 0;
				}
				.warning { 
					background-color: #f8d7da; 
					padding: 15px; 
					border-radius: 4px; 
					border-left: 4px solid #dc3545;
					margin: 20px 0;
				}
				.footer { margin-top: 30px; font-size: 12px; color: #666; }
			</style>
		</head>
		<body>
			<div class="container">
				<h2>Password Changed Successfully</h2>
				<p>Hello %s,</p>
				<div class="alert">
					<p><strong>✅ Your password has been changed successfully.</strong></p>
					<p>All your existing sessions have been logged out for security.</p>
				</div>
				<div class="warning">
					<p><strong>⚠️ Did you make this change?</strong></p>
					<p>If you did not change your password, please contact support immediately at support@taskaapp.com</p>
				</div>
				<div class="footer">
					<p>This is a security notification email.</p>
					<p>This is an automated email, please do not reply.</p>
				</div>
			</div>
		</body>
		</html>
	`, fullName)

	return subject, body
}

// SendPasswordChangedNotification sends password changed notification
func (h *EmailHelper) SendPasswordChangedNotification(ctx context.Context, email, fullName string) {
	go func() {
		subject, body := h.PasswordChangedEmail(fullName)
		if err := h.sender.Send(email, subject, body); err != nil {
			h.logger.Error("failed to send password changed notification",
				zap.String("email", email),
				zap.Error(err),
			)
		} else {
			h.logger.Info("password changed notification sent",
				zap.String("email", email),
			)
		}
	}()
}

// ========== OTP Email ==========

// OTPEmail builds an OTP verification email
func (h *EmailHelper) OTPEmail(otp string) (string, string) {
	subject := "Your Verification Code - TaskaApp"
	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.otp-box { 
					background-color: #f8f9fa; 
					padding: 30px; 
					text-align: center; 
					border-radius: 8px; 
					margin: 30px 0;
					border: 2px dashed #dee2e6;
				}
				.otp-code {
					font-size: 36px;
					font-weight: bold;
					letter-spacing: 8px;
					color: #2196F3;
					font-family: 'Courier New', monospace;
				}
				.warning { 
					background-color: #fff3cd; 
					padding: 15px; 
					border-radius: 4px; 
					border-left: 4px solid #ffc107;
					margin: 20px 0;
				}
				.footer { margin-top: 30px; font-size: 12px; color: #666; }
			</style>
		</head>
		<body>
			<div class="container">
				<h2>Email Verification</h2>
				<p>Thank you for signing up with TaskaApp!</p>
				<p>Please use the following code to verify your email address:</p>
				<div class="otp-box">
					<div class="otp-code">%s</div>
					<p style="margin-top: 15px; color: #666;">Enter this code to continue</p>
				</div>
				<div class="warning">
					<strong>⚠️ Security Notice:</strong>
					<ul>
						<li>This code will expire in 10 minutes</li>
						<li>Never share this code with anyone</li>
						<li>If you didn't request this code, please ignore this email</li>
					</ul>
				</div>
				<div class="footer">
					<p>This is an automated email, please do not reply.</p>
					<p>© 2024 TaskaApp. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, otp)

	return subject, body
}

// SendOTPEmail sends OTP email asynchronously
func (h *EmailHelper) SendOTPEmail(ctx context.Context, email, otp string) error {
	subject, body := h.OTPEmail(otp)
	if err := h.sender.Send(email, subject, body); err != nil {
		h.logger.Error("failed to send OTP email",
			zap.String("email", email),
			zap.Error(err),
		)
		return err
	}
	
	h.logger.Info("OTP email sent",
		zap.String("email", email),
	)
	return nil
}

// ========== Helper Functions ==========

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}