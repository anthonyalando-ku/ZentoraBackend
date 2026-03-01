// internal/pkg/jwt/claims.go
package jwt

import (
	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT claims
type Claims struct {
	IdentityID     int64                  `json:"identity_id"`
	Roles          []string               `json:"roles,omitempty"`
	Permissions    []string               `json:"permissions,omitempty"`
	Device         string                 `json:"device,omitempty"`
	IsTemp         bool                   `json:"is_temp"`
	SessionPurpose string                 `json:"session_purpose"` // access, refresh, password_reset, email_verification, etc.
	ExtraData      map[string]interface{} `json:"extra_data,omitempty"`
	jwt.RegisteredClaims
}

// HasRole checks if the claims contain a specific role
func (c *Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if the claims contain a specific permission
func (c *Claims) HasPermission(permission string) bool {
	for _, p := range c.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// IsSuperAdmin checks if user is a super admin
func (c *Claims) IsSuperAdmin() bool {
	return c.HasRole("super_admin")
}

// IsAdmin checks if user is an admin (including super admin)
func (c *Claims) IsAdmin() bool {
	return c.HasRole("admin") || c.HasRole("super_admin")
}

// VerifyAudience checks if the expected audience is listed in the claims.
func (c *Claims) VerifyAudience(audience string, required bool) bool {
	if len(c.Audience) == 0 {
		// If audience is required but missing
		return !required
	}

	for _, aud := range c.Audience {
		if aud == audience {
			return true
		}
	}

	return false
}
