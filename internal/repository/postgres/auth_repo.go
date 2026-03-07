// internal/repository/postgres/auth_repository.go
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"zentora-service/internal/domain/auth"
	xerrors "zentora-service/internal/pkg/errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepository struct {
	db *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{db: db}
}

// ========== Identity Methods ==========

// FindIdentityByEmail retrieves an identity by email
func (r *AuthRepository) FindIdentityByEmail(ctx context.Context, email string) (*auth.Identity, error) {
	query := `
		SELECT id, email, email_verified, email_verified_at, 
		       phone, phone_verified, phone_verified_at,
		       status, last_login, failed_login_attempts, locked_until,
		       created_at, updated_at, deleted_at
		FROM auth_identities
		WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL
	`

	var identity auth.Identity
	err := r.db.QueryRow(ctx, query, email).Scan(
		&identity.ID, &identity.Email, &identity.EmailVerified, &identity.EmailVerifiedAt,
		&identity.Phone, &identity.PhoneVerified, &identity.PhoneVerifiedAt,
		&identity.Status, &identity.LastLogin, &identity.FailedLoginAttempts, &identity.LockedUntil,
		&identity.CreatedAt, &identity.UpdatedAt, &identity.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find identity: %w", err)
	}

	return &identity, nil
}

// FindIdentityByPhone retrieves an identity by phone
func (r *AuthRepository) FindIdentityByPhone(ctx context.Context, phone string) (*auth.Identity, error) {
	query := `
		SELECT id, email, email_verified, email_verified_at, 
		       phone, phone_verified, phone_verified_at,
		       status, last_login, failed_login_attempts, locked_until,
		       created_at, updated_at, deleted_at
		FROM auth_identities
		WHERE phone = $1 AND deleted_at IS NULL
	`

	var identity auth.Identity
	err := r.db.QueryRow(ctx, query, phone).Scan(
		&identity.ID, &identity.Email, &identity.EmailVerified, &identity.EmailVerifiedAt,
		&identity.Phone, &identity.PhoneVerified, &identity.PhoneVerifiedAt,
		&identity.Status, &identity.LastLogin, &identity.FailedLoginAttempts, &identity.LockedUntil,
		&identity.CreatedAt, &identity.UpdatedAt, &identity.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find identity: %w", err)
	}

	return &identity, nil
}

// FindIdentityByID retrieves an identity by ID
func (r *AuthRepository) FindIdentityByID(ctx context.Context, id int64) (*auth.Identity, error) {
	query := `
		SELECT id, email, email_verified, email_verified_at, 
		       phone, phone_verified, phone_verified_at,
		       status, last_login, failed_login_attempts, locked_until,
		       created_at, updated_at, deleted_at
		FROM auth_identities
		WHERE id = $1 AND deleted_at IS NULL
	`

	var identity auth.Identity
	err := r.db.QueryRow(ctx, query, id).Scan(
		&identity.ID, &identity.Email, &identity.EmailVerified, &identity.EmailVerifiedAt,
		&identity.Phone, &identity.PhoneVerified, &identity.PhoneVerifiedAt,
		&identity.Status, &identity.LastLogin, &identity.FailedLoginAttempts, &identity.LockedUntil,
		&identity.CreatedAt, &identity.UpdatedAt, &identity.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find identity: %w", err)
	}

	return &identity, nil
}

