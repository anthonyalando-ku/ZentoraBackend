package category

import (
	"strings"
	"unicode/utf8"
)

// ─── Create ──────────────────────────────────────────────────────────────────

// CreateRequest is the inbound payload for creating a category.
// Slug is intentionally absent – it is derived from Name by the service layer.
type CreateRequest struct {
	Name     string `json:"name"`
	ParentID *int64 `json:"parent_id,omitempty"`
	IsActive *bool  `json:"is_active,omitempty"`
}
type SetProductCategoriesRequest struct {
	CategoryIDs []int64 `json:"category_ids"`
}

func (r *SetProductCategoriesRequest) Validate() error {
	for _, id := range r.CategoryIDs {
		if id <= 0 {
			return ErrInvalidCategoryID
		}
	}
	return nil
}

// Validate checks all business rules for a create request.
func (r *CreateRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" || utf8.RuneCountInString(r.Name) > 255 {
		return ErrInvalidName
	}
	if r.ParentID != nil && *r.ParentID <= 0 {
		return ErrInvalidParent
	}
	return nil
}

// ─── Update ──────────────────────────────────────────────────────────────────

// UpdateRequest is the inbound payload for patching a category.
// All fields are optional (pointer = nil means "don't change").
// Slug regeneration is triggered automatically when Name changes.
type UpdateRequest struct {
	Name     *string `json:"name,omitempty"`
	ParentID *int64  `json:"parent_id,omitempty"` // 0 = clear parent
	IsActive *bool   `json:"is_active,omitempty"`
}

// Validate checks all business rules for an update request.
func (r *UpdateRequest) Validate() error {
	if r.Name != nil {
		*r.Name = strings.TrimSpace(*r.Name)
		if *r.Name == "" || utf8.RuneCountInString(*r.Name) > 255 {
			return ErrInvalidName
		}
	}
	// ParentID == 0 is the sentinel for "clear parent" – that's valid.
	// Negative values are not.
	if r.ParentID != nil && *r.ParentID < 0 {
		return ErrInvalidParent
	}
	return nil
}

// ─── List filters ────────────────────────────────────────────────────────────

// ListFilter carries optional filters for listing categories.
type ListFilter struct {
	ParentID   *int64 // nil = all, 0 = roots only
	ActiveOnly bool
}