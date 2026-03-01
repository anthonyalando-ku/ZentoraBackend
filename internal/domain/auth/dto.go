// internal/domain/auth/dto.go
package auth

import "time"

// RegisterRequest for user registration
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Phone     string `json:"phone"`
	Password  string `json:"password" binding:"required,min=8"`
	Token    string `json:"token"` // Optional token for email verification
	FullName  string `json:"full_name"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Device    string `json:"device"`
	IPAddress string `json:"-"`
	UserAgent string `json:"-"`
}
// EmailVerificationRequest for initial email verification
type EmailVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// EmailVerificationResponse after sending OTP
type EmailVerificationResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Message   string    `json:"message"`
}

// OTPVerificationRequest for verifying OTP code
type OTPVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
	Token string `json:"token" binding:"required"`
	OTP   string `json:"otp" binding:"required,len=6"`
}

// ResendOTPRequest for resending OTP
type ResendOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	Token string `json:"token" binding:"required"`
}

// OTPVerificationResponse after successful OTP verification
type OTPVerificationResponse struct {
	VerificationToken string    `json:"token"`
	ExpiresAt         time.Time `json:"expires_at"`
	Message           string    `json:"message"`
}
// LoginRequest for user login
type LoginRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required"`
	Device    string `json:"device"`
	IPAddress string `json:"-"`
	UserAgent string `json:"-"`
}

// LoginResponse successful login response
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         UserInfo  `json:"user"`
}

// CreateAdminRequest for creating admin accounts
type CreateAdminRequest struct {
	Email    string   `json:"email" binding:"required,email"`
	Phone    string   `json:"phone"`
	FullName string   `json:"full_name" binding:"required"`
	Roles    []string `json:"roles" binding:"required"` // e.g., ["admin"] or ["super_admin"]
}

// UserInfo minimal user information
type UserInfo struct {
	IdentityID  int64    `json:"identity_id"`
	Email       string   `json:"email"`
	FullName    string   `json:"full_name"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

// ChangePasswordRequest for password change
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// ForgotPasswordRequest for password reset
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest for completing password reset
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// UpdateProfileRequest for profile updates
type UpdateProfileRequest struct {
	FullName  string                 `json:"full_name"`
	AvatarURL string                 `json:"avatar_url"`
	Bio       string                 `json:"bio"`
	Metadata  map[string]interface{} `json:"metadata"`
}