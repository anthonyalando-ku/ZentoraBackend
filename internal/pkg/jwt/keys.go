package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// parsePEMBytes parses PEM bytes into an RSA private key
func parseRSAPrivateKeyFromPEMBytes(b []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(b)
	if block == nil || (block.Type != "RSA PRIVATE KEY" && block.Type != "PRIVATE KEY") {
		return nil, fmt.Errorf("invalid PEM private key type: %s", block.Type)
	}

	if block.Type == "PRIVATE KEY" {
		// PKCS8 format
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS8 private key: %w", err)
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
		return rsaKey, nil
	}

	// PKCS1 format
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// parsePEMBytes parses PEM bytes into an RSA public key
func parseRSAPublicKeyFromPEMBytes(b []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(b)
	if block == nil || (block.Type != "RSA PUBLIC KEY" && block.Type != "PUBLIC KEY") {
		return nil, fmt.Errorf("invalid PEM public key type: %s", block.Type)
	}

	if block.Type == "PUBLIC KEY" {
		// PKIX format
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKIX public key: %w", err)
		}
		rsaKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA public key")
		}
		return rsaKey, nil
	}

	// PKCS1 format
	return x509.ParsePKCS1PublicKey(block.Bytes)
}

// LoadRSAPrivateKeyFromPEM tries to read from file, if not exists treats input as PEM string
func LoadRSAPrivateKeyFromPEM(pathOrPEM string) (*rsa.PrivateKey, error) {
	b, err := os.ReadFile(pathOrPEM)
	if err != nil {
		// File not found, treat input as PEM string
		b = []byte(pathOrPEM)
	}
	return parseRSAPrivateKeyFromPEMBytes(b)
}

// LoadRSAPublicKeyFromPEM tries to read from file, if not exists treats input as PEM string
func LoadRSAPublicKeyFromPEM(pathOrPEM string) (*rsa.PublicKey, error) {
	b, err := os.ReadFile(pathOrPEM)
	if err != nil {
		// File not found, treat input as PEM string
		b = []byte(pathOrPEM)
	}
	return parseRSAPublicKeyFromPEMBytes(b)
}