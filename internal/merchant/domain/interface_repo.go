package merchant

import (
	"context"
	"time"
)

// ---------------------------------------------------------------------------
// Repository interface
// ---------------------------------------------------------------------------

// Repository is the primary port for loading merchant feed data.
// Implementations are in the infrastructure/postgres layer.
type Repository interface {
	// LoadFeedBatch returns a batch of feed-ready items using keyset pagination.
	// Cursor is the last seen variant_id from the previous batch (0 = start).
	LoadFeedBatch(ctx context.Context, filter MerchantFeedFilter) (*MerchantFeedBatch, error)

	// LoadProduct returns a fully-hydrated MerchantProduct for a single product.
	// Primarily used for real-time content API updates.
	LoadProduct(ctx context.Context, productID int64) (*MerchantProduct, error)

	// LoadVariant returns a single hydrated variant feed item.
	LoadVariant(ctx context.Context, variantID int64) (*MerchantVariant, error)

	// CountEligibleVariants returns the total count of publishable variants
	// matching the given filter. Used for feed metadata.
	CountEligibleVariants(ctx context.Context, filter MerchantFeedFilter) (int64, error)

	// LoadUpdatedVariantIDs returns variant IDs modified after the given time.
	// Used to drive incremental feed generation.
	LoadUpdatedVariantIDs(ctx context.Context, since time.Time, limit int) ([]int64, error)
}

// HydrationService enriches raw MerchantProduct records into per-variant
// MerchantVariant feed items. Implementations can run pure in-process.
type HydrationService interface {
	// HydrateProduct transforms a MerchantProduct into a slice of feed-ready
	// MerchantVariant records (one per active, purchasable variant).
	HydrateProduct(
		ctx context.Context,
		product *MerchantProduct,
		cfg HydrationConfig,
	) ([]*MerchantVariant, error)
}

// HydrationConfig carries runtime configuration for feed hydration.
type HydrationConfig struct {
	StoreBaseURL      string // "https://zentorashop.co.ke"
	DefaultCurrency   string // "KES"
	DefaultCondition  Condition
	DefaultCountry    string // "KE" — for shipping annotations
	DefaultShippingFee Money
	DefaultTaxRate     float64 // e.g. 0.16
	GoogleProductCategoryMap map[int64]string // category_id → GMC taxonomy path
}