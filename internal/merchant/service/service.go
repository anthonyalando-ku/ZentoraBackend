// Package merchantsvc provides the MerchantFeedService — the single
// application-layer entry point for the Google Merchant Center feed system.
//
// It is designed to be constructed once in app.Server.Start() alongside the
// existing catalog / discovery services, following the same pattern.
package merchantsvc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"zentora-service/internal/merchant/domain"
	"zentora-service/internal/merchant/application/feedgen"
	"zentora-service/internal/merchant/application/hydration"

	merchantrepo "zentora-service/internal/merchant/infrastructure/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MerchantFeedService is the top-level orchestrator for feed generation.
// Wire it into app.Server and call GenerateFull / GenerateIncremental
// from a scheduler or an admin HTTP handler.
type MerchantFeedService struct {
	generator  *feedgen.Generator
	repo       merchant.Repository
	cfg        merchant.HydrationConfig
	logger     *slog.Logger
}

// Config holds all runtime knobs for the merchant feed.
type Config struct {
	StoreBaseURL      string  // "https://zentora.com" — no trailing slash
	StoreTitle        string  // shown in the feed <channel><title>
	DefaultCurrency   string  // "KES"
	DefaultCountry    string  // ISO 3166-1 alpha-2, e.g. "KE"
	ShippingFeeKES    float64 // flat shipping fee in KES (e.g. 200)
	TaxRate           float64 // decimal, e.g. 0.16 for Kenyan VAT
	BatchSize         int     // variants per SQL batch, e.g. 500
}

// DefaultConfig returns sensible Zentora production defaults.
func DefaultConfig() Config {
	return Config{
		StoreBaseURL:    "https://zentorashop.co.ke",
		StoreTitle:      "Zentora Shop",
		DefaultCurrency: "KES",
		DefaultCountry:  "KE",
		ShippingFeeKES:  200,
		TaxRate:         0.16, // Kenya standard VAT
		BatchSize:       500,
	}
}

// New constructs the MerchantFeedService.
// Call this in app.Server.Start() after the pgxpool is established.
func New(pool *pgxpool.Pool, cfg Config, logger *slog.Logger) *MerchantFeedService {
	repo := merchantrepo.NewMerchantRepository(pool)
	hydra := hydration.New()

	hydrationCfg := merchant.HydrationConfig{
		StoreBaseURL:    cfg.StoreBaseURL,
		DefaultCurrency: cfg.DefaultCurrency,
		DefaultCondition: merchant.ConditionNew,
		DefaultCountry:  cfg.DefaultCountry,
		DefaultShippingFee: merchant.MoneyFromDecimal(cfg.ShippingFeeKES, cfg.DefaultCurrency),
		DefaultTaxRate:  cfg.TaxRate,
		GoogleProductCategoryMap: ZentoraCategoryMap(),
	}

	gen := feedgen.New(repo, hydra, hydrationCfg, logger)

	return &MerchantFeedService{
		generator: gen,
		repo:      repo,
		cfg:       hydrationCfg,
		logger:    logger,
	}
}

// GenerateFull runs a complete catalog export.
// Returns the result ready for XML serialisation.
func (s *MerchantFeedService) GenerateFull(ctx context.Context) (*feedgen.GenerateResult, error) {
	filter := merchant.DefaultFeedFilter()
	filter.Limit = 500

	result, err := s.generator.GenerateFull(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("merchant feed: full generation: %w", err)
	}
	return result, nil
}

// GenerateIncremental generates only the products modified since `since`.
// Suitable for hourly delta pushes to the Content API.
func (s *MerchantFeedService) GenerateIncremental(ctx context.Context, since time.Time) (*feedgen.GenerateResult, error) {
	filter := merchant.DefaultFeedFilter()
	filter.Limit = 500

	result, err := s.generator.GenerateIncremental(ctx, filter, since)
	if err != nil {
		return nil, fmt.Errorf("merchant feed: incremental generation: %w", err)
	}
	return result, nil
}

// Repo exposes the underlying repository for advanced callers
// (e.g. scheduled jobs that need CountEligibleVariants).
func (s *MerchantFeedService) Repo() merchant.Repository { return s.repo }

// ---------------------------------------------------------------------------
// ZentoraCategoryMap
// Hard-coded GMC taxonomy mapping for the categories in the Zentora DB.
// Matches the seed in 008_merchant_category_seed.sql.
// Keep in sync whenever new categories are added.
// ---------------------------------------------------------------------------

// ZentoraCategoryMap returns the category_id → GMC taxonomy path map.
func ZentoraCategoryMap() map[int64]string {
	return map[int64]string{
		1:  "Electronics",
		12: "Home & Garden > Kitchen & Dining",
		13: "Home & Garden",
		14: "Health & Beauty > Personal Care",
		15: "Sporting Goods",
		16: "Apparel & Accessories",
		17: "Baby & Toddler",
		18: "Health & Beauty",
		19: "Vehicles & Parts",
		20: "Media > Books",
		23: "Vehicles & Parts > Vehicle Parts & Accessories",
		24: "Hardware > Tools > Hand Tools",
		25: "Electronics > Electronics Accessories > Power > Portable Power Stations",
		26: "Hardware > Tools > Power Tools > Air Compressors",
		27: "Electronics > Computers > Tablet Computers",
		28: "Electronics > Communications > Telephony > Mobile Phones",
		29: "Electronics > Networking > Smart Home Devices",
		30: "Electronics > Computers > Tablet Computers",
		31: "Business & Industrial > Agriculture",
		32: "Hardware > Tools",
	}
}