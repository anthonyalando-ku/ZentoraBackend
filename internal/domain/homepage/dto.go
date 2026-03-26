package homepage

import (
	"strings"
	"unicode/utf8"
)

type CreateSectionRequest struct {
	Title       *string     `json:"title,omitempty"`
	Type        SectionType `json:"type"`
	ReferenceID *int64      `json:"reference_id,omitempty"`
	SortOrder   int         `json:"sort_order"`
	IsActive    *bool       `json:"is_active,omitempty"`
}

func (r *CreateSectionRequest) Validate() error {
	if !r.Type.Valid() {
		return ErrInvalidType
	}
	if r.Title != nil {
		*r.Title = strings.TrimSpace(*r.Title)
		if utf8.RuneCountInString(*r.Title) > 255 {
			return ErrInvalidTitle
		}
	}
	// category sections must reference a category
	if r.Type == SectionTypeCategory && (r.ReferenceID == nil || *r.ReferenceID <= 0) {
		return ErrReferenceRequired
	}
	if r.SortOrder < 0 {
		return ErrInvalidSortOrder
	}
	return nil
}

type UpdateSectionRequest struct {
	Title       *string      `json:"title,omitempty"`
	Type        *SectionType `json:"type,omitempty"`
	ReferenceID *int64       `json:"reference_id,omitempty"`
	SortOrder   *int         `json:"sort_order,omitempty"`
	IsActive    *bool        `json:"is_active,omitempty"`
}

func (r *UpdateSectionRequest) Validate() error {
	if r.Type != nil && !r.Type.Valid() {
		return ErrInvalidType
	}
	if r.Title != nil {
		*r.Title = strings.TrimSpace(*r.Title)
		if utf8.RuneCountInString(*r.Title) > 255 {
			return ErrInvalidTitle
		}
	}
	if r.SortOrder != nil && *r.SortOrder < 0 {
		return ErrInvalidSortOrder
	}
	return nil
}

// ReorderRequest replaces all section sort orders atomically.
type ReorderRequest struct {
	// Items maps section_id -> new sort_order
	Items []ReorderItem `json:"items"`
}

type ReorderItem struct {
	ID        int64 `json:"id"`
	SortOrder int   `json:"sort_order"`
}

func (r *ReorderRequest) Validate() error {
	if len(r.Items) == 0 {
		return ErrEmptyReorder
	}
	seen := make(map[int64]struct{}, len(r.Items))
	for _, item := range r.Items {
		if item.ID <= 0 {
			return ErrInvalidID
		}
		if item.SortOrder < 0 {
			return ErrInvalidSortOrder
		}
		if _, dup := seen[item.ID]; dup {
			return ErrDuplicateID
		}
		seen[item.ID] = struct{}{}
	}
	return nil
}

// ListFilter controls which sections are fetched.
type ListFilter struct {
	ActiveOnly bool
	Type       *SectionType
}