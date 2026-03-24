package postgres
import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"strings"

	"zentora-service/internal/domain/auth"
	xerrors "zentora-service/internal/pkg/errors"

)

type AdminUserRow struct {
	IdentityID int64
	Email      sql.NullString
	FullName   sql.NullString
	AvatarURL  sql.NullString
	Status     string
	CreatedAt  time.Time
	LastLogin  sql.NullTime
	Roles      []string
}

func (r *AuthRepository) AdminListUsers(ctx context.Context, page, size int) ([]AdminUserRow, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	offset := (page - 1) * size

	countQuery := `
		SELECT COUNT(1)
		FROM auth_identities ai
		WHERE ai.deleted_at IS NULL
	`
	var total int64
	if err := r.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	query := `
		SELECT
			ai.id,
			ai.email,
			COALESCE(up.full_name, '') AS full_name,
			up.avatar_url,
			ai.status,
			ai.created_at,
			ai.last_login,
			COALESCE(
				ARRAY_AGG(DISTINCT ar.name) FILTER (WHERE ar.name IS NOT NULL),
				ARRAY[]::text[]
			) AS roles
		FROM auth_identities ai
		LEFT JOIN user_profiles up ON up.identity_id = ai.id
		LEFT JOIN auth_identity_roles air ON air.identity_id = ai.id AND air.is_active = TRUE AND (air.expires_at IS NULL OR air.expires_at > NOW())
		LEFT JOIN auth_roles ar ON ar.id = air.role_id AND ar.is_active = TRUE
		WHERE ai.deleted_at IS NULL
		GROUP BY ai.id, ai.email, up.full_name, up.avatar_url, ai.status, ai.created_at, ai.last_login
		ORDER BY ai.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, size, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	out := make([]AdminUserRow, 0, size)
	for rows.Next() {
		var row AdminUserRow
		var fullName string
		err := rows.Scan(
			&row.IdentityID,
			&row.Email,
			&fullName,
			&row.AvatarURL,
			&row.Status,
			&row.CreatedAt,
			&row.LastLogin,
			&row.Roles,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user row: %w", err)
		}
		row.FullName = sql.NullString{String: fullName, Valid: fullName != ""}
		out = append(out, row)
	}

	return out, total, nil
}

func (r *AuthRepository) AdminGetUser(ctx context.Context, identityID int64) (*AdminUserRow, error) {
	query := `
		SELECT
			ai.id,
			ai.email,
			COALESCE(up.full_name, '') AS full_name,
			up.avatar_url,
			ai.status,
			ai.created_at,
			ai.last_login,
			COALESCE(
				ARRAY_AGG(DISTINCT ar.name) FILTER (WHERE ar.name IS NOT NULL),
				ARRAY[]::text[]
			) AS roles
		FROM auth_identities ai
		LEFT JOIN user_profiles up ON up.identity_id = ai.id
		LEFT JOIN auth_identity_roles air ON air.identity_id = ai.id AND air.is_active = TRUE AND (air.expires_at IS NULL OR air.expires_at > NOW())
		LEFT JOIN auth_roles ar ON ar.id = air.role_id AND ar.is_active = TRUE
		WHERE ai.deleted_at IS NULL AND ai.id = $1
		GROUP BY ai.id, ai.email, up.full_name, up.avatar_url, ai.status, ai.created_at, ai.last_login
	`
	var row AdminUserRow
	var fullName string
	err := r.db.QueryRow(ctx, query, identityID).Scan(
		&row.IdentityID,
		&row.Email,
		&fullName,
		&row.AvatarURL,
		&row.Status,
		&row.CreatedAt,
		&row.LastLogin,
		&row.Roles,
	)
	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	row.FullName = sql.NullString{String: fullName, Valid: fullName != ""}
	return &row, nil
}

func (r *AuthRepository) AdminSearchUsers(ctx context.Context, q string, limit int) ([]AdminUserRow, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []AdminUserRow{}, nil
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// basic fuzzy search across email + full name
	query := `
		SELECT
			ai.id,
			ai.email,
			COALESCE(up.full_name, '') AS full_name,
			up.avatar_url,
			ai.status,
			ai.created_at,
			ai.last_login,
			COALESCE(
				ARRAY_AGG(DISTINCT ar.name) FILTER (WHERE ar.name IS NOT NULL),
				ARRAY[]::text[]
			) AS roles
		FROM auth_identities ai
		LEFT JOIN user_profiles up ON up.identity_id = ai.id
		LEFT JOIN auth_identity_roles air ON air.identity_id = ai.id AND air.is_active = TRUE AND (air.expires_at IS NULL OR air.expires_at > NOW())
		LEFT JOIN auth_roles ar ON ar.id = air.role_id AND ar.is_active = TRUE
		WHERE ai.deleted_at IS NULL
		  AND (
			LOWER(COALESCE(ai.email, '')) LIKE LOWER('%' || $1 || '%')
			OR LOWER(COALESCE(up.full_name, '')) LIKE LOWER('%' || $1 || '%')
		  )
		GROUP BY ai.id, ai.email, up.full_name, up.avatar_url, ai.status, ai.created_at, ai.last_login
		ORDER BY ai.created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, q, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	out := make([]AdminUserRow, 0, limit)
	for rows.Next() {
		var row AdminUserRow
		var fullName string
		err := rows.Scan(
			&row.IdentityID,
			&row.Email,
			&fullName,
			&row.AvatarURL,
			&row.Status,
			&row.CreatedAt,
			&row.LastLogin,
			&row.Roles,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search row: %w", err)
		}
		row.FullName = sql.NullString{String: fullName, Valid: fullName != ""}
		out = append(out, row)
	}
	return out, nil
}

