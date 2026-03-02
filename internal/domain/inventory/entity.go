package inventory

import (
	"database/sql"
	"time"
)

type Location struct {
	ID           int64          `json:"id"`
	Name         string         `json:"name"`
	LocationCode sql.NullString `json:"location_code"`
	IsActive     bool           `json:"is_active"`
	CreatedAt    time.Time      `json:"created_at"`
}

type Item struct {
	ID           int64     `json:"id"`
	VariantID    int64     `json:"variant_id"`
	LocationID   int64     `json:"location_id"`
	AvailableQty int       `json:"available_qty"`
	ReservedQty  int       `json:"reserved_qty"`
	IncomingQty  int       `json:"incoming_qty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ItemWithLocation struct {
	Item
	LocationName string         `json:"location_name"`
	LocationCode sql.NullString `json:"location_code"`
}

// StockSummary aggregates stock across all locations for a variant.
type StockSummary struct {
	VariantID    int64 `json:"variant_id"`
	AvailableQty int   `json:"available_qty"`
	ReservedQty  int   `json:"reserved_qty"`
	IncomingQty  int   `json:"incoming_qty"`
}