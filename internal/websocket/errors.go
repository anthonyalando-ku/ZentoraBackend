// internal/websocket/errors.go
package websocket

import "errors"

var (
	ErrTokenBlacklisted = errors.New("token has been blacklisted")
	ErrSessionExpired   = errors.New("session has expired")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrInvalidToken     = errors.New("invalid token")
)