// Soft delete user
func (r *AuthRepository) AdminDeleteUser(ctx context.Context, identityID int64) error {
	query := `
		UPDATE auth_identities
		SET deleted_at = NOW(), updated_at = NOW(), status = 'inactive'
		WHERE id = $1 AND deleted_at IS NULL
	`
	ct, err := r.db.Exec(ctx, query, identityID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}
	return nil
}

func (r *AuthRepository) AdminUserStats(ctx context.Context) (*auth.AdminUsersStatsResponse, error) {
	// counts by status
	stats := &auth.AdminUsersStatsResponse{}

	q1 := `
		SELECT
			COUNT(1) AS total,
			COUNT(1) FILTER (WHERE status = 'active') AS active,
			COUNT(1) FILTER (WHERE status = 'inactive') AS inactive,
			COUNT(1) FILTER (WHERE status = 'suspended') AS suspended,
			COUNT(1) FILTER (WHERE status = 'pending_verification') AS pending
		FROM auth_identities
		WHERE deleted_at IS NULL
	`
	if err := r.db.QueryRow(ctx, q1).Scan(
		&stats.TotalUsers,
		&stats.ActiveUsers,
		&stats.InactiveUsers,
		&stats.SuspendedUsers,
		&stats.PendingUsers,
	); err != nil {
		return nil, fmt.Errorf("failed to compute user stats: %w", err)
	}

	// counts by role (active role assignments)
	q2 := `
		SELECT
			COUNT(1) FILTER (WHERE r.name = 'admin') AS admin_users,
			COUNT(1) FILTER (WHERE r.name = 'super_admin') AS super_admin_users
		FROM auth_identity_roles ir
		JOIN auth_roles r ON r.id = ir.role_id
		JOIN auth_identities ai ON ai.id = ir.identity_id
		WHERE ir.is_active = TRUE
		  AND r.is_active = TRUE
		  AND (ir.expires_at IS NULL OR ir.expires_at > NOW())
		  AND ai.deleted_at IS NULL
	`
	if err := r.db.QueryRow(ctx, q2).Scan(&stats.AdminUsers, &stats.SuperAdminUsers); err != nil {
		return nil, fmt.Errorf("failed to compute role stats: %w", err)
	}

	return stats, nil
}