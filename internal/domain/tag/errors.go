package tag

import "errors"

var (
	ErrNotFound    = errors.New("tag not found")
	ErrInvalidName = errors.New("tag name is required and must be at most 100 characters")
)