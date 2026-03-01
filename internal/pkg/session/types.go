// internal/pkg/session/types.go
package session

import "time"

type SessionData struct {
	JTI               string                 `json:"jti"`
	IdentityID        int64                  `json:"identity_id"`
	SessionID         int64                  `json:"session_id"` // DB session ID
	Email             string                 `json:"email"`
	Roles             []string               `json:"roles"`
	Permissions       []string               `json:"permissions"`
	Device            string                 `json:"device,omitempty"`
	DeviceID          string                 `json:"device_id,omitempty"`
	DeviceName        string                 `json:"device_name,omitempty"`
	IPAddress         string                 `json:"ip_address"`
	UserAgent         string                 `json:"user_agent"`
	Provider          string                 `json:"provider"` // local, google, etc.
	LoginAt           time.Time              `json:"login_at"`
	LastActivityAt    time.Time              `json:"last_activity_at"`
	ExpiresAt         time.Time              `json:"expires_at"`
	IsActive          bool                   `json:"is_active"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// UserInfo represents minimal user information in session
type UserInfo struct {
	IdentityID int64    `json:"identity_id"`
	Email      string   `json:"email"`
	FullName   string   `json:"full_name"`
	Roles      []string `json:"roles"`
}