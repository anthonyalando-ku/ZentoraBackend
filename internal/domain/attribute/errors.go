package attribute

import "errors"

var (
	ErrNotFound           = errors.New("attribute not found")
	ErrValueNotFound      = errors.New("attribute value not found")
	ErrSlugConflict       = errors.New("attribute slug already exists")
	ErrValueConflict      = errors.New("attribute value already exists for this attribute")
	ErrInvalidName        = errors.New("attribute name is required and must be at most 100 characters")
	ErrInvalidValue       = errors.New("attribute value is required and must be at most 100 characters")
	ErrInvalidSlug        = errors.New("attribute slug is required and must be at most 100 characters")
)