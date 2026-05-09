// Package hydration implements merchant.HydrationService.
// It converts raw MerchantProduct + MerchantProductVariant records
// (as loaded by the repository) into feed-ready MerchantVariant items.
// This layer is pure domain logic — no SQL, no HTTP.
package hydration

import (
	"context"
	"fmt"
	"strings"

	"zentora-service/internal/merchant/domain"
)

// Service implements merchant.HydrationService.
type Service struct{}

// New returns a production-ready HydrationService.
func New() *Service { return &Service{} }

// HydrateProduct transforms one MerchantProduct into a slice of
// MerchantVariant feed items — one per active, purchasable variant.
func (s *Service) HydrateProduct(
	_ context.Context,
	p *merchant.MerchantProduct,
	cfg merchant.HydrationConfig,
) ([]*merchant.MerchantVariant, error) {

	if len(p.Variants) == 0 {
		return nil, merchant.ErrNoVariants
	}

	cur := cfg.DefaultCurrency
	if cur == "" {
		cur = "KES"
	}

	cond := cfg.DefaultCondition
	if !cond.Valid() {
		cond = merchant.ConditionNew
	}

	// Pre-select primary and additional images
	primaryImg, additionalImgs := partitionImages(p.Images)

	var items []*merchant.MerchantVariant

	for i := range p.Variants {
		v := &p.Variants[i]
		if !v.IsActive {
			continue
		}

		// --- Pricing ---
		pricing := resolvePricing(v, cur)

		// --- Availability ---
		avail := merchant.MerchantAvailability{
			AvailableQty: v.AvailableQty,
			ReservedQty:  v.ReservedQty,
			IncomingQty:  v.IncomingQty,
		}
		avail.Status = merchant.ResolveAvailability(v.AvailableQty, v.ReservedQty, v.IncomingQty)

		// --- Google Product Category ---
		gpc := p.GoogleProductCategory
		if gpc == "" && cfg.GoogleProductCategoryMap != nil {
			if mapped, ok := cfg.GoogleProductCategoryMap[p.PrimaryCategoryID]; ok {
				gpc = mapped
			}
		}

		// --- Custom labels ---
		labels := buildCustomLabels(v, p)

		// --- Variant title ---
		title := buildTitle(p.Name, v.Attributes)
		if utf8RuneLen(title) > 150 {
			title = truncateRunes(title, 150)
		}

		// --- Description ---
		desc := strings.TrimSpace(p.Description)
		if utf8RuneLen(desc) > 5000 {
			desc = truncateRunes(desc, 5000)
		}

		item := &merchant.MerchantVariant{
			ProductID:   p.ProductID,
			VariantID:   v.VariantID,
			ID:          fmt.Sprintf("%d-%d", p.ProductID, v.VariantID),
			ItemGroupID: merchant.BuildItemGroupID(p.ProductID),
			SKU:         v.SKU,
			GTIN:        v.GTIN,
			MPN:         v.MPN,
			ProductSlug: p.Slug,
			VariantName: buildVariantName(v.Attributes),

			Title:       title,
			Description: desc,
			Link:        merchant.BuildVariantURL(cfg.StoreBaseURL, p.Slug, v.VariantID),
			MobileLink:  merchant.BuildVariantURL(cfg.StoreBaseURL, p.Slug, v.VariantID),

			ImageLink:            primaryImg,
			AdditionalImageLinks: additionalImgs,

			Availability: avail,
			Pricing:      pricing,
			Condition:    cond,

			Brand:                 p.BrandName,
			GoogleProductCategory: gpc,
			ProductType:           p.CategoryPath,
			CategoryPath:          p.CategoryPath,

			Color:    v.Attributes["color"],
			Size:     v.Attributes["size"],
			Gender:   merchant.Gender(v.Attributes["gender"]),
			AgeGroup: merchant.AgeGroup(v.Attributes["age_group"]),
			Material: v.Attributes["material"],
			Pattern:  v.Attributes["pattern"],

			ShippingWeightGrams: v.WeightGrams,
			Shipping:            buildShipping(cfg),
			Tax:                 buildTax(cfg),
			CustomLabels:        labels,

			IdentifierExists: v.GTIN != "" || v.MPN != "",
			UpdatedAt:        p.UpdatedAt,
		}

		item.NormalizeImages()

		items = append(items, item)
	}

	if len(items) == 0 {
		return nil, merchant.ErrNoVariants
	}

	return items, nil
}

// ---------------------------------------------------------------------------
// Pricing resolution
// ---------------------------------------------------------------------------

