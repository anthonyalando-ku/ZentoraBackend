// internal/domain/auth/entity.go
package auth

import (
	"database/sql"
	"time"
)

// Identity represents the core user identity
type Identity struct {
	ID                   int64          `json:"id" db:"id"`
	Email                sql.NullString `json:"email" db:"email"`
	EmailVerified        bool           `json:"email_verified" db:"email_verified"`
	EmailVerifiedAt      sql.NullTime   `json:"email_verified_at" db:"email_verified_at"`
	Phone                sql.NullString `json:"phone" db:"phone"`
	PhoneVerified        bool           `json:"phone_verified" db:"phone_verified"`
	PhoneVerifiedAt      sql.NullTime   `json:"phone_verified_at" db:"phone_verified_at"`
	Status               string         `json:"status" db:"status"` // active, inactive, suspended, pending_verification
	LastLogin            sql.NullTime   `json:"last_login" db:"last_login"`
	FailedLoginAttempts  int            `json:"-" db:"failed_login_attempts"`
	LockedUntil          sql.NullTime   `json:"-" db:"locked_until"`
	CreatedAt            time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at" db:"updated_at"`
	DeletedAt            sql.NullTime   `json:"-" db:"deleted_at"`
}

// Provider represents an authentication provider (local, google, etc.)
type Provider struct {
	ID                int64                  `json:"id" db:"id"`
	IdentityID        int64                  `json:"identity_id" db:"identity_id"`
	Provider          string                 `json:"provider" db:"provider"` // local, google, facebook, etc.
	ProviderUserID    sql.NullString         `json:"provider_user_id" db:"provider_user_id"`
	ProviderEmail     sql.NullString         `json:"provider_email" db:"provider_email"`
	ProviderUsername  sql.NullString         `json:"provider_username" db:"provider_username"`
	PasswordHash      sql.NullString         `json:"-" db:"password_hash"` // Only for local provider
	AccessToken       sql.NullString         `json:"-" db:"access_token"`
	RefreshToken      sql.NullString         `json:"-" db:"refresh_token"`
	TokenExpiresAt    sql.NullTime           `json:"-" db:"token_expires_at"`
	ProviderData      map[string]interface{} `json:"provider_data" db:"provider_data"`
	IsPrimary         bool                   `json:"is_primary" db:"is_primary"`
	PasswordChangedAt sql.NullTime           `json:"-" db:"password_changed_at"`
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" db:"updated_at"`
}

// Session represents a user session
type Session struct {
	ID                int64                  `json:"id" db:"id"`
	IdentityID        int64                  `json:"identity_id" db:"identity_id"`
	SessionToken      string                 `json:"-" db:"session_token"`
	RefreshToken      sql.NullString         `json:"-" db:"refresh_token"`
	Provider          string                 `json:"provider" db:"provider"`
	IPAddress         sql.NullString         `json:"ip_address" db:"ip_address"`
	UserAgent         sql.NullString         `json:"user_agent" db:"user_agent"`
	DeviceID          sql.NullString         `json:"device_id" db:"device_id"`
	DeviceName        sql.NullString         `json:"device_name" db:"device_name"`
	DeviceFingerprint sql.NullString         `json:"device_fingerprint" db:"device_fingerprint"`
	Status            string                 `json:"status" db:"status"` // active, expired, revoked
	LoginAt           time.Time              `json:"login_at" db:"login_at"`
	LastActivityAt    time.Time              `json:"last_activity_at" db:"last_activity_at"`
	ExpiresAt         time.Time              `json:"expires_at" db:"expires_at"`
	LogoutAt          sql.NullTime           `json:"logout_at" db:"logout_at"`
	Metadata          map[string]interface{} `json:"metadata" db:"metadata"`
}

// UserProfile represents user profile data
type UserProfile struct {
	ID         int64                  `json:"id" db:"id"`
	IdentityID int64                  `json:"identity_id" db:"identity_id"`
	FullName   sql.NullString         `json:"full_name" db:"full_name"`
	AvatarURL  sql.NullString         `json:"avatar_url" db:"avatar_url"`
	Bio        sql.NullString         `json:"bio" db:"bio"`
	Metadata   map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at" db:"updated_at"`
}

// VerificationToken represents a verification/reset token
type VerificationToken struct {
	ID         int64                  `json:"id" db:"id"`
	IdentityID int64                  `json:"identity_id" db:"identity_id"`
	TokenType  string                 `json:"token_type" db:"token_type"` // password_reset, email_verify, phone_verify
	Token      string                 `json:"token" db:"token"`
	Code       sql.NullString         `json:"code" db:"code"` // For OTP
	ExpiresAt  time.Time              `json:"expires_at" db:"expires_at"`
	UsedAt     sql.NullTime           `json:"used_at" db:"used_at"`
	VerifiedAt sql.NullTime           `json:"verified_at" db:"verified_at"`
	Attempts   int                    `json:"attempts" db:"attempts"`
	Metadata   map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}

type Role struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	DisplayName string    `json:"display_name" db:"display_name"`
	Description string    `json:"description" db:"description"`
	IsSystem    bool      `json:"is_system" db:"is_system"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}