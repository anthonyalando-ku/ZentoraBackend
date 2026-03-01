// internal/pkg/jwt/generator.go
package jwt

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/oklog/ulid/v2"
)

type Generator struct {
	priv     *rsa.PrivateKey
	issuer   string
	audience string
	kid      string // key id for rotation
	Ttl      time.Duration
}

func NewGenerator(priv *rsa.PrivateKey, issuer, audience, kid string, ttl time.Duration) *Generator {
	return &Generator{
		priv:     priv,
		issuer:   issuer,
		audience: audience,
		kid:      kid,
		Ttl:      ttl,
	}
}

// Generate creates a new JWT token with the given parameters
func (g *Generator) Generate(identityID int64, roles []string, permissions []string, device, purpose string, isTemp bool, extraData map[string]interface{}) (string, string, error) {
	if g.priv == nil {
		return "", "", fmt.Errorf("jwt generator has nil private key")
	}

	now := time.Now()
	jti := ulid.Make().String()
	expiresIn := g.Ttl

	// Override TTL for temporary tokens
	if isTemp {
		expiresIn = 30 * time.Minute
	}

	claims := &Claims{
		IdentityID:     identityID,
		Roles:          roles,
		Permissions:    permissions,
		Device:         device,
		IsTemp:         isTemp,
		SessionPurpose: purpose,
		ExtraData:      extraData,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    g.issuer,
			Subject:   fmt.Sprintf("%d", identityID),
			Audience:  []string{g.audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        jti,
		},
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	if g.kid != "" {
		tok.Header["kid"] = g.kid
	}

	signed, err := tok.SignedString(g.priv)
	return signed, jti, err
}

// GenerateAccessToken generates a standard access token
func (g *Generator) GenerateAccessToken(identityID int64, roles []string, permissions []string, device string, extraData map[string]interface{}) (string, string, error) {
	return g.Generate(identityID, roles, permissions, device, "access", false, extraData)
}

// GenerateRefreshToken generates a refresh token (longer TTL)
func (g *Generator) GenerateRefreshToken(identityID int64, device string) (string, string, error) {
	// Refresh tokens don't need roles/permissions, they're only for getting new access tokens
	refreshGenerator := &Generator{
		priv:     g.priv,
		issuer:   g.issuer,
		audience: g.audience,
		kid:      g.kid,
		Ttl:      7 * 24 * time.Hour, // 7 days for refresh token
	}
	return refreshGenerator.Generate(identityID, nil, nil, device, "refresh", false, nil)
}

// GeneratePasswordResetToken generates a temporary token for password reset
func (g *Generator) GeneratePasswordResetToken(identityID int64, extraData map[string]interface{}) (string, string, error) {
	return g.Generate(identityID, nil, nil, "", "password_reset", true, extraData)
}

// GenerateEmailVerificationToken generates a token for email verification
func (g *Generator) GenerateEmailVerificationToken(identityID int64, email string) (string, string, error) {
	extraData := map[string]interface{}{
		"email": email,
	}
	return g.Generate(identityID, nil, nil, "", "email_verification", true, extraData)
}

// GeneratePhoneVerificationToken generates a token for phone verification
func (g *Generator) GeneratePhoneVerificationToken(identityID int64, phone string) (string, string, error) {
	extraData := map[string]interface{}{
		"phone": phone,
	}
	return g.Generate(identityID, nil, nil, "", "phone_verification", true, extraData)
}

// GenerateMagicLinkToken generates a token for passwordless login
func (g *Generator) GenerateMagicLinkToken(identityID int64, email string) (string, string, error) {
	extraData := map[string]interface{}{
		"email": email,
	}
	// Magic links expire in 15 minutes
	tempGenerator := &Generator{
		priv:     g.priv,
		issuer:   g.issuer,
		audience: g.audience,
		kid:      g.kid,
		Ttl:      15 * time.Minute,
	}
	return tempGenerator.Generate(identityID, nil, nil, "", "magic_link", true, extraData)
}