func resolvePricing(v *merchant.MerchantProductVariant, currency string) merchant.MerchantPricing {
	regular := merchant.MoneyFromDecimal(v.Price, currency)
	pricing := merchant.MerchantPricing{
		Price:           regular,
		DiscountPercent: v.DiscountPercent,
		DiscountType:    v.DiscountType,
	}

	if v.DiscountPercent <= 0 {
		return pricing
	}

	var saleCents int64
	switch v.DiscountType {
	case "percentage":
		saleCents = int64(float64(regular.AmountCents) * (1.0 - v.DiscountPercent/100.0))
	case "fixed":
		// DiscountPercent was already normalised to a percent equivalent by the
		// query; use the raw discount value stored in DiscountPercent for safety.
		saleCents = regular.AmountCents - int64(v.DiscountPercent/100.0*float64(regular.AmountCents))
	default:
		saleCents = int64(float64(regular.AmountCents) * (1.0 - v.DiscountPercent/100.0))
	}

	if saleCents < 0 {
		saleCents = 0
	}

	pricing.SalePrice = merchant.Money{AmountCents: saleCents, Currency: currency}
	pricing.SalePriceEffectiveStart = v.DiscountStartsAt
	pricing.SalePriceEffectiveEnd = v.DiscountEndsAt

	return pricing
}

// ---------------------------------------------------------------------------
// Image partitioning
// ---------------------------------------------------------------------------

func partitionImages(images []merchant.MerchantImage) (primary string, additional []string) {
	for _, img := range images {
		url := img.URL
		if strings.HasPrefix(url, "http://") {
			url = "https://" + url[7:]
		}
		if img.IsPrimary && primary == "" {
			primary = url
		} else if len(additional) < 10 {
			additional = append(additional, url)
		}
	}
	// Fallback: use first image as primary if none explicitly marked
	if primary == "" && len(images) > 0 {
		url := images[0].URL
		if strings.HasPrefix(url, "http://") {
			url = "https://" + url[7:]
		}
		primary = url
	}
	return
}

// ---------------------------------------------------------------------------
// Shipping / Tax builders
// ---------------------------------------------------------------------------

func buildShipping(cfg merchant.HydrationConfig) []merchant.MerchantShipping {
	if cfg.DefaultCountry == "" {
		return nil
	}
	fee := cfg.DefaultShippingFee
	if fee.Currency == "" {
		fee.Currency = cfg.DefaultCurrency
	}
	return []merchant.MerchantShipping{
		{
			Country:      cfg.DefaultCountry,
			Service:      "Standard Shipping",
			PriceAmount:  fee,
		},
	}
}

func buildTax(cfg merchant.HydrationConfig) []merchant.MerchantTax {
	if cfg.DefaultTaxRate == 0 || cfg.DefaultCountry == "" {
		return nil
	}
	return []merchant.MerchantTax{
		{
			Country: cfg.DefaultCountry,
			Rate:    cfg.DefaultTaxRate,
			TaxShip: false,
		},
	}
}

// ---------------------------------------------------------------------------
// Title / variant name helpers
// ---------------------------------------------------------------------------

func buildTitle(productName string, attrs map[string]string) string {
	parts := []string{productName}
	if color := attrs["color"]; color != "" {
		parts = append(parts, color)
	}
	if size := attrs["size"]; size != "" {
		parts = append(parts, size)
	}
	if storage := attrs["storage"]; storage != "" {
		parts = append(parts, storage)
	}
	if len(parts) == 1 {
		return productName
	}
	return parts[0] + " - " + strings.Join(parts[1:], " / ")
}

func buildVariantName(attrs map[string]string) string {
	var parts []string
	for _, key := range []string{"color", "size", "storage", "weight"} {
		if v := attrs[key]; v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, " / ")
}

// ---------------------------------------------------------------------------
// Custom label logic
//-- Label0: discount tier ("sale-high" / "sale-low" / "no-sale")
//-- Label1: availability ("in-stock" / "low-stock" / "out-of-stock")
// ---------------------------------------------------------------------------

func buildCustomLabels(v *merchant.MerchantProductVariant, p *merchant.MerchantProduct) merchant.MerchantCustomLabels {
	var labels merchant.MerchantCustomLabels

	// Label 0 — discount tier
	switch {
	case v.DiscountPercent >= 30:
		labels.Label0 = "sale-high"
	case v.DiscountPercent > 0:
		labels.Label0 = "sale-low"
	default:
		labels.Label0 = "no-sale"
	}

	// Label 1 — stock tier
	net := v.AvailableQty - v.ReservedQty
	switch {
	case net <= 0:
		labels.Label1 = "out-of-stock"
	case net <= 5:
		labels.Label1 = "low-stock"
	default:
		labels.Label1 = "in-stock"
	}

	// Label 2 — rating tier
	switch {
	case p.Rating >= 4.5:
		labels.Label2 = "top-rated"
	case p.Rating >= 3.5:
		labels.Label2 = "well-rated"
	default:
		labels.Label2 = ""
	}

	// Label 3 — category slug (useful for Smart Shopping campaigns)
	labels.Label3 = p.BrandSlug

	return labels
}

// ---------------------------------------------------------------------------
// Rune-safe string helpers
// ---------------------------------------------------------------------------

func utf8RuneLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}