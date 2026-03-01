// internal/middleware/helpers.go
package middleware

import "github.com/gin-gonic/gin"

// MustGetIdentityID gets identity ID from context or panics
func MustGetIdentityID(c *gin.Context) int64 {
	identityID, exists := GetIdentityID(c)
	if !exists {
		panic("identity_id not found in context")
	}
	return identityID
}

// MustGetJTI gets JTI from context or panics
func MustGetJTI(c *gin.Context) string {
	jti, exists := GetJTI(c)
	if !exists {
		panic("jti not found in context")
	}
	return jti
}

// GetRoles gets user roles from context
func GetRoles(c *gin.Context) []string {
	roles, exists := c.Get("roles")
	if !exists {
		return []string{}
	}

	rolesList, ok := roles.([]string)
	if !ok {
		return []string{}
	}

	return rolesList
}

// GetPermissions gets user permissions from context
func GetPermissions(c *gin.Context) []string {
	permissions, exists := c.Get("permissions")
	if !exists {
		return []string{}
	}

	permissionsList, ok := permissions.([]string)
	if !ok {
		return []string{}
	}

	return permissionsList
}

// IsAuthenticated checks if request is authenticated
func IsAuthenticated(c *gin.Context) bool {
	_, exists := c.Get("identity_id")
	return exists
}

// IsAdmin checks if user is an admin
func IsAdmin(c *gin.Context) bool {
	return HasRole(c, "admin") || HasRole(c, "super_admin")
}

// IsSuperAdmin checks if user is a super admin
func IsSuperAdmin(c *gin.Context) bool {
	return HasRole(c, "super_admin")
}