// CreateIdentity creates a new identity
func (r *AuthRepository) CreateIdentity(ctx context.Context, identity *auth.Identity) error {
	query := `
		INSERT INTO auth_identities (email, phone, status)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query, identity.Email, identity.Phone, identity.Status).
		Scan(&identity.ID, &identity.CreatedAt, &identity.UpdatedAt)

	return err
}

// UpdateIdentityLastLogin updates the last login timestamp
func (r *AuthRepository) UpdateIdentityLastLogin(ctx context.Context, id int64) error {
	query := `
		UPDATE auth_identities 
		SET last_login = $1, failed_login_attempts = 0, locked_until = NULL 
		WHERE id = $2
	`
	_, err := r.db.Exec(ctx, query, time.Now(), id)
	return err
}

// IncrementFailedLoginAttempts increments failed login attempts
func (r *AuthRepository) IncrementFailedLoginAttempts(ctx context.Context, id int64, lockDuration time.Duration) error {
	query := `
		UPDATE auth_identities 
		SET failed_login_attempts = failed_login_attempts + 1,
		    locked_until = CASE 
		        WHEN failed_login_attempts + 1 >= 5 THEN $1 
		        ELSE NULL 
		    END
		WHERE id = $2
	`
	lockUntil := time.Now().Add(lockDuration)
	_, err := r.db.Exec(ctx, query, lockUntil, id)
	return err
}

// UpdateIdentityStatus updates identity status
func (r *AuthRepository) UpdateIdentityStatus(ctx context.Context, id int64, status string) error {
	query := `UPDATE auth_identities SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.Exec(ctx, query, status, time.Now(), id)
	return err
}

// ========== Provider Methods ==========

// FindProviderByIdentityAndType finds a provider by identity ID and provider type
func (r *AuthRepository) FindProviderByIdentityAndType(ctx context.Context, identityID int64, providerType string) (*auth.Provider, error) {
	query := `
		SELECT id, identity_id, provider, provider_user_id, provider_email, provider_username,
		       password_hash, access_token, refresh_token, token_expires_at,
		       provider_data, is_primary, password_changed_at, created_at, updated_at
		FROM auth_providers
		WHERE identity_id = $1 AND provider = $2
	`

	var provider auth.Provider
	var providerDataJSON []byte

	err := r.db.QueryRow(ctx, query, identityID, providerType).Scan(
		&provider.ID, &provider.IdentityID, &provider.Provider, &provider.ProviderUserID,
		&provider.ProviderEmail, &provider.ProviderUsername, &provider.PasswordHash,
		&provider.AccessToken, &provider.RefreshToken, &provider.TokenExpiresAt,
		&providerDataJSON, &provider.IsPrimary, &provider.PasswordChangedAt,
		&provider.CreatedAt, &provider.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}

	if len(providerDataJSON) > 0 {
		if err := json.Unmarshal(providerDataJSON, &provider.ProviderData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal provider data: %w", err)
		}
	}

	return &provider, nil
}

// CreateProvider creates a new auth provider
func (r *AuthRepository) CreateProvider(ctx context.Context, provider *auth.Provider) error {
	query := `
		INSERT INTO auth_providers (
			identity_id, provider, provider_user_id, provider_email, provider_username,
			password_hash, access_token, refresh_token, token_expires_at,
			provider_data, is_primary
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`

	var providerDataJSON []byte
	var err error
	if provider.ProviderData != nil {
		providerDataJSON, err = json.Marshal(provider.ProviderData)
		if err != nil {
			return fmt.Errorf("failed to marshal provider data: %w", err)
		}
	}

	err = r.db.QueryRow(
		ctx, query,
		provider.IdentityID, provider.Provider, provider.ProviderUserID,
		provider.ProviderEmail, provider.ProviderUsername, provider.PasswordHash,
		provider.AccessToken, provider.RefreshToken, provider.TokenExpiresAt,
		providerDataJSON, provider.IsPrimary,
	).Scan(&provider.ID, &provider.CreatedAt, &provider.UpdatedAt)

	return err
}

// UpdateProviderPassword updates the password hash for a local provider
func (r *AuthRepository) UpdateProviderPassword(ctx context.Context, id int64, passwordHash string) error {
	query := `
		UPDATE auth_providers 
		SET password_hash = $1, password_changed_at = $2, updated_at = $3 
		WHERE id = $4 AND provider = 'local'
	`
	_, err := r.db.Exec(ctx, query, passwordHash, time.Now(), time.Now(), id)
	return err
}

