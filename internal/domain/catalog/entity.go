// internal/domain/catalog/entity.go
package catalog

import (
	"database/sql"
	"time"
)

// ========== Category ==========

// Category represents a product category node.
type Category struct {
	ID       int64          `json:"id" db:"id"`
	Name     string         `json:"name" db:"name"`
	Slug     string         `json:"slug" db:"slug"`
	ParentID sql.NullInt64  `json:"parent_id" db:"parent_id"`
	IsActive bool           `json:"is_active" db:"is_active"`
	CreatedAt time.Time     `json:"created_at" db:"created_at"`
}

// CategoryClosure represents a row in the category_closure closure table.
type CategoryClosure struct {
	AncestorID   int64 `json:"ancestor_id" db:"ancestor_id"`
	DescendantID int64 `json:"descendant_id" db:"descendant_id"`
	Depth        int   `json:"depth" db:"depth"`
}

// ========== Brand ==========

// Brand represents a product brand.
type Brand struct {
	ID        int64          `json:"id" db:"id"`
	Name      string         `json:"name" db:"name"`
	Slug      string         `json:"slug" db:"slug"`
	LogoURL   sql.NullString `json:"logo_url" db:"logo_url"`
	IsActive  bool           `json:"is_active" db:"is_active"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
}

// ========== Tag ==========

// Tag represents a product tag.
type Tag struct {
	ID   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
	Slug string `json:"slug" db:"slug"`
}

// ========== Product ==========

// Product represents a product.
type Product struct {
	ID               int64          `json:"id" db:"id"`
	Name             string         `json:"name" db:"name"`
	Slug             string         `json:"slug" db:"slug"`
	Description      sql.NullString `json:"description" db:"description"`
	ShortDescription sql.NullString `json:"short_description" db:"short_description"`
	BrandID          sql.NullInt64  `json:"brand_id" db:"brand_id"`
	BasePrice        float64        `json:"base_price" db:"base_price"`
	Status           string         `json:"status" db:"status"` // active, draft, archived
	IsFeatured       bool           `json:"is_featured" db:"is_featured"`
	IsDigital        bool           `json:"is_digital" db:"is_digital"`
	Rating           float64        `json:"rating" db:"rating"`
	ReviewCount      int            `json:"review_count" db:"review_count"`
	CreatedBy        sql.NullInt64  `json:"created_by" db:"created_by"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at" db:"updated_at"`
}

// ProductCategoryMap represents the many-to-many between products and categories.
type ProductCategoryMap struct {
	ProductID  int64 `json:"product_id" db:"product_id"`
	CategoryID int64 `json:"category_id" db:"category_id"`
}

// ProductTag represents the many-to-many between products and tags.
type ProductTag struct {
	ProductID int64 `json:"product_id" db:"product_id"`
	TagID     int64 `json:"tag_id" db:"tag_id"`
}

// ProductImage represents an image associated with a product.
type ProductImage struct {
	ID        int64     `json:"id" db:"id"`
	ProductID int64     `json:"product_id" db:"product_id"`
	ImageURL  string    `json:"image_url" db:"image_url"`
	IsPrimary bool      `json:"is_primary" db:"is_primary"`
	SortOrder int       `json:"sort_order" db:"sort_order"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ========== Attributes ==========

// Attribute represents a product attribute dimension (e.g. Color, Size).
type Attribute struct {
	ID                 int64  `json:"id" db:"id"`
	Name               string `json:"name" db:"name"`
	Slug               string `json:"slug" db:"slug"`
	IsVariantDimension bool   `json:"is_variant_dimension" db:"is_variant_dimension"`
}

// AttributeValue represents a specific value for an attribute (e.g. Red, XL).
type AttributeValue struct {
	ID          int64  `json:"id" db:"id"`
	AttributeID int64  `json:"attribute_id" db:"attribute_id"`
	Value       string `json:"value" db:"value"`
}

// ProductAttributeValue is the many-to-many joining products with attribute values.
type ProductAttributeValue struct {
	ProductID        int64 `json:"product_id" db:"product_id"`
	AttributeValueID int64 `json:"attribute_value_id" db:"attribute_value_id"`
}

// ========== Variants ==========

// ProductVariant represents a specific variant of a product (e.g. Red XL).
type ProductVariant struct {
	ID        int64          `json:"id" db:"id"`
	ProductID int64          `json:"product_id" db:"product_id"`
	SKU       string         `json:"sku" db:"sku"`
	Price     float64        `json:"price" db:"price"`
	Weight    sql.NullFloat64 `json:"weight" db:"weight"`
	IsActive  bool           `json:"is_active" db:"is_active"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
}

// VariantAttributeValue is the many-to-many joining variants with attribute values.
type VariantAttributeValue struct {
	VariantID        int64 `json:"variant_id" db:"variant_id"`
	AttributeValueID int64 `json:"attribute_value_id" db:"attribute_value_id"`
}
