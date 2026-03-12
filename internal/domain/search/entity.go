package productsearch

import "time"

type Document struct {
	ProductID       int64
	SearchDocument  string
	SearchVector    string // not returned directly from DB usually; keep for completeness
	UpdatedAt       time.Time
}