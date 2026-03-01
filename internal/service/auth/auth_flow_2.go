// internal/usecase/auth/auth_service.go
package auth

import (
	"context"
	"crypto/rand"
	"encoding/json"

	//"errors"
	"fmt"
	"time"

	"diary-service/internal/domain/auth"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// SendEmailVerificationOTP sends OTP to email for verification
func (s *AuthService) SendEmailVerificationOTP(ctx context.Context, email string) (*auth.EmailVerificationResponse, error) {
	// Check if locked
	locked, ttl, err := s.rateLimiter.IsEmailVerificationLocked(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}
	if locked {
		minutes := int(ttl.Minutes())
		return nil, fmt.Errorf("too many verification attempts. Please try again in %d minutes", minutes)
	}

	// Check rate limit
	allowed, err := s.rateLimiter.CheckEmailVerificationAttempt(ctx, email)
	if err != nil {
		return nil, err
	}
	if !allowed {
		// Get cooldown time
		cooldown, _ := s.rateLimiter.GetEmailVerificationCooldown(ctx, email)
		seconds := int(cooldown.Seconds())
		return nil, fmt.Errorf("please wait %d seconds before requesting another code", seconds)
	}

	// Check if email already exists
	exists, err := s.authRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("email already registered")
	}

	// Generate 6-digit OTP
	otp := generateOTP(6)

	// Generate verification token
	token := generateToken()

	// Store OTP in cache with 10 minute expiry
	cacheKey := fmt.Sprintf("email_otp:%s:%s", email, token)
	if err := s.cache.Set(ctx, cacheKey, otp, 10*time.Minute).Err(); err != nil {
		return nil, fmt.Errorf("failed to store OTP: %w", err)
	}

	// Store metadata for resend
	metadataKey := fmt.Sprintf("email_otp_meta:%s:%s", email, token)
	metadata := map[string]interface{}{
		"email":      email,
		"created_at": time.Now().Unix(),
		"resend_count": 0,
	}
	metadataJSON, _ := json.Marshal(metadata)
	s.cache.Set(ctx, metadataKey, metadataJSON, 10*time.Minute)

	// Send OTP via email
	if err := s.emailHelper.SendOTPEmail(ctx, email, otp); err != nil {
		s.logger.Error("failed to send OTP email", zap.Error(err))
		return nil, fmt.Errorf("failed to send OTP email: %w", err)
	}

	s.logger.Info("OTP sent to email",
		zap.String("email", email),
		zap.String("token", token),
	)

	// Get remaining attempts
	remaining, _ := s.rateLimiter.GetRemainingEmailVerificationAttempts(ctx, email)

	return &auth.EmailVerificationResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Message:   fmt.Sprintf("OTP sent to your email. Valid for 10 minutes. %d attempts remaining.", remaining),
	}, nil
}

// ResendEmailVerificationOTP resends OTP using existing token
func (s *AuthService) ResendEmailVerificationOTP(ctx context.Context, email, token string) (*auth.EmailVerificationResponse, error) {
	// Check if locked
	locked, ttl, err := s.rateLimiter.IsEmailVerificationLocked(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}
	if locked {
		minutes := int(ttl.Minutes())
		return nil, fmt.Errorf("too many verification attempts. Please try again in %d minutes", minutes)
	}

	// Check rate limit (same limits apply)
	allowed, err := s.rateLimiter.CheckEmailVerificationAttempt(ctx, email)
	if err != nil {
		return nil, err
	}
	if !allowed {
		cooldown, _ := s.rateLimiter.GetEmailVerificationCooldown(ctx, email)
		seconds := int(cooldown.Seconds())
		return nil, fmt.Errorf("please wait %d seconds before requesting another code", seconds)
	}

	// Check if original OTP exists
	cacheKey := fmt.Sprintf("email_otp:%s:%s", email, token)
	exists, err := s.cache.Exists(ctx, cacheKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to check OTP: %w", err)
	}
	if exists == 0 {
		return nil, fmt.Errorf("OTP session expired. Please start verification again")
	}

	// Get metadata
	metadataKey := fmt.Sprintf("email_otp_meta:%s:%s", email, token)
	metadataJSON, err := s.cache.Get(ctx, metadataKey).Result()
	if err != nil {
		return nil, fmt.Errorf("OTP session expired. Please start verification again")
	}

	var metadata map[string]interface{}
	json.Unmarshal([]byte(metadataJSON), &metadata)

	// Check resend count (max 3 resends)
	resendCount, _ := metadata["resend_count"].(float64)
	if resendCount >= 3 {
		return nil, fmt.Errorf("maximum resend attempts reached. Please start verification again")
	}

	// Generate new OTP
	otp := generateOTP(6)

	// Update OTP in cache (extend expiry to 10 minutes from now)
	if err := s.cache.Set(ctx, cacheKey, otp, 10*time.Minute).Err(); err != nil {
		return nil, fmt.Errorf("failed to update OTP: %w", err)
	}

	// Update metadata
	metadata["resend_count"] = resendCount + 1
	metadata["last_resend"] = time.Now().Unix()
	updatedMetadataJSON, _ := json.Marshal(metadata)
	s.cache.Set(ctx, metadataKey, updatedMetadataJSON, 10*time.Minute)

	// Send new OTP via email
	if err := s.emailHelper.SendOTPEmail(ctx, email, otp); err != nil {
		s.logger.Error("failed to resend OTP email", zap.Error(err))
		return nil, fmt.Errorf("failed to send OTP email: %w", err)
	}

	s.logger.Info("OTP resent to email",
		zap.String("email", email),
		zap.String("token", token),
		zap.Float64("resend_count", resendCount+1),
	)

	// Get remaining attempts
	remaining, _ := s.rateLimiter.GetRemainingEmailVerificationAttempts(ctx, email)

	return &auth.EmailVerificationResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Message:   fmt.Sprintf("New OTP sent to your email. Valid for 10 minutes. %d attempts remaining.", remaining),
	}, nil
}

// VerifyEmailOTP verifies the OTP code
func (s *AuthService) VerifyEmailOTP(ctx context.Context, req *auth.OTPVerificationRequest) (*auth.OTPVerificationResponse, error) {
	// Get OTP from cache
	cacheKey := fmt.Sprintf("email_otp:%s:%s", req.Email, req.Token)
	cachedOTP, err := s.cache.Get(ctx, cacheKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("OTP expired or invalid")
		}
		return nil, fmt.Errorf("failed to verify OTP: %w", err)
	}

	// Verify OTP
	if cachedOTP != req.OTP {
		return nil, fmt.Errorf("invalid OTP code")
	}

	// Delete used OTP
	s.cache.Del(ctx, cacheKey)

	// Generate verification token for registration
	verificationToken := generateToken()

	// Store verification token with 1 hour expiry
	verificationKey := fmt.Sprintf("email_verified:%s", verificationToken)
	if err := s.cache.Set(ctx, verificationKey, req.Email, 1*time.Hour).Err(); err != nil {
		return nil, fmt.Errorf("failed to store verification: %w", err)
	}

	s.logger.Info("email verified with OTP",
		zap.String("email", req.Email),
		zap.String("verification_token", verificationToken),
	)

	return &auth.OTPVerificationResponse{
		VerificationToken: verificationToken,
		ExpiresAt:         time.Now().Add(1 * time.Hour),
		Message:           "Email verified successfully. Please complete registration.",
	}, nil
}

// ValidateVerificationToken checks if verification token is valid
func (s *AuthService) ValidateVerificationToken(ctx context.Context, token, email string) (bool, error) {
	if token == "" {
		return false, nil
	}

	verificationKey := fmt.Sprintf("email_verified:%s", token)
	verifiedEmail, err := s.cache.Get(ctx, verificationKey).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, fmt.Errorf("failed to validate token: %w", err)
	}

	// Check if email matches
	if verifiedEmail != email {
		return false, nil
	}

	return true, nil
}

// generateOTP generates a numeric OTP code
func generateOTP(length int) string {
	const charset = "0123456789"
	b := make([]byte, length)
	rand.Read(b)

	otp := make([]byte, length)
	for i := range otp {
		otp[i] = charset[int(b[i])%len(charset)]
	}

	return string(otp)
}