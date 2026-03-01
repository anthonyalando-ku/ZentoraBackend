// internal/token/loader.go
package jwt

import (
	"fmt"
	"time"
)

type Config struct {
	PrivPath string
	PubPath  string
	Issuer   string
	Audience string
	TTL      time.Duration
	KID      string
}

type Manager struct {
	Generator *Generator
	Verifier  *Verifier
}

func LoadAndBuild(cfg Config) (*Manager, error) {
	// Load private key
	priv, err := LoadRSAPrivateKeyFromPEM(cfg.PrivPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key from %s: %w", cfg.PrivPath, err)
	}

	// Load public key
	pub, err := LoadRSAPublicKeyFromPEM(cfg.PubPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load public key from %s: %w", cfg.PubPath, err)
	}

	gen := NewGenerator(priv, cfg.Issuer, cfg.Audience, cfg.KID, cfg.TTL)
	ver := NewVerifier(pub, cfg.Issuer, cfg.Audience)

	return &Manager{
		Generator: gen,
		Verifier:  ver,
	}, nil
}