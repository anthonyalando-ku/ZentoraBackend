// internal/domain/admin/repository.go
package admin

import (
	"context"
	"time"
)

type Repository interface {
	Create(ctx context.Context, a *Admin) (*Admin, error)
	Update(ctx context.Context, a *Admin) error
	Delete(ctx context.Context, id int64) error
	// Authentication
	FindByEmail(ctx context.Context, email string) (*Admin, error)
	FindByID(ctx context.Context, id int64) (*Admin, error)
	UpdateLastLogin(ctx context.Context, id int64) error
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
	
	// Password Reset
	SetPasswordResetToken(ctx context.Context, id int64, token string, expiresAt time.Time) error
	FindByResetToken(ctx context.Context, token string) (*Admin, error)
	ClearPasswordResetToken(ctx context.Context, id int64) error
	
	// Session Management
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, adminID int64, jti string) (*Session, error)
	InvalidateSession(ctx context.Context, sessionID int64) error
	InvalidateAllSessions(ctx context.Context, adminID int64) error
}

type Session struct {
	ID           int64     `db:"id"`
	AdminID      int64     `db:"admin_id"`
	SessionToken string    `db:"session_token"`
	IPAddress    *string   `db:"ip_address"`
	UserAgent    *string   `db:"user_agent"`
	LoginAt      time.Time `db:"login_at"`
	LogoutAt     *time.Time `db:"logout_at"`
	IsActive     bool      `db:"is_active"`
}
