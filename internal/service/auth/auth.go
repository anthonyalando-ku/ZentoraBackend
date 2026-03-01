// internal/usecase/auth/auth_service.go
package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	//"errors"
	"fmt"
	"time"

	"diary-service/internal/domain/auth"
	"diary-service/internal/domain/websocket"
	xerrors "diary-service/internal/pkg/errors"
	"diary-service/internal/pkg/jwt"
	"diary-service/internal/pkg/session"
	"diary-service/internal/repository/postgres"
	"diary-service/internal/service/email"
	ws "diary-service/internal/websocket"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	authRepo       *postgres.AuthRepository
	jwtManager     *jwt.Manager
	sessionManager *session.Manager
	rateLimiter    *session.RateLimiter
	emailSender    *email.EmailSender
	emailHelper *EmailHelper
	hub            *ws.Hub
	cache          *redis.Client
	logger         *zap.Logger
}

func NewAuthService(
	authRepo *postgres.AuthRepository,
	jwtManager *jwt.Manager,
	sessionManager *session.Manager,
	rateLimiter *session.RateLimiter,
	emailSender *email.EmailSender,
	hub *ws.Hub,
	cache *redis.Client,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		authRepo:       authRepo,
		jwtManager:     jwtManager,
		sessionManager: sessionManager,
		rateLimiter:    rateLimiter,
		emailSender:    emailSender,
		emailHelper:    NewEmailHelper(emailSender, logger, "https://your-base-url.com"),
		hub:            hub,
		cache:          cache,
		logger:         logger,
	}
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, req *auth.RegisterRequest) (*auth.LoginResponse, error) {
	// If token is provided, validate it
	if req.FullName == "" {
		req.FullName = fmt.Sprintf("%s %s", req.FirstName, req.LastName)
	}
	if req.Token != "" {
		valid, err := s.ValidateVerificationToken(ctx, req.Token, req.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to validate verification token: %w", err)
		}
		if !valid {
			return nil, fmt.Errorf("invalid or expired verification token")
		}

		// Delete used verification token
		verificationKey := fmt.Sprintf("email_verified:%s", req.Token)
		s.cache.Del(ctx, verificationKey)

		s.logger.Info("registering with pre-verified email",
			zap.String("email", req.Email),
		)
	} else {
		// Original flow - check if email exists
		exists, err := s.authRepo.ExistsByEmail(ctx, req.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to check email: %w", err)
		}
		if exists {
			return nil, xerrors.ErrDuplicateEntry
		}
	}

	// Check if phone already exists (if provided)
	if req.Phone != "" {
		exists, err := s.authRepo.ExistsByPhone(ctx, req.Phone)
		if err != nil {
			return nil, fmt.Errorf("failed to check phone: %w", err)
		}
		if exists {
			return nil, xerrors.ErrDuplicateEntry
		}
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create identity
	identity := &auth.Identity{
		Email:  sql.NullString{String: req.Email, Valid: true},
		Phone:  sql.NullString{String: req.Phone, Valid: req.Phone != ""},
		Status: "pending_verification",
	}

	// If registered with verification token, mark email as verified
	if req.Token != "" {
		identity.EmailVerified = true
		identity.EmailVerifiedAt = sql.NullTime{Time: time.Now(), Valid: true}
		identity.Status = "active"
	}

	if err := s.authRepo.CreateIdentity(ctx, identity); err != nil {
		return nil, fmt.Errorf("failed to create identity: %w", err)
	}

	// Create local auth provider
	provider := &auth.Provider{
		IdentityID:   identity.ID,
		Provider:     "local",
		PasswordHash: sql.NullString{String: string(hashedPassword), Valid: true},
		IsPrimary:    true,
	}

	if err := s.authRepo.CreateProvider(ctx, provider); err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Create user profile
	profile := &auth.UserProfile{
		IdentityID: identity.ID,
		FullName:   sql.NullString{String: req.FullName, Valid: req.FullName != ""},
	}

	if err := s.authRepo.CreateUserProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	// Send email verification only if not pre-verified
	if req.Token == "" {
		if err := s.SendEmailVerification(ctx, identity.ID, req.Email); err != nil {
			s.logger.Error("failed to send verification email", zap.Error(err))
			// Don't fail registration if email sending fails
		}
	} else {
		// Send welcome email for pre-verified accounts
		s.emailHelper.SendWelcomeEmail(ctx, req.Email, req.FullName);
		
	}

	// Auto-login after registration
	return s.loginWithIdentity(ctx, identity, provider, req.Device, req.IPAddress, req.UserAgent)
}
// Update methods to use emailHelper
func (s *AuthService) SendEmailVerification(ctx context.Context, identityID int64, email string) error {
	// Generate verification token
	token := generateToken()
	
	vToken := &auth.VerificationToken{
		IdentityID: identityID,
		TokenType:  "email_verify",
		Token:      token,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	
	if err := s.authRepo.CreateVerificationToken(ctx, vToken); err != nil {
		return err
	}
	
	profile, _ := s.authRepo.GetUserProfile(ctx, identityID)
	fullName := "User"
	if profile != nil && profile.FullName.Valid {
		fullName = profile.FullName.String
	}
	
	s.emailHelper.SendEmailVerification(ctx, email, fullName, token)
	return nil
}

func generateToken() string {
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	return base64.URLEncoding.EncodeToString(tokenBytes)
}
// ========== Login ==========

// Login authenticates a user with email/password
func (s *AuthService) Login(ctx context.Context, req *auth.LoginRequest) (*auth.LoginResponse, error) {
	// Rate limiting
	allowed, remaining, err := s.rateLimiter.CheckLoginAttempt(ctx, req.IPAddress, req.Email)
	if err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("too many login attempts, please try again in 15 minutes")
	}

	// Find identity by email
	identity, err := s.authRepo.FindIdentityByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if account is active
	if identity.Status == "inactive" {
		return nil, fmt.Errorf("account is inactive")
	}
	if identity.Status == "suspended" {
		return nil, fmt.Errorf("account is suspended")
	}

	// Check if account is locked
	if identity.LockedUntil.Valid && identity.LockedUntil.Time.After(time.Now()) {
		return nil, fmt.Errorf("account is temporarily locked until %s", identity.LockedUntil.Time.Format(time.RFC3339))
	}

	// Get local provider
	provider, err := s.authRepo.FindProviderByIdentityAndType(ctx, identity.ID, "local")
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(provider.PasswordHash.String), []byte(req.Password)); err != nil {
		// Increment failed login attempts
		s.authRepo.IncrementFailedLoginAttempts(ctx, identity.ID, 30*time.Minute)
		return nil, fmt.Errorf("invalid credentials (attempts remaining: %d)", remaining-1)
	}

	// Reset failed attempts and update last login
	if err := s.authRepo.UpdateIdentityLastLogin(ctx, identity.ID); err != nil {
		s.logger.Error("failed to update last login", zap.Error(err))
	}
	s.rateLimiter.ResetLoginAttempts(ctx, req.IPAddress, req.Email)

	return s.loginWithIdentity(ctx, identity, provider, req.Device, req.IPAddress, req.UserAgent)
}

