// internal/pkg/jwt/verifier.go
package jwt

import (
	"crypto/rsa"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type Verifier struct {
	pub      *rsa.PublicKey
	issuer   string
	audience string
}

func NewVerifier(pub *rsa.PublicKey, issuer, audience string) *Verifier {
	return &Verifier{
		pub:      pub,
		issuer:   issuer,
		audience: audience,
	}
}

// Verify validates a JWT token and returns the claims
func (v *Verifier) Verify(tokenString string) (*Claims, error) {
	if v.pub == nil {
		return nil, fmt.Errorf("jwt verifier has nil public key")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.pub, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Verify issuer
	if claims.Issuer != v.issuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", v.issuer, claims.Issuer)
	}

	// Verify audience
	if !claims.VerifyAudience(v.audience, true) {
		return nil, fmt.Errorf("invalid audience")
	}

	return claims, nil
}

// VerifyAccessToken verifies that the token is for access purposes
func (v *Verifier) VerifyAccessToken(tokenString string) (*Claims, error) {
	claims, err := v.Verify(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.SessionPurpose != "access" {
		return nil, fmt.Errorf("token is not an access token")
	}

	if claims.IsTemp {
		return nil, fmt.Errorf("cannot use temporary token for access")
	}

	return claims, nil
}

// VerifyRefreshToken verifies that the token is for refresh purposes
func (v *Verifier) VerifyRefreshToken(tokenString string) (*Claims, error) {
	claims, err := v.Verify(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.SessionPurpose != "refresh" {
		return nil, fmt.Errorf("token is not a refresh token")
	}

	return claims, nil
}

// VerifyPasswordResetToken verifies that the token is for password reset
func (v *Verifier) VerifyPasswordResetToken(tokenString string) (*Claims, error) {
	claims, err := v.Verify(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.SessionPurpose != "password_reset" {
		return nil, fmt.Errorf("token is not for password reset")
	}

	if !claims.IsTemp {
		return nil, fmt.Errorf("password reset token must be temporary")
	}

	return claims, nil
}

// VerifyEmailVerificationToken verifies email verification token
func (v *Verifier) VerifyEmailVerificationToken(tokenString string) (*Claims, error) {
	claims, err := v.Verify(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.SessionPurpose != "email_verification" {
		return nil, fmt.Errorf("token is not for email verification")
	}

	return claims, nil
}

// VerifyPhoneVerificationToken verifies phone verification token
func (v *Verifier) VerifyPhoneVerificationToken(tokenString string) (*Claims, error) {
	claims, err := v.Verify(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.SessionPurpose != "phone_verification" {
		return nil, fmt.Errorf("token is not for phone verification")
	}

	return claims, nil
}

// VerifyMagicLinkToken verifies magic link token
func (v *Verifier) VerifyMagicLinkToken(tokenString string) (*Claims, error) {
	claims, err := v.Verify(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.SessionPurpose != "magic_link" {
		return nil, fmt.Errorf("token is not a magic link")
	}

	return claims, nil
}

// HasRole checks if the claims contain a specific role
func (v *Verifier) HasRole(claims *Claims, role string) bool {
	for _, r := range claims.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if the claims contain a specific permission
func (v *Verifier) HasPermission(claims *Claims, permission string) bool {
	for _, p := range claims.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the claims contain any of the specified roles
func (v *Verifier) HasAnyRole(claims *Claims, roles ...string) bool {
	for _, role := range roles {
		if v.HasRole(claims, role) {
			return true
		}
	}
	return false
}

// HasAllRoles checks if the claims contain all of the specified roles
func (v *Verifier) HasAllRoles(claims *Claims, roles ...string) bool {
	for _, role := range roles {
		if !v.HasRole(claims, role) {
			return false
		}
	}
	return true
}

// HasAnyPermission checks if the claims contain any of the specified permissions
func (v *Verifier) HasAnyPermission(claims *Claims, permissions ...string) bool {
	for _, permission := range permissions {
		if v.HasPermission(claims, permission) {
			return true
		}
	}
	return false
}

// HasAllPermissions checks if the claims contain all of the specified permissions
func (v *Verifier) HasAllPermissions(claims *Claims, permissions ...string) bool {
	for _, permission := range permissions {
		if !v.HasPermission(claims, permission) {
			return false
		}
	}
	return true
}