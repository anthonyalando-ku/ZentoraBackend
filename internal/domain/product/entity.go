package product

import (
	"database/sql"
	"time"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusDraft    Status = "draft"
	StatusArchived Status = "archived"
)

type Product struct {
	ID               int64          `json:"id"`
	Name             string         `json:"name"`
	Slug             string         `json:"slug"`
	Description      sql.NullString `json:"description"`
	ShortDescription sql.NullString `json:"short_description"`
	BrandID          sql.NullInt64  `json:"brand_id"`
	BasePrice        float64        `json:"base_price"`
	Status           Status         `json:"status"`
	IsFeatured       bool           `json:"is_featured"`
	IsDigital        bool           `json:"is_digital"`
	Rating           float64        `json:"rating"`
	ReviewCount      int            `json:"review_count"`
	CreatedBy        sql.NullInt64  `json:"created_by"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type Image struct {
	ID        int64     `json:"id"`
	ProductID int64     `json:"product_id"`
	ImageURL  string    `json:"image_url"`
	IsPrimary bool      `json:"is_primary"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

type ProductDetail struct {
	Product
	Images          []Image      `json:"images"`
	Categories      []RelatedRef `json:"categories,omitempty"`
	Tags            []RelatedRef `json:"tags,omitempty"`
	AttributeValues []RelatedRef `json:"attribute_values,omitempty"`
	Variants        []RelatedRef `json:"variants,omitempty"`
}

type RelatedRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ListFilter struct {
	Status     *Status
	BrandID    *int64
	BrandIDs   []int64
	CategoryID *int64
	IsFeatured *bool
	Search     *string

	PriceMin      *float64
	PriceMax      *float64
	MinRating     *float64
	DiscountOnly  bool
	InStockOnly   bool
	TagIDs        []int64
}