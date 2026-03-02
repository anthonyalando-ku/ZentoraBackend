package category

import (
	"database/sql"
	"time"
)

// Category represents a product category node.
type Category struct {
	ID        int64         `json:"id"`
	Name      string        `json:"name"`
	Slug      string        `json:"slug"`
	ParentID  sql.NullInt64 `json:"parent_id"`
	IsActive  bool          `json:"is_active"`
	CreatedAt time.Time     `json:"created_at"`
}

// CategoryClosure represents one row in the category_closure table.
type CategoryClosure struct {
	AncestorID   int64 `json:"ancestor_id"`
	DescendantID int64 `json:"descendant_id"`
	Depth        int   `json:"depth"`
}

// CategoryWithPath is a Category enriched with its full breadcrumb path,
// useful for list/tree responses.
type CategoryWithPath struct {
	Category
	// Ancestors ordered from root → direct parent (depth DESC from closure query).
	Ancestors []CategoryClosure `json:"ancestors,omitempty"`
}