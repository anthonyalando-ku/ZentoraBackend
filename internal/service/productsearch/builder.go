package productsearch

import (
	"strings"

	"zentora-service/internal/domain/product"
)

// BuildSearchDocument builds the document used for full-text search.
// Keep it deterministic and stable (so updates don't churn unnecessarily).
func BuildSearchDocument(p *product.Product) string {
	parts := make([]string, 0, 6)

	if p == nil {
		return ""
	}
	if s := strings.TrimSpace(p.Name); s != "" {
		parts = append(parts, s)
	}
	if s := strings.TrimSpace(p.Slug); s != "" {
		parts = append(parts, s)
	}
	shortDescriptionStr := ""
	if p.ShortDescription.Valid {
		shortDescriptionStr = strings.TrimSpace(p.ShortDescription.String)
	}
	descriptionStr := ""
	if p.Description.Valid {
		descriptionStr = strings.TrimSpace(p.Description.String)
	}
	if shortDescriptionStr != "" {
		parts = append(parts, shortDescriptionStr)
	}
	// optionally include description (can be long; but helps search)
	if descriptionStr != "" {
		parts = append(parts, descriptionStr)
	}

	// later: add brand name, category names, tags, variant SKUs, attribute values, etc.
	return strings.Join(parts, " ")
}