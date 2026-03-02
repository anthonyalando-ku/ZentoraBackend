package variant

import (
	"database/sql"
	"time"
)

type Variant struct {
	ID        int64           `json:"id"`
	ProductID int64           `json:"product_id"`
	SKU       string          `json:"sku"`
	Price     float64         `json:"price"`
	Weight    sql.NullFloat64 `json:"weight"`
	IsActive  bool            `json:"is_active"`
	CreatedAt time.Time       `json:"created_at"`
}

type VariantWithAttributes struct {
	Variant
	AttributeValueIDs []int64 `json:"attribute_value_ids"`
}