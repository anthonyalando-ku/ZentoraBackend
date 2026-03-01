package xerrors

import (
	"errors"
	"fmt"
)

// Common reusable application errors
var (
	ErrNotFound       = errors.New("resource not found")
	ErrUnauthorized   = errors.New("unauthorized access")
	ErrForbidden      = errors.New("forbidden")
	ErrInvalidInput   = errors.New("invalid input")
	ErrConflict       = errors.New("conflict: resource already exists")
	ErrInternal       = errors.New("internal server error")
	ErrRateLimited    = errors.New("too many requests")
	ErrSessionExpired = errors.New("session expired or invalid")
	ErrBadRequest     = errors.New("bad request")
	ErrDuplicateEntry  = errors.New("duplicate entry")
)

// Wrap adds context to an error (similar to fmt.Errorf("%w")).
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Is allows checking whether an error is a specific sentinel error.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// Unwrap extracts the underlying wrapped error.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// MessageOrDefault returns err.Error() or a fallback message if err is nil.
func MessageOrDefault(err error, fallback string) string {
	if err != nil {
		return err.Error()
	}
	return fallback
}