// loginWithIdentity is a helper that creates session and generates tokens
func (s *AuthService) loginWithIdentity(ctx context.Context, identity *auth.Identity, provider *auth.Provider, device, ipAddress, userAgent string) (*auth.LoginResponse, error) {
	// Get user roles and permissions
	roles, permissions, err := s.getUserRolesAndPermissions(ctx, identity.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles: %w", err)
	}

	// Generate tokens
	accessToken, accessJTI, err := s.jwtManager.Generator.GenerateAccessToken(
		identity.ID,
		roles,
		permissions,
		device,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshJTI, err := s.jwtManager.Generator.GenerateRefreshToken(identity.ID, device)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	expiresAt := time.Now().Add(s.jwtManager.Generator.Ttl)
	refreshExpiresAt := time.Now().Add(7 * 24 * time.Hour)
	_ = refreshExpiresAt

	// Create session in database
	dbSession := &auth.Session{
		IdentityID:    identity.ID,
		SessionToken:  accessJTI,
		RefreshToken:  sql.NullString{String: refreshJTI, Valid: true},
		Provider:      provider.Provider,
		IPAddress:     sql.NullString{String: ipAddress, Valid: ipAddress != ""},
		UserAgent:     sql.NullString{String: userAgent, Valid: userAgent != ""},
		DeviceID:      sql.NullString{String: device, Valid: device != ""},
		ExpiresAt:     expiresAt,
	}

	if err := s.authRepo.CreateSession(ctx, dbSession); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Create session in Redis
	sessionData := &session.SessionData{
		JTI:            accessJTI,
		IdentityID:     identity.ID,
		SessionID:      dbSession.ID,
		Email:          identity.Email.String,
		Roles:          roles,
		Permissions:    permissions,
		Device:         device,
		DeviceID:       device,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
		Provider:       provider.Provider,
		LoginAt:        time.Now(),
		LastActivityAt: time.Now(),
		ExpiresAt:      expiresAt,
		IsActive:       true,
	}

	if err := s.sessionManager.CreateSession(ctx, sessionData); err != nil {
		return nil, fmt.Errorf("failed to create session cache: %w", err)
	}

	// Get user profile
	profile, _ := s.authRepo.GetUserProfile(ctx, identity.ID)

	return &auth.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.jwtManager.Generator.Ttl.Seconds()),
		ExpiresAt:    expiresAt,
		User: auth.UserInfo{
			IdentityID: identity.ID,
			Email:      identity.Email.String,
			FullName:   profile.FullName.String,
			Roles:      roles,
			Permissions: permissions,
		},
	}, nil
}

// ========== Logout ==========

// Logout invalidates the current session
func (s *AuthService) Logout(ctx context.Context, identityID int64, jti string) error {
	// Invalidate session in Redis and DB
	if err := s.sessionManager.InvalidateSession(ctx, identityID, jti); err != nil {
		return fmt.Errorf("failed to invalidate session: %w", err)
	}

	// Blacklist the token
	remainingTTL := time.Until(time.Now().Add(s.jwtManager.Generator.Ttl))
	if err := s.sessionManager.BlacklistToken(ctx, jti, remainingTTL); err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	// Notify via WebSocket
	s.hub.ForceLogout(identityID, jti, "User logged out")

	return nil
}

// LogoutAllSessions invalidates all sessions for a user
func (s *AuthService) LogoutAllSessions(ctx context.Context, identityID int64) error {
	if err := s.sessionManager.InvalidateAllUserSessions(ctx, identityID); err != nil {
		return fmt.Errorf("failed to invalidate sessions: %w", err)
	}

	// Notify via WebSocket
	s.hub.ForceLogout(identityID, "", "All sessions logged out")

	return nil
}

// ========== Password Management ==========

// ChangePassword changes user password (requires current password)
func (s *AuthService) ChangePassword(ctx context.Context, identityID int64, req *auth.ChangePasswordRequest) error {
	// Get local provider
	provider, err := s.authRepo.FindProviderByIdentityAndType(ctx, identityID, "local")
	if err != nil {
		return fmt.Errorf("local auth not found: %w", err)
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(provider.PasswordHash.String), []byte(req.CurrentPassword)); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.authRepo.UpdateProviderPassword(ctx, provider.ID, string(hashedPassword)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Invalidate all sessions
	if err := s.LogoutAllSessions(ctx, identityID); err != nil {
		return fmt.Errorf("failed to invalidate sessions: %w", err)
	}

	return nil
}

// ForgotPassword initiates password reset process
func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	// Check rate limit
	allowed, err := s.rateLimiter.CheckPasswordResetAttempt(ctx, email)
	if err != nil {
		return fmt.Errorf("rate limiter error: %w", err)
	}
	if !allowed {
		return fmt.Errorf("too many password reset attempts, please try again later")
	}

	// Find identity
	identity, err := s.authRepo.FindIdentityByEmail(ctx, email)
	if err != nil {
		// Don't reveal if email exists
		return nil
	}

	// Generate reset token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}
	resetToken := base64.URLEncoding.EncodeToString(tokenBytes)

	// Store in database
	verificationToken := &auth.VerificationToken{
		IdentityID: identity.ID,
		TokenType:  "password_reset",
		Token:      resetToken,
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}

	if err := s.authRepo.CreateVerificationToken(ctx, verificationToken); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	// Cache for quick lookup
	cacheKey := fmt.Sprintf("password_reset:%d", identity.ID)
	if err := s.cache.Set(ctx, cacheKey, resetToken, 1*time.Hour).Err(); err != nil {
		s.logger.Error("failed to cache reset token", zap.Error(err))
	}

	// Send email
	profile, _ := s.authRepo.GetUserProfile(ctx, identity.ID)
	fullName := "User"
	if profile != nil && profile.FullName.Valid {
		fullName = profile.FullName.String
	}
	s.emailHelper.SendPasswordResetEmail(ctx, identity.Email.String, fullName, resetToken)

	return nil
}

// ResetPassword resets password using token
func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	// Find token
	vToken, err := s.authRepo.FindVerificationToken(ctx, "password_reset", token)
	if err != nil {
		return fmt.Errorf("invalid or expired token")
	}

	// Get local provider
	provider, err := s.authRepo.FindProviderByIdentityAndType(ctx, vToken.IdentityID, "local")
	if err != nil {
		return fmt.Errorf("local auth not found: %w", err)
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.authRepo.UpdateProviderPassword(ctx, provider.ID, string(hashedPassword)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Mark token as used
	if err := s.authRepo.MarkTokenAsUsed(ctx, vToken.ID); err != nil {
		s.logger.Error("failed to mark token as used", zap.Error(err))
	}

	// Clear cache
	cacheKey := fmt.Sprintf("password_reset:%d", vToken.IdentityID)
	s.cache.Del(ctx, cacheKey)

	// Invalidate all sessions
	if err := s.LogoutAllSessions(ctx, vToken.IdentityID); err != nil {
		return fmt.Errorf("failed to invalidate sessions: %w", err)
	}

	return nil
}

// ========== Profile Management ==========

// UpdateProfile updates user profile and broadcasts to WebSocket
func (s *AuthService) UpdateProfile(ctx context.Context, identityID int64, req *auth.UpdateProfileRequest) (*auth.UserProfile, error) {
	profile := &auth.UserProfile{
		FullName:  sql.NullString{String: req.FullName, Valid: req.FullName != ""},
		AvatarURL: sql.NullString{String: req.AvatarURL, Valid: req.AvatarURL != ""},
		Bio:       sql.NullString{String: req.Bio, Valid: req.Bio != ""},
		Metadata:  req.Metadata,
	}

	if err := s.authRepo.UpdateUserProfile(ctx, identityID, profile); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	// Get updated profile
	updated, err := s.authRepo.GetUserProfile(ctx, identityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated profile: %w", err)
	}

	// Broadcast profile update via WebSocket
	s.broadcastProfileUpdate(identityID, updated)

	return updated, nil
}

// broadcastProfileUpdate sends profile update to all user's active sessions
func (s *AuthService) broadcastProfileUpdate(identityID int64, profile *auth.UserProfile) {
	// Create event data
	data := map[string]interface{}{
		"identity_id": identityID,
		"full_name":   profile.FullName.String,
		"avatar_url":  profile.AvatarURL.String,
		"bio":         profile.Bio.String,
		"updated_at":  profile.UpdatedAt,
	}

	// Broadcast via WebSocket
	msg := websocket.NewMessage("profile:updated", data)
	s.hub.BroadcastMessage(&ws.BroadcastMessage{
		IdentityIDs: []int64{identityID},
		Channel:     websocket.ChannelSystem,
		Message:     msg,
	})
}

// ========== Helper Methods ==========

func (s *AuthService) getUserRolesAndPermissions(ctx context.Context, identityID int64) ([]string, []string, error) {
	// Get roles
	roles, err := s.authRepo.GetUserRoles(ctx, identityID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get roles: %w", err)
	}

	// Get permissions
	permissions, err := s.authRepo.GetUserPermissions(ctx, identityID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get permissions: %w", err)
	}

	// Default to user role if no roles assigned
	if len(roles) == 0 {
		roles = []string{"user"}
	}

	return roles, permissions, nil
}

// ValidateToken validates a JWT token and session
func (s *AuthService) ValidateToken(ctx context.Context, token string) (*jwt.Claims, error) {
	claims, err := s.jwtManager.Verifier.VerifyAccessToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Check blacklist
	blacklisted, err := s.sessionManager.IsTokenBlacklisted(ctx, claims.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check blacklist: %w", err)
	}
	if blacklisted {
		return nil, fmt.Errorf("token has been revoked")
	}

	// Verify session
	_, err = s.sessionManager.GetSession(ctx, claims.IdentityID, claims.ID)
	if err != nil {
		return nil, fmt.Errorf("session not found or expired: %w", err)
	}

	return claims, nil
}

// Add these methods to internal/service/auth/auth_service.go

// ========== Profile Methods ==========

// GetProfile retrieves user profile
func (s *AuthService) GetProfile(ctx context.Context, identityID int64) (*auth.UserProfile, error) {
	profile, err := s.authRepo.GetUserProfile(ctx, identityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	return profile, nil
}

// ========== Email Verification ==========

// VerifyEmail verifies user email using token
func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	// Find verification token
	vToken, err := s.authRepo.FindVerificationToken(ctx, "email_verify", token)
	if err != nil {
		return fmt.Errorf("invalid or expired token")
	}

	// Update identity email verification status
	identity, err := s.authRepo.FindIdentityByID(ctx, vToken.IdentityID)
	if err != nil {
		return fmt.Errorf("identity not found: %w", err)
	}

	identity.EmailVerified = true
	identity.EmailVerifiedAt = sql.NullTime{Time: time.Now(), Valid: true}
	identity.Status = "active" // Activate account after email verification

	if err := s.authRepo.UpdateIdentityStatus(ctx, identity.ID, "active"); err != nil {
		return fmt.Errorf("failed to update identity: %w", err)
	}

	// Mark token as used
	if err := s.authRepo.MarkTokenAsUsed(ctx, vToken.ID); err != nil {
		s.logger.Error("failed to mark token as used", zap.Error(err))
	}

	// Send welcome email
	profile, _ := s.authRepo.GetUserProfile(ctx, identity.ID)
	fullName := "User"
	if profile != nil && profile.FullName.Valid {
		fullName = profile.FullName.String
	}
	s.emailHelper.SendWelcomeEmail(ctx, identity.Email.String, fullName)

	return nil
}

// ResendEmailVerification resends email verification
func (s *AuthService) ResendEmailVerification(ctx context.Context, identityID int64) error {
	identity, err := s.authRepo.FindIdentityByID(ctx, identityID)
	if err != nil {
		return fmt.Errorf("identity not found: %w", err)
	}

	if identity.EmailVerified {
		return fmt.Errorf("email already verified")
	}

	return s.SendEmailVerification(ctx, identityID, identity.Email.String)
}

// ========== Session Management ==========

// GetActiveSessions returns all active sessions for a user
func (s *AuthService) GetActiveSessions(ctx context.Context, identityID int64) ([]*session.SessionData, error) {
	sessions, err := s.sessionManager.GetUserActiveSessions(ctx, identityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	return sessions, nil
}

// RevokeSession revokes a specific session
func (s *AuthService) RevokeSession(ctx context.Context, identityID int64, sessionID string) error {
	if err := s.sessionManager.InvalidateSession(ctx, identityID, sessionID); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	// Blacklist the token
	remainingTTL := time.Until(time.Now().Add(s.jwtManager.Generator.Ttl))
	if err := s.sessionManager.BlacklistToken(ctx, sessionID, remainingTTL); err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	return nil
}

// ========== Admin Management (Super Admin Only) ==========

// CreateAdmin creates a new admin user
func (s *AuthService) CreateAdmin(ctx context.Context, req *auth.CreateAdminRequest, createdBy int64) (*auth.UserProfile, string, error) {
	// Check if email already exists
	exists, err := s.authRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, "", fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return nil, "", xerrors.ErrDuplicateEntry
	}

	// Generate temporary password
	temporaryPassword := generateTemporaryPassword()

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(temporaryPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Create identity
	identity := &auth.Identity{
		Email:  sql.NullString{String: req.Email, Valid: true},
		Phone:  sql.NullString{String: req.Phone, Valid: req.Phone != ""},
		Status: "active", // Admins are active by default
		EmailVerified: true, // Assume email is verified for admins
		EmailVerifiedAt: sql.NullTime{Time: time.Now(), Valid: true},
	}

	if err := s.authRepo.CreateIdentity(ctx, identity); err != nil {
		return nil, "", fmt.Errorf("failed to create identity: %w", err)
	}

	// Create local auth provider
	provider := &auth.Provider{
		IdentityID:   identity.ID,
		Provider:     "local",
		PasswordHash: sql.NullString{String: string(hashedPassword), Valid: true},
		IsPrimary:    true,
	}

	if err := s.authRepo.CreateProvider(ctx, provider); err != nil {
		return nil, "", fmt.Errorf("failed to create provider: %w", err)
	}

	// Create user profile
	profile := &auth.UserProfile{
		IdentityID: identity.ID,
		FullName:   sql.NullString{String: req.FullName, Valid: true},
	}

	if err := s.authRepo.CreateUserProfile(ctx, profile); err != nil {
		return nil, "", fmt.Errorf("failed to create profile: %w", err)
	}

	// Assign roles
	// TODO: Implement role assignment based on req.Roles
	// For now, we'll assume the roles are assigned manually

	// Send account created email
	s.emailHelper.SendAccountCreatedByAdmin(ctx, req.Email, req.FullName, temporaryPassword, req.Roles)

	return profile, temporaryPassword, nil
}

// ListAdmins lists all admin users
func (s *AuthService) ListAdmins(ctx context.Context) ([]*auth.UserProfile, error) {
	// TODO: Implement this by querying users with admin roles
	// This requires adding a method to authRepo to get users by roles
	return nil, fmt.Errorf("not implemented")
}

// DeactivateUser deactivates a user account
func (s *AuthService) DeactivateUser(ctx context.Context, identityID int64) error {
	if err := s.authRepo.UpdateIdentityStatus(ctx, identityID, "inactive"); err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	// Invalidate all sessions
	if err := s.LogoutAllSessions(ctx, identityID); err != nil {
		return fmt.Errorf("failed to invalidate sessions: %w", err)
	}

	// Disconnect WebSocket
	s.hub.DisconnectUser(identityID, "Account deactivated")

	return nil
}

// ========== Helper Functions ==========

// generateTemporaryPassword generates a random temporary password
func generateTemporaryPassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%"
	const length = 12

	b := make([]byte, length)
	rand.Read(b)

	password := make([]byte, length)
	for i := range password {
		password[i] = charset[int(b[i])%len(charset)]
	}

	return string(password)
}