// UpdateProviderTokens updates OAuth tokens
func (r *AuthRepository) UpdateProviderTokens(ctx context.Context, id int64, accessToken, refreshToken string, expiresAt time.Time) error {
	query := `
		UPDATE auth_providers 
		SET access_token = $1, refresh_token = $2, token_expires_at = $3, updated_at = $4
		WHERE id = $5
	`
	_, err := r.db.Exec(ctx, query, accessToken, refreshToken, expiresAt, time.Now(), id)
	return err
}

// ========== Session Methods ==========

// CreateSession creates a new session
func (r *AuthRepository) CreateSession(ctx context.Context, session *auth.Session) error {
	query := `
		INSERT INTO auth_sessions (
			identity_id, session_token, refresh_token, provider, ip_address,
			user_agent, device_id, device_name, device_fingerprint,
			expires_at, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, login_at, last_activity_at
	`

	var metadataJSON []byte
	var err error
	if session.Metadata != nil {
		metadataJSON, err = json.Marshal(session.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	err = r.db.QueryRow(
		ctx, query,
		session.IdentityID, session.SessionToken, session.RefreshToken,
		session.Provider, session.IPAddress, session.UserAgent,
		session.DeviceID, session.DeviceName, session.DeviceFingerprint,
		session.ExpiresAt, metadataJSON,
	).Scan(&session.ID, &session.LoginAt, &session.LastActivityAt)

	return err
}

// FindSessionByToken finds a session by session token
func (r *AuthRepository) FindSessionByToken(ctx context.Context, token string) (*auth.Session, error) {
	query := `
		SELECT id, identity_id, session_token, refresh_token, provider,
		       ip_address, user_agent, device_id, device_name, device_fingerprint,
		       status, login_at, last_activity_at, expires_at, logout_at, metadata
		FROM auth_sessions
		WHERE session_token = $1 AND status = 'active' AND expires_at > NOW()
	`

	var session auth.Session
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, token).Scan(
		&session.ID, &session.IdentityID, &session.SessionToken, &session.RefreshToken,
		&session.Provider, &session.IPAddress, &session.UserAgent, &session.DeviceID,
		&session.DeviceName, &session.DeviceFingerprint, &session.Status,
		&session.LoginAt, &session.LastActivityAt, &session.ExpiresAt,
		&session.LogoutAt, &metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find session: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &session.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &session, nil
}

// UpdateSessionActivity updates the last activity timestamp
func (r *AuthRepository) UpdateSessionActivity(ctx context.Context, id int64) error {
	query := `UPDATE auth_sessions SET last_activity_at = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, time.Now(), id)
	return err
}

// InvalidateSession invalidates a session
func (r *AuthRepository) InvalidateSession(ctx context.Context, id int64) error {
	query := `
		UPDATE auth_sessions 
		SET status = 'revoked', logout_at = $1 
		WHERE id = $2
	`
	_, err := r.db.Exec(ctx, query, time.Now(), id)
	return err
}

// InvalidateAllUserSessions invalidates all sessions for a user
func (r *AuthRepository) InvalidateAllUserSessions(ctx context.Context, identityID int64) error {
	query := `
		UPDATE auth_sessions 
		SET status = 'revoked', logout_at = $1 
		WHERE identity_id = $2 AND status = 'active'
	`
	_, err := r.db.Exec(ctx, query, time.Now(), identityID)
	return err
}

// ========== User Profile Methods ==========

// GetUserProfile retrieves user profile
func (r *AuthRepository) GetUserProfile(ctx context.Context, identityID int64) (*auth.UserProfile, error) {
	query := `
		SELECT id, identity_id, full_name, avatar_url, bio, metadata, created_at, updated_at
		FROM user_profiles
		WHERE identity_id = $1
	`

	var profile auth.UserProfile
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, identityID).Scan(
		&profile.ID, &profile.IdentityID, &profile.FullName, &profile.AvatarURL,
		&profile.Bio, &metadataJSON, &profile.CreatedAt, &profile.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &profile.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &profile, nil
}

// CreateUserProfile creates a new user profile
func (r *AuthRepository) CreateUserProfile(ctx context.Context, profile *auth.UserProfile) error {
	query := `
		INSERT INTO user_profiles (identity_id, full_name, avatar_url, bio, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	var metadataJSON []byte
	var err error
	if profile.Metadata != nil {
		metadataJSON, err = json.Marshal(profile.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	err = r.db.QueryRow(
		ctx, query,
		profile.IdentityID, profile.FullName, profile.AvatarURL,
		profile.Bio, metadataJSON,
	).Scan(&profile.ID, &profile.CreatedAt, &profile.UpdatedAt)

	return err
}

// UpdateUserProfile updates user profile
func (r *AuthRepository) UpdateUserProfile(ctx context.Context, identityID int64, profile *auth.UserProfile) error {
	query := `
		UPDATE user_profiles 
		SET full_name = $1, avatar_url = $2, bio = $3, metadata = $4, updated_at = $5
		WHERE identity_id = $6
	`

	var metadataJSON []byte
	var err error
	if profile.Metadata != nil {
		metadataJSON, err = json.Marshal(profile.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	result, err := r.db.Exec(
		ctx, query,
		profile.FullName, profile.AvatarURL, profile.Bio,
		metadataJSON, time.Now(), identityID,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// ========== Verification Token Methods ==========

// CreateVerificationToken creates a verification token
func (r *AuthRepository) CreateVerificationToken(ctx context.Context, token *auth.VerificationToken) error {
	query := `
		INSERT INTO auth_verification_tokens (identity_id, token_type, token, code, expires_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	var metadataJSON []byte
	var err error
	if token.Metadata != nil {
		metadataJSON, err = json.Marshal(token.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	err = r.db.QueryRow(
		ctx, query,
		token.IdentityID, token.TokenType, token.Token,
		token.Code, token.ExpiresAt, metadataJSON,
	).Scan(&token.ID, &token.CreatedAt)

	return err
}

// FindVerificationToken finds a valid verification token
func (r *AuthRepository) FindVerificationToken(ctx context.Context, tokenType, token string) (*auth.VerificationToken, error) {
	query := `
		SELECT id, identity_id, token_type, token, code, expires_at, used_at, 
		       verified_at, attempts, metadata, created_at
		FROM auth_verification_tokens
		WHERE token_type = $1 AND token = $2 AND expires_at > NOW() AND used_at IS NULL
	`

	var vToken auth.VerificationToken
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, tokenType, token).Scan(
		&vToken.ID, &vToken.IdentityID, &vToken.TokenType, &vToken.Token,
		&vToken.Code, &vToken.ExpiresAt, &vToken.UsedAt, &vToken.VerifiedAt,
		&vToken.Attempts, &metadataJSON, &vToken.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find verification token: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &vToken.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &vToken, nil
}

// MarkTokenAsUsed marks a verification token as used
func (r *AuthRepository) MarkTokenAsUsed(ctx context.Context, id int64) error {
	query := `UPDATE auth_verification_tokens SET used_at = $1, verified_at = $2 WHERE id = $3`
	_, err := r.db.Exec(ctx, query, time.Now(), time.Now(), id)
	return err
}

// IncrementTokenAttempts increments token verification attempts
func (r *AuthRepository) IncrementTokenAttempts(ctx context.Context, id int64) error {
	query := `UPDATE auth_verification_tokens SET attempts = attempts + 1 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// ========== Utility Methods ==========

// ExistsByEmail checks if an identity with email exists
func (r *AuthRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM auth_identities WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL)`
	var exists bool
	err := r.db.QueryRow(ctx, query, email).Scan(&exists)
	return exists, err
}

// ExistsByPhone checks if an identity with phone exists
func (r *AuthRepository) ExistsByPhone(ctx context.Context, phone string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM auth_identities WHERE phone = $1 AND deleted_at IS NULL)`
	var exists bool
	err := r.db.QueryRow(ctx, query, phone).Scan(&exists)
	return exists, err
}

// ========== Role Management ==========

// GetUserRoles retrieves all roles for a user
func (r *AuthRepository) GetUserRoles(ctx context.Context, identityID int64) ([]string, error) {
	query := `
		SELECT r.name
		FROM auth_identity_roles ir
		JOIN auth_roles r ON ir.role_id = r.id
		WHERE ir.identity_id = $1
		  AND ir.is_active = TRUE
		  AND (ir.expires_at IS NULL OR ir.expires_at > NOW())
		  AND r.is_active = TRUE
	`

	rows, err := r.db.Query(ctx, query, identityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// GetUserPermissions retrieves all permissions for a user
func (r *AuthRepository) GetUserPermissions(ctx context.Context, identityID int64) ([]string, error) {
	query := `
		SELECT DISTINCT p.name
		FROM auth_permissions p
		WHERE 
			-- From role permissions
			p.id IN (
				SELECT rp.permission_id
				FROM auth_identity_roles ir
				JOIN auth_role_permissions rp ON ir.role_id = rp.role_id
				WHERE ir.identity_id = $1
				  AND ir.is_active = TRUE
				  AND (ir.expires_at IS NULL OR ir.expires_at > NOW())
			)
			-- Add user-specific grants
			OR p.id IN (
				SELECT ip.permission_id
				FROM auth_identity_permissions ip
				WHERE ip.identity_id = $1
				  AND ip.is_granted = TRUE
				  AND (ip.expires_at IS NULL OR ip.expires_at > NOW())
			)
			-- Exclude user-specific revokes
			AND p.id NOT IN (
				SELECT ip.permission_id
				FROM auth_identity_permissions ip
				WHERE ip.identity_id = $1
				  AND ip.is_granted = FALSE
				  AND (ip.expires_at IS NULL OR ip.expires_at > NOW())
			)
			AND p.is_active = TRUE
	`

	rows, err := r.db.Query(ctx, query, identityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

// AssignRole assigns a role to a user
func (r *AuthRepository) AssignRole(ctx context.Context, identityID, roleID, assignedBy int64) error {
	query := `
		INSERT INTO auth_identity_roles (identity_id, role_id, assigned_by, is_active)
		VALUES ($1, $2, $3, TRUE)
		ON CONFLICT (identity_id, role_id) DO UPDATE
		SET is_active = TRUE, assigned_by = $3, assigned_at = NOW()
	`

	_, err := r.db.Exec(ctx, query, identityID, roleID, assignedBy)
	return err
}

// GetRoleByName gets a role by its name
func (r *AuthRepository) GetRoleByName(ctx context.Context, name string) (*auth.Role, error) {
	query := `
		SELECT id, name, display_name, description, is_system, is_active, created_at, updated_at
		FROM auth_roles
		WHERE name = $1
	`

	var role auth.Role
	err := r.db.QueryRow(ctx, query, name).Scan(
		&role.ID, &role.Name, &role.DisplayName, &role.Description,
		&role.IsSystem, &role.IsActive, &role.CreatedAt, &role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return &role, nil
}

func (r *AuthRepository) SuperAdminExists(ctx context.Context) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 
			FROM auth_identity_roles ir
			JOIN auth_roles r ON ir.role_id = r.id
			WHERE r.name = 'super_admin'
		)
	`
	
	var exists bool
	err := r.db.QueryRow(ctx, query).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check super admin existence: %w", err)
	}
	
	return exists, nil
}

// AssignRole assigns a role to a user
func (r *AuthRepository) AssignRoleByName(ctx context.Context, identityID int64, roleName string) error {
	query := `
		INSERT INTO auth_identity_roles (identity_id, role_id)
		SELECT $1, id FROM auth_roles WHERE name = $2
		ON CONFLICT (identity_id, role_id) DO NOTHING
	`
	
	_, err := r.db.Exec(ctx, query, identityID, roleName)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}
	
	return nil
}