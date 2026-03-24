package auth

import ( 
	"context"

	"time"

	"zentora-service/internal/domain/auth"
	//xerrors "zentora-service/internal/pkg/errors"
	)

func (s *AuthService) AdminListUsers(ctx context.Context, page, size int) (*auth.AdminUsersListResponse, error) {
	rows, total, err := s.authRepo.AdminListUsers(ctx, page, size)
	if err != nil {
		return nil, err
	}

	items := make([]auth.AdminUserCard, 0, len(rows))
	for _, r := range rows {
		var lastLogin *time.Time
		if r.LastLogin.Valid {
			t := r.LastLogin.Time
			lastLogin = &t
		}

		items = append(items, auth.AdminUserCard{
			IdentityID: r.IdentityID,
			Email:      r.Email.String,
			FullName:   r.FullName.String,
			AvatarURL:  r.AvatarURL.String,
			Status:     r.Status,
			Roles:      r.Roles,
			CreatedAt:  r.CreatedAt,
			LastLogin:  lastLogin,
		})
	}

	return &auth.AdminUsersListResponse{
		Items: items,
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}

func (s *AuthService) AdminGetUser(ctx context.Context, identityID int64) (*auth.AdminUserCard, error) {
	row, err := s.authRepo.AdminGetUser(ctx, identityID)
	if err != nil {
		return nil, err
	}

	var lastLogin *time.Time
	if row.LastLogin.Valid {
		t := row.LastLogin.Time
		lastLogin = &t
	}

	return &auth.AdminUserCard{
		IdentityID: row.IdentityID,
		Email:      row.Email.String,
		FullName:   row.FullName.String,
		AvatarURL:  row.AvatarURL.String,
		Status:     row.Status,
		Roles:      row.Roles,
		CreatedAt:  row.CreatedAt,
		LastLogin:  lastLogin,
	}, nil
}

func (s *AuthService) AdminSearchUsers(ctx context.Context, q string, limit int) ([]auth.AdminUserCard, error) {
	rows, err := s.authRepo.AdminSearchUsers(ctx, q, limit)
	if err != nil {
		return nil, err
	}

	out := make([]auth.AdminUserCard, 0, len(rows))
	for _, r := range rows {
		var lastLogin *time.Time
		if r.LastLogin.Valid {
			t := r.LastLogin.Time
			lastLogin = &t
		}

		out = append(out, auth.AdminUserCard{
			IdentityID: r.IdentityID,
			Email:      r.Email.String,
			FullName:   r.FullName.String,
			AvatarURL:  r.AvatarURL.String,
			Status:     r.Status,
			Roles:      r.Roles,
			CreatedAt:  r.CreatedAt,
			LastLogin:  lastLogin,
		})
	}

	return out, nil
}

func (s *AuthService) AdminDeleteUser(ctx context.Context, identityID int64, reason string) error {
	// soft delete identity
	if err := s.authRepo.AdminDeleteUser(ctx, identityID); err != nil {
		return err
	}

	// best-effort: invalidate all sessions + disconnect websockets
	_ = s.LogoutAllSessions(ctx, identityID)
	s.hub.DisconnectUser(identityID, "Account deleted by admin")

	// optional: could write audit log if you already have a repo for it (not included here)
	_ = reason

	return nil
}

func (s *AuthService) AdminUserStats(ctx context.Context) (*auth.AdminUsersStatsResponse, error) {
	return s.authRepo.AdminUserStats(ctx)
}