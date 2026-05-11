package productsearch

import (
	"strings"

	"zentora-service/internal/domain/product"
)

// SearchInput carries all enriched data needed to build a complete search vector.
// The repository populates this from the same transaction data available at
// create/update time, so no extra DB round-trips are needed.
type SearchInput struct {
	Product *product.Product

	// Resolved names — pass the actual strings, not IDs.
	BrandName     string
	CategoryNames []string
	TagNames      []string

	// Variant-level data
	VariantSKUs []string

	// Attribute names and their resolved values (e.g. "Color: Black, Red")
	// Pass both the attribute name and each value name.
	AttributeNames []string
	AttributeValues []string // flat list of all value names across all attributes
}

// BuildSearchDocument builds the tsvector source string used for full-text search.
//
// Field weighting strategy (passed to to_tsvector in SQL, or done here in order):
//   - Product name, brand name         → highest signal
//   - Category names, tag names        → high signal
//   - Attribute names + values, SKUs   → medium signal
//   - Short description                → medium signal
//   - Full description                 → lower signal (verbose but useful)
//   - Slug                             → low signal (already derived from name)
//
// Keep output deterministic and stable so unchanged products don't get
// unnecessary tsvector updates.
func BuildSearchDocument(in *SearchInput) string {
	if in == nil || in.Product == nil {
		return ""
	}
	p := in.Product

	// Pre-allocate generously — most products have ~10-20 tokens
	parts := make([]string, 0, 20)

	// ── High-signal fields ────────────────────────────────────────────────────

	if s := strings.TrimSpace(p.Name); s != "" {
		parts = append(parts, s)
	}
	if s := strings.TrimSpace(in.BrandName); s != "" {
		parts = append(parts, s)
	}

	// ── Category and tag names ────────────────────────────────────────────────

	for _, c := range in.CategoryNames {
		if s := strings.TrimSpace(c); s != "" {
			parts = append(parts, s)
		}
	}
	for _, t := range in.TagNames {
		if s := strings.TrimSpace(t); s != "" {
			parts = append(parts, s)
		}
	}

	// ── Attribute names and values ────────────────────────────────────────────
	// e.g. "Color", "Black", "RAM", "8GB", "Storage", "128GB"
	// Including both name and value lets users search "8GB RAM tablet" or just "8GB".

	for _, a := range in.AttributeNames {
		if s := strings.TrimSpace(a); s != "" {
			parts = append(parts, s)
		}
	}
	for _, v := range in.AttributeValues {
		if s := strings.TrimSpace(v); s != "" {
			parts = append(parts, s)
		}
	}

	// ── Variant SKUs ──────────────────────────────────────────────────────────
	// Useful for exact SKU searches from admin or returning customers.

	for _, sku := range in.VariantSKUs {
		if s := strings.TrimSpace(sku); s != "" {
			parts = append(parts, s)
		}
	}

	// ── Descriptions ─────────────────────────────────────────────────────────

	if p.ShortDescription.Valid {
		if s := strings.TrimSpace(p.ShortDescription.String); s != "" {
			parts = append(parts, s)
		}
	}
	if p.Description.Valid {
		if s := strings.TrimSpace(p.Description.String); s != "" {
			parts = append(parts, s)
		}
	}

	// ── Slug (lowest signal — already derived from name) ─────────────────────

	if s := strings.TrimSpace(p.Slug); s != "" {
		parts = append(parts, s)
	}

	return strings.Join(parts, " ")
}