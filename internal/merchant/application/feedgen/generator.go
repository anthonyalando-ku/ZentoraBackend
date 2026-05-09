// Package feedgen implements the feed generation orchestration layer.
// It drives batched loading from the repository, hydration, validation,
// and assembles MerchantFeedBatch slices ready for XML serialisation.
package feedgen

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"zentora-service/internal/merchant/domain"
)

// Generator orchestrates full and incremental merchant feed generation.
type Generator struct {
	repo      merchant.Repository
	hydration merchant.HydrationService
	cfg       merchant.HydrationConfig
	logger    *slog.Logger
}

// New creates a feed generator.
func New(
	repo merchant.Repository,
	hydration merchant.HydrationService,
	cfg merchant.HydrationConfig,
	logger *slog.Logger,
) *Generator {
	return &Generator{
		repo:      repo,
		hydration: hydration,
		cfg:       cfg,
		logger:    logger,
	}
}

// GenerateResult is returned after a complete feed run.
type GenerateResult struct {
	Metadata merchant.MerchantFeedMetadata
	Batches  []*merchant.MerchantFeedBatch
}

// GenerateFull performs a full catalog feed generation.
func (g *Generator) GenerateFull(
	ctx context.Context,
	filter merchant.MerchantFeedFilter,
) (*GenerateResult, error) {
	return g.run(ctx, filter, merchant.FeedExportFull)
}

// GenerateIncremental generates a feed for products updated after `since`.
func (g *Generator) GenerateIncremental(
	ctx context.Context,
	filter merchant.MerchantFeedFilter,
	since time.Time,
) (*GenerateResult, error) {
	filter.UpdatedAfter = &since
	return g.run(ctx, filter, merchant.FeedExportIncremental)
}

func (g *Generator) run(
	ctx context.Context,
	filter merchant.MerchantFeedFilter,
	mode merchant.FeedExportMode,
) (*GenerateResult, error) {
	start := time.Now()

	total, err := g.repo.CountEligibleVariants(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("feedgen: count variants: %w", err)
	}

	g.logger.Info("merchant feed generation started",
		"mode", mode,
		"total_eligible", total,
		"batch_size", filter.Limit,
	)

	var (
		allBatches   []*merchant.MerchantFeedBatch
		validCount   int64
		invalidCount int64
		skippedNoImg int64
		cursor       int64
	)

	for {
		filter.Cursor = cursor

		batch, err := g.repo.LoadFeedBatch(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("feedgen: load batch (cursor=%d): %w", cursor, err)
		}

		if len(batch.Items) == 0 {
			break
		}

		validItems := make([]merchant.MerchantFeedItem, 0, len(batch.Items))
		for i := range batch.Items {
			item := &batch.Items[i]

			// Resolve store URL placeholders
			item.Link = resolveLink(item.Link, g.cfg.StoreBaseURL)
			item.MobileLink = item.Link

			// Apply all normalisation before validation
			normaliseItem(item)

			if item.ImageLink == "" {
				skippedNoImg++
				g.logger.Warn("merchant feed: skipping item — no valid primary image",
					"item_id", item.ID,
					"title", item.Title,
				)
				continue
			}

			if err := validateFeedItem(item); err != nil {
				invalidCount++
				g.logger.Warn("merchant feed: skipping invalid item",
					"item_id", item.ID,
					"title", item.Title,
					"error", err.Error(),
				)
				continue
			}
			validCount++
			validItems = append(validItems, *item)
		}

		batch.Items = validItems
		batch.TotalCount = total
		allBatches = append(allBatches, batch)

		if !batch.HasMore {
			break
		}
		cursor = batch.NextCursor
	}

	meta := merchant.MerchantFeedMetadata{
		FeedID:          fmt.Sprintf("zentora-feed-%s", mode),
		GeneratedAt:     time.Now().UTC(),
		ExportMode:      mode,
		TotalItems:      total,
		ValidItems:      validCount,
		InvalidItems:    invalidCount,
		DurationSeconds: time.Since(start).Seconds(),
		SchemaVersion:   "1.0",
	}

	g.logger.Info("merchant feed generation completed",
		"valid", validCount,
		"invalid", invalidCount,
		"skipped_no_image", skippedNoImg,
		"duration_s", meta.DurationSeconds,
	)

	return &GenerateResult{
		Metadata: meta,
		Batches:  allBatches,
	}, nil
}

// ---------------------------------------------------------------------------
// normaliseItem applies all pre-serialisation cleanup to a feed item.
// This is the single place where empty/invalid fields are zeroed out so
// the XML layer simply emits whatever is non-empty.
// ---------------------------------------------------------------------------

var multiSpaceRe = regexp.MustCompile(`[ \t]{2,}`)
var multiNewlineRe = regexp.MustCompile(`\n{3,}`)

func normaliseItem(item *merchant.MerchantFeedItem) {
	// 1. Images — deduplicate and ensure HTTPS; primary must not appear in
	//    additional links.
	item.ImageLink = normaliseURL(item.ImageLink)
	item.AdditionalImageLinks = deduplicateImages(item.ImageLink, item.AdditionalImageLinks)

	// 2. Title — trim, collapse whitespace, cap at 150 runes.
	item.Title = normaliseTitle(item.Title)

	// 3. Description — clean whitespace, cap at 5000 runes.
	item.Description = normaliseDescription(item.Description)

	// 4. Identifiers — clear empty strings so omitempty works correctly.
	item.GTIN = strings.TrimSpace(item.GTIN)
	item.MPN = strings.TrimSpace(item.MPN)

	// identifier_exists: yes when brand+GTIN/MPN exist; no only for truly
	// unbranded/generic. Never emit "no" just because GTIN/MPN are absent —
	// that triggers Google warnings for branded products.
	hasBrand := item.Brand != "" && !strings.EqualFold(item.Brand, "generic")
	hasIdentifier := item.GTIN != "" || item.MPN != ""
	switch {
	case hasIdentifier:
		item.IdentifierExists = "yes"
	case hasBrand:
		// Brand is an identifier in GMC's model; omit the field entirely so
		// Google infers it rather than being told "no".
		item.IdentifierExists = ""
	default:
		item.IdentifierExists = "no"
	}

	// 5. Pricing — zero out the sale fields if no real sale exists.
	if item.SalePrice == "" || item.SalePrice == item.Price {
		item.SalePrice = ""
		item.SalePriceEffectiveDate = ""
	}

	// 6. Physical — zero out shipping_weight when it is 0 or 1 g (placeholder).
	if item.ShippingWeight == "0 g" || item.ShippingWeight == "1 g" {
		item.ShippingWeight = ""
	}

	// 7. Multipack — zero means "not a multipack product"; omit.
	if item.Multipack <= 1 {
		item.Multipack = 0
	}

	// 8. Adult — omit unless explicitly true; default assumption is non-adult.
	if item.Adult != "yes" {
		item.Adult = ""
	}

	// 9. Variant attributes — blank out fields that carry no real data.
	//    Only keep attributes that are relevant to the product's category.
	item.Size = blankIfIrrelevant(item.Size)
	item.Gender = blankIfIrrelevant(item.Gender)
	item.AgeGroup = blankIfIrrelevant(item.AgeGroup)
	item.Material = blankIfIrrelevant(item.Material)
	item.Pattern = blankIfIrrelevant(item.Pattern)
	item.Color = blankIfIrrelevant(item.Color)

	// 10. Optional fields — never emit empty optional strings.
	item.EnergyEfficiencyClass = strings.TrimSpace(item.EnergyEfficiencyClass)
	item.UnitPricingMeasure = strings.TrimSpace(item.UnitPricingMeasure)
	item.UnitPricingBaseMeasure = strings.TrimSpace(item.UnitPricingBaseMeasure)
	item.AdsRedirect = strings.TrimSpace(item.AdsRedirect)
	item.PickupMethod = strings.TrimSpace(item.PickupMethod)
	item.PickupSLA = strings.TrimSpace(item.PickupSLA)
	item.ExpirationDate = strings.TrimSpace(item.ExpirationDate)
	item.AvailabilityDate = strings.TrimSpace(item.AvailabilityDate)
	item.CostOfGoodsSold = strings.TrimSpace(item.CostOfGoodsSold)

	// 11. Custom labels — blank if empty.
	item.CustomLabel0 = strings.TrimSpace(item.CustomLabel0)
	item.CustomLabel1 = strings.TrimSpace(item.CustomLabel1)
	item.CustomLabel2 = strings.TrimSpace(item.CustomLabel2)
	item.CustomLabel3 = strings.TrimSpace(item.CustomLabel3)
	item.CustomLabel4 = strings.TrimSpace(item.CustomLabel4)

	// 12. Google product category — blank if empty so omitempty fires.
	item.GoogleProductCategory = strings.TrimSpace(item.GoogleProductCategory)
}

func normaliseURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	if strings.HasPrefix(u, "http://") {
		u = "https://" + u[7:]
	}
	if !strings.HasPrefix(u, "https://") {
		return ""
	}
	return u
}

// deduplicateImages removes the primary image from the additional list,
// deduplicates, validates HTTPS, and caps at 10 entries.
func deduplicateImages(primary string, additional []string) []string {
	seen := map[string]struct{}{}
	if primary != "" {
		seen[primary] = struct{}{}
	}
	out := make([]string, 0, len(additional))
	for _, u := range additional {
		u = normaliseURL(u)
		if u == "" {
			continue
		}
		if _, dup := seen[u]; dup {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
		if len(out) == 10 {
			break
		}
	}
	return out
}

func normaliseTitle(s string) string {
	s = strings.TrimSpace(s)
	s = multiSpaceRe.ReplaceAllString(s, " ")
	runes := []rune(s)
	if len(runes) > 150 {
		s = string(runes[:150])
	}
	return s
}

func normaliseDescription(s string) string {
	s = strings.TrimSpace(s)
	// Collapse 3+ consecutive newlines down to two (one blank line).
	s = multiNewlineRe.ReplaceAllString(s, "\n\n")
	// Collapse multiple spaces/tabs on a single line.
	s = multiSpaceRe.ReplaceAllString(s, " ")
	runes := []rune(s)
	if utf8.RuneCountInString(s) > 5000 {
		s = string(runes[:5000])
	}
	return s
}

// blankIfIrrelevant returns empty string for values that indicate no real data.
func blankIfIrrelevant(s string) string {
	s = strings.TrimSpace(s)
	return s
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

func validateFeedItem(item *merchant.MerchantFeedItem) error {
	if item.ID == "" {
		return fmt.Errorf("missing id")
	}
	if strings.TrimSpace(item.Title) == "" {
		return merchant.ErrMissingTitle
	}
	if !isValidHTTPSURL(item.Link) {
		return merchant.ErrMissingLink
	}
	if !isValidHTTPSURL(item.ImageLink) {
		return merchant.ErrMissingImageLink
	}
	if item.Price == "" {
		return merchant.ErrMissingPrice
	}
	if !merchant.Availability(item.Availability).Valid() {
		return merchant.ErrInvalidAvailability
	}
	if !merchant.Condition(item.Condition).Valid() {
		return merchant.ErrInvalidCondition
	}
	return nil
}

func isValidHTTPSURL(u string) bool {
	return strings.HasPrefix(u, "https://") && len(u) > 10
}

func resolveLink(link, baseURL string) string {
	const placeholder = "__STORE_URL__"
	if strings.HasPrefix(link, placeholder) {
		return baseURL + link[len(placeholder):]
	}
	return link
}