package order

import "time"

// ListFilter is used by both user + admin listing (single implementation).
// Nil fields are ignored.
type ListFilter struct {
	OrderID      *int64
	OrderNumber  *string
	UserID       *int64
	CartID       *int64
	Statuses     []OrderStatus

	CreatedFrom  *time.Time
	CreatedTo    *time.Time

	Limit  int
	Offset int

	// Sorting
	SortBy   string // "created_at", "total_amount", "id"
	SortDesc bool
}