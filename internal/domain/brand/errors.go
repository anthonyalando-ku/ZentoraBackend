package brand

import "errors"

var (
	ErrNotFound     = errors.New("brand not found")
	ErrSlugConflict = errors.New("brand slug already exists")
	ErrNameConflict = errors.New("brand name already exists")
	ErrInvalidName  = errors.New("brand name is required and must be at most 255 characters")
	ErrInvalidLogo  = errors.New("logo_url must be at most 500 characters")
)