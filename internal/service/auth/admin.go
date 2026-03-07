// internal/usecase/auth/auth_service.go
package auth

import (
	"context"
	"database/sql"

	//"errors"
	"fmt"
	"time"

	"zentora-service/internal/domain/auth"
	
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// EnsureSuperAdminExists creates a super admin account if none exists (called on startup)
func (s *AuthService) EnsureSuperAdminExists(ctx context.Context, email, password, fullName string) error {
	// Check if super admin already exists
	exists, err := s.authRepo.SuperAdminExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check super admin existence: %w", err)
	}

	if exists {
		s.logger.Info("super admin already exists, skipping creation")
		return nil
	}

	s.logger.Info("creating super admin account", zap.String("email", email))

	// Validate inputs
	if email == "" || password == "" || fullName == "" {
		return fmt.Errorf("super admin email, password, and name must be provided via environment variables")
	}

	// Check if email already exists (shouldn't happen, but double-check)
	emailExists, err := s.authRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to check email: %w", err)
	}
	if emailExists {
		return fmt.Errorf("email %s already exists but super admin role not assigned", email)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Create identity
	identity := &auth.Identity{
		Email:           sql.NullString{String: email, Valid: true},
		Status:          "active",
		EmailVerified:   true, // Super admin is pre-verified
		EmailVerifiedAt: sql.NullTime{Time: time.Now(), Valid: true},
	}

	if err := s.authRepo.CreateIdentity(ctx, identity); err != nil {
		return fmt.Errorf("failed to create identity: %w", err)
	}

	// Create local auth provider
	provider := &auth.Provider{
		IdentityID:   identity.ID,
		Provider:     "local",
		PasswordHash: sql.NullString{String: string(hashedPassword), Valid: true},
		IsPrimary:    true,
	}

	if err := s.authRepo.CreateProvider(ctx, provider); err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// Create user profile
	profile := &auth.UserProfile{
		IdentityID: identity.ID,
		FullName:   sql.NullString{String: fullName, Valid: true},
	}

	if err := s.authRepo.CreateUserProfile(ctx, profile); err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}

	// Assign super_admin role
	if err := s.authRepo.AssignRoleByName(ctx, identity.ID, "super_admin"); err != nil {
		// log only, since we don't want to fail startup if role assignment fails (though it shouldn't)
		s.logger.Error("failed to assign super admin role", zap.Error(err))
	}

	s.logger.Info("super admin created successfully",
		zap.String("email", email),
		zap.String("full_name", fullName),
		zap.Int64("identity_id", identity.ID),
	)

	return nil
}