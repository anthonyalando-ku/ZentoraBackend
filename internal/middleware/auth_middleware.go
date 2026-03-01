// internal/middleware/auth_middleware.go
package middleware

import (
	"errors"
	"net/http"
	"strings"

	"diary-service/internal/pkg/response"
	"diary-service/internal/service/auth"

	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	authService *auth.AuthService
}

func NewAuthMiddleware(authService *auth.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// Auth is the base authentication middleware that validates JWT tokens
func (m *AuthMiddleware) Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			response.Error(c, http.StatusUnauthorized, "missing authorization token", nil)
			return
		}

		claims, err := m.authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "invalid or expired token", err)
			return
		}

		// Set user context
		c.Set("identity_id", claims.IdentityID)
		c.Set("jti", claims.ID)
		c.Set("roles", claims.Roles)
		c.Set("permissions", claims.Permissions)
		c.Set("device", claims.Device)
		c.Set("session_purpose", claims.SessionPurpose)
		c.Set("is_temp", claims.IsTemp)

		// Set extra data if available
		if claims.ExtraData != nil {
			for key, value := range claims.ExtraData {
				c.Set(key, value)
			}
		}

		c.Next()
	}
}

// RequireRole middleware that requires user to have at least one of the specified roles
// MUST be used after Auth() middleware
func (m *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get("roles")
		if !exists {
			response.Error(c, http.StatusForbidden, "no roles found - authentication required", nil)
			return
		}

		userRolesList, ok := userRoles.([]string)
		if !ok {
			response.Error(c, http.StatusInternalServerError, "invalid roles format", nil)
			return
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, userRole := range userRolesList {
			for _, requiredRole := range roles {
				if userRole == requiredRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			err := errors.New("user does not have required role")
			response.Error(c, http.StatusForbidden, "insufficient permissions", err, map[string]interface{}{
				"required_roles": roles,
				"user_roles":     userRolesList,
			})
			return
		}

		c.Next()
	}
}

// RequirePermission middleware that requires user to have at least one of the specified permissions
// MUST be used after Auth() middleware
func (m *AuthMiddleware) RequirePermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userPermissions, exists := c.Get("permissions")
		if !exists {
			response.Error(c, http.StatusForbidden, "no permissions found - authentication required", nil)
			return
		}

		userPermissionsList, ok := userPermissions.([]string)
		if !ok {
			response.Error(c, http.StatusInternalServerError, "invalid permissions format", nil)
			return
		}

		// Check if user has any of the required permissions
		hasPermission := false
		for _, userPerm := range userPermissionsList {
			for _, requiredPerm := range permissions {
				if userPerm == requiredPerm {
					hasPermission = true
					break
				}
			}
			if hasPermission {
				break
			}
		}

		if !hasPermission {
			err := errors.New("user does not have required permission")
			response.Error(c, http.StatusForbidden, "insufficient permissions", err, map[string]interface{}{
				"required_permissions": permissions,
				"user_permissions":     userPermissionsList,
			})
			return
		}

		c.Next()
	}
}

// RequireAllRoles middleware that requires user to have ALL specified roles
// MUST be used after Auth() middleware
func (m *AuthMiddleware) RequireAllRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get("roles")
		if !exists {
			response.Error(c, http.StatusForbidden, "no roles found - authentication required", nil)
			return
		}

		userRolesList, ok := userRoles.([]string)
		if !ok {
			response.Error(c, http.StatusInternalServerError, "invalid roles format", nil)
			return
		}

		// Check if user has all required roles
		userRoleMap := make(map[string]bool)
		for _, role := range userRolesList {
			userRoleMap[role] = true
		}

		for _, requiredRole := range roles {
			if !userRoleMap[requiredRole] {
				err := errors.New("user does not have all required roles")
				response.Error(c, http.StatusForbidden, "insufficient permissions", err, map[string]interface{}{
					"required_roles": roles,
					"user_roles":     userRolesList,
					"missing_role":   requiredRole,
				})
				return
			}
		}

		c.Next()
	}
}

// RequireAllPermissions middleware that requires user to have ALL specified permissions
// MUST be used after Auth() middleware
func (m *AuthMiddleware) RequireAllPermissions(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userPermissions, exists := c.Get("permissions")
		if !exists {
			response.Error(c, http.StatusForbidden, "no permissions found - authentication required", nil)
			return
		}

		userPermissionsList, ok := userPermissions.([]string)
		if !ok {
			response.Error(c, http.StatusInternalServerError, "invalid permissions format", nil)
			return
		}

		// Check if user has all required permissions
		userPermMap := make(map[string]bool)
		for _, perm := range userPermissionsList {
			userPermMap[perm] = true
		}

		for _, requiredPerm := range permissions {
			if !userPermMap[requiredPerm] {
				err := errors.New("user does not have all required permissions")
				response.Error(c, http.StatusForbidden, "insufficient permissions", err, map[string]interface{}{
					"required_permissions": permissions,
					"user_permissions":     userPermissionsList,
					"missing_permission":   requiredPerm,
				})
				return
			}
		}

		c.Next()
	}
}

// Composed middleware functions that combine Auth + Role checks
// These are convenience functions that return multiple middlewares

// AdminOnly returns middlewares for admin-only routes (Auth + RequireRole)
func (m *AuthMiddleware) AdminOnly() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		m.Auth(),
		m.RequireRole("admin", "super_admin"),
	}
}

// SuperAdminOnly returns middlewares for super admin-only routes (Auth + RequireRole)
func (m *AuthMiddleware) SuperAdminOnly() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		m.Auth(),
		m.RequireRole("super_admin"),
	}
}

// WithPermission returns middlewares for permission-based routes (Auth + RequirePermission)
func (m *AuthMiddleware) WithPermission(permissions ...string) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		m.Auth(),
		m.RequirePermission(permissions...),
	}
}

// OptionalAuth middleware that doesn't abort if no token is provided
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.Next()
			return
		}

		claims, err := m.authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			// Don't abort, just continue without setting user context
			c.Next()
			return
		}

		// Set user context
		c.Set("identity_id", claims.IdentityID)
		c.Set("jti", claims.ID)
		c.Set("roles", claims.Roles)
		c.Set("permissions", claims.Permissions)
		c.Set("authenticated", true)

		c.Next()
	}
}

// extractToken extracts Bearer token from Authorization header
func extractToken(c *gin.Context) string {
	// Try header first
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}

	// Fallback to query param (use with caution in production)
	token := c.Query("token")
	if token != "" {
		return token
	}

	return ""
}

// Helper function to get identity ID from context
func GetIdentityID(c *gin.Context) (int64, bool) {
	identityID, exists := c.Get("identity_id")
	if !exists {
		return 0, false
	}

	id, ok := identityID.(int64)
	return id, ok
}

// Helper function to get JTI from context
func GetJTI(c *gin.Context) (string, bool) {
	jti, exists := c.Get("jti")
	if !exists {
		return "", false
	}

	jtiStr, ok := jti.(string)
	return jtiStr, ok
}

// Helper function to check if user has role
func HasRole(c *gin.Context, role string) bool {
	roles, exists := c.Get("roles")
	if !exists {
		return false
	}

	rolesList, ok := roles.([]string)
	if !ok {
		return false
	}

	for _, r := range rolesList {
		if r == role {
			return true
		}
	}

	return false
}

// Helper function to check if user has permission
func HasPermission(c *gin.Context, permission string) bool {
	permissions, exists := c.Get("permissions")
	if !exists {
		return false
	}

	permissionsList, ok := permissions.([]string)
	if !ok {
		return false
	}

	for _, p := range permissionsList {
		if p == permission {
			return true
		}
	}

	return false
}