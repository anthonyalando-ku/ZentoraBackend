// internal/pkg/session/rate_limiter.go
package session

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// CheckLoginAttempt checks if login attempt is allowed
func (r *RateLimiter) CheckLoginAttempt(ctx context.Context, ip, email string) (bool, int64, error) {
	key := fmt.Sprintf("ratelimit:login:%s:%s", ip, email)
	
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return false, 0, fmt.Errorf("failed to increment login attempt: %w", err)
	}

	// Set expiration on first attempt
	if count == 1 {
		r.client.Expire(ctx, key, 15*time.Minute)
	}

	maxAttempts := int64(5)
	remaining := maxAttempts - count
	if remaining < 0 {
		remaining = 0
	}

	// Allow up to 5 attempts per 15 minutes
	return count <= maxAttempts, remaining, nil
}

// GetRemainingAttempts returns remaining login attempts
func (r *RateLimiter) GetRemainingAttempts(ctx context.Context, ip, email string) (int64, error) {
	key := fmt.Sprintf("ratelimit:login:%s:%s", ip, email)
	
	count, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 5, nil // Full attempts available
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get login attempts: %w", err)
	}

	remaining := 5 - count
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// ResetLoginAttempts resets the login attempt counter
func (r *RateLimiter) ResetLoginAttempts(ctx context.Context, ip, email string) error {
	key := fmt.Sprintf("ratelimit:login:%s:%s", ip, email)
	return r.client.Del(ctx, key).Err()
}

// CheckPasswordResetAttempt checks password reset rate limit
func (r *RateLimiter) CheckPasswordResetAttempt(ctx context.Context, email string) (bool, error) {
	key := fmt.Sprintf("ratelimit:password_reset:%s", email)
	
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to increment password reset attempt: %w", err)
	}

	// Set expiration on first attempt
	if count == 1 {
		r.client.Expire(ctx, key, 1*time.Hour)
	}

	// Allow up to 3 password resets per hour
	return count <= 3, nil
}

// CheckOTPAttempt checks OTP verification rate limit
func (r *RateLimiter) CheckOTPAttempt(ctx context.Context, identityID int64) (bool, error) {
	key := fmt.Sprintf("ratelimit:otp:%d", identityID)
	
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to increment OTP attempt: %w", err)
	}

	// Set expiration on first attempt
	if count == 1 {
		r.client.Expire(ctx, key, 10*time.Minute)
	}

	// Allow up to 5 OTP attempts per 10 minutes
	return count <= 5, nil
}

// ResetOTPAttempts resets OTP attempts
func (r *RateLimiter) ResetOTPAttempts(ctx context.Context, identityID int64) error {
	key := fmt.Sprintf("ratelimit:otp:%d", identityID)
	return r.client.Del(ctx, key).Err()
}

// CheckAPIRateLimit checks general API rate limiting
func (r *RateLimiter) CheckAPIRateLimit(ctx context.Context, identityID int64, endpoint string, maxRequests int64, window time.Duration) (bool, error) {
	key := fmt.Sprintf("ratelimit:api:%d:%s", identityID, endpoint)
	
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to increment API rate limit: %w", err)
	}

	// Set expiration on first attempt
	if count == 1 {
		r.client.Expire(ctx, key, window)
	}

	return count <= maxRequests, nil
}

// IsAccountTemporarilyLocked checks if account is temporarily locked
func (r *RateLimiter) IsAccountTemporarilyLocked(ctx context.Context, identityID int64) (bool, time.Duration, error) {
	key := fmt.Sprintf("account:locked:%d", identityID)
	
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}

	if ttl < 0 {
		return false, 0, nil // Not locked
	}

	return true, ttl, nil
}

// LockAccount temporarily locks an account
func (r *RateLimiter) LockAccount(ctx context.Context, identityID int64, duration time.Duration) error {
	key := fmt.Sprintf("account:locked:%d", identityID)
	return r.client.Set(ctx, key, "1", duration).Err()
}

// UnlockAccount unlocks a temporarily locked account
func (r *RateLimiter) UnlockAccount(ctx context.Context, identityID int64) error {
	key := fmt.Sprintf("account:locked:%d", identityID)
	return r.client.Del(ctx, key).Err()
}


// Auth flow 2: Email verification with OTP

// SendEmailVerificationOTP sends OTP to email for verification

// CheckEmailVerificationAttempt checks email verification OTP request rate limit
func (r *RateLimiter) CheckEmailVerificationAttempt(ctx context.Context, email string) (bool, error) {
	// Check short-term limit (1 per minute)
	shortKey := fmt.Sprintf("ratelimit:email_verify_short:%s", email)
	shortCount, err := r.client.Incr(ctx, shortKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to increment short-term verification attempt: %w", err)
	}

	// Set expiration on first attempt
	if shortCount == 1 {
		r.client.Expire(ctx, shortKey, 1*time.Minute)
	}

	// Only allow 1 request per minute
	if shortCount > 1 {
		return false, fmt.Errorf("please wait 1 minute before requesting another code")
	}

	// Check long-term limit (4 per 30 minutes)
	longKey := fmt.Sprintf("ratelimit:email_verify_long:%s", email)
	longCount, err := r.client.Incr(ctx, longKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to increment long-term verification attempt: %w", err)
	}

	// Set expiration on first attempt
	if longCount == 1 {
		r.client.Expire(ctx, longKey, 30*time.Minute)
	}

	// Block if more than 4 attempts in 30 minutes
	if longCount > 4 {
		// Lock the email from verification attempts
		if err := r.LockEmailVerification(ctx, email, 30*time.Minute); err != nil {
			return false, err
		}
		return false, fmt.Errorf("too many verification attempts. Please try again in 30 minutes")
	}

	return true, nil
}

// LockEmailVerification temporarily locks email verification for an email
func (r *RateLimiter) LockEmailVerification(ctx context.Context, email string, duration time.Duration) error {
	key := fmt.Sprintf("email_verify:locked:%s", email)
	return r.client.Set(ctx, key, "1", duration).Err()
}

// IsEmailVerificationLocked checks if email verification is locked
func (r *RateLimiter) IsEmailVerificationLocked(ctx context.Context, email string) (bool, time.Duration, error) {
	key := fmt.Sprintf("email_verify:locked:%s", email)
	
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}

	if ttl < 0 {
		return false, 0, nil // Not locked
	}

	return true, ttl, nil
}

// GetEmailVerificationCooldown returns remaining cooldown time
func (r *RateLimiter) GetEmailVerificationCooldown(ctx context.Context, email string) (time.Duration, error) {
	shortKey := fmt.Sprintf("ratelimit:email_verify_short:%s", email)
	
	ttl, err := r.client.TTL(ctx, shortKey).Result()
	if err != nil {
		return 0, err
	}

	if ttl < 0 {
		return 0, nil // No cooldown
	}

	return ttl, nil
}

// GetRemainingEmailVerificationAttempts returns remaining attempts
func (r *RateLimiter) GetRemainingEmailVerificationAttempts(ctx context.Context, email string) (int64, error) {
	longKey := fmt.Sprintf("ratelimit:email_verify_long:%s", email)
	
	count, err := r.client.Get(ctx, longKey).Int64()
	if err == redis.Nil {
		return 4, nil // Full attempts available
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get verification attempts: %w", err)
	}

	remaining := 4 - count
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// checkLimit is a helper function for rate limiting
func (r *RateLimiter) checkLimit(ctx context.Context, key string, maxAttempts int64, window time.Duration) (bool, error) {
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to increment counter: %w", err)
	}

	// Set expiration on first attempt
	if count == 1 {
		r.client.Expire(ctx, key, window)
	}

	return count <= maxAttempts, nil
}