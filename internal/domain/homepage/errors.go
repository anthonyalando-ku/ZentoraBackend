package homepage

import "errors"

var (
	ErrNotFound         = errors.New("homepage section not found")
	ErrInvalidType      = errors.New("type must be trending, featured, category, or custom")
	ErrInvalidTitle     = errors.New("title must be at most 255 characters")
	ErrInvalidSortOrder = errors.New("sort_order must be >= 0")
	ErrReferenceRequired = errors.New("reference_id is required for category sections")
	ErrEmptyReorder     = errors.New("reorder list must not be empty")
	ErrInvalidID        = errors.New("section id must be > 0")
	ErrDuplicateID      = errors.New("duplicate section id in reorder list")
)