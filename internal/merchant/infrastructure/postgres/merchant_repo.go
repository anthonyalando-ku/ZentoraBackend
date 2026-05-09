// Package postgres implements the merchant.Repository interface against the
// Zentora PostgreSQL schema.  It reuses the discount, inventory, image, and
// category hydration patterns established in the discovery engine but is
// completely independent of ranking or recommendation logic.
package postgres

import (
	"context"
	"fmt"
	"time"

	"zentora-service/internal/merchant/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MerchantRepository implements merchant.Repository.
type MerchantRepository struct {
	db *pgxpool.Pool
}

// NewMerchantRepository creates a production-ready merchant repository.
func NewMerchantRepository(db *pgxpool.Pool) *MerchantRepository {
	return &MerchantRepository{db: db}
}

// ---------------------------------------------------------------------------
// LoadFeedBatch  (primary paginated fetch)
// ---------------------------------------------------------------------------

// LoadFeedBatch loads a page of fully-hydrated feed items using efficient
// keyset pagination on variant_id.  One query fetches all required data
// via CTEs — no N+1 patterns.
func (r *MerchantRepository) LoadFeedBatch(
	ctx context.Context,
	filter merchant.MerchantFeedFilter,
) (*merchant.MerchantFeedBatch, error) {
 
	rows, err := r.db.Query(ctx, queryLoadFeedBatch,
		filter.Cursor,                   // $1 keyset cursor
		filter.Limit,                    // $2 batch size
		toNullBool(filter.ActiveOnly),   // $3
		toNullBool(filter.InStockOnly),  // $4
		toNullBool(filter.ExcludeDigital), // $5
		toInt64Array(filter.CategoryIDs),  // $6
		toInt64Array(filter.BrandIDs),     // $7
		filter.UpdatedAfter,               // $8 nullable timestamp
	)
	if err != nil {
		return nil, fmt.Errorf("merchant: load feed batch: %w", err)
	}
	defer rows.Close()
 
	return scanFeedBatchRows(rows, filter.Limit)
}
 
// ---------------------------------------------------------------------------
// LoadProduct  (single product hydration)
// ---------------------------------------------------------------------------
 
func (r *MerchantRepository) LoadProduct(
	ctx context.Context,
	productID int64,
) (*merchant.MerchantProduct, error) {
 
	rows, err := r.db.Query(ctx, queryLoadSingleProduct, productID)
	if err != nil {
		return nil, fmt.Errorf("merchant: load product %d: %w", productID, err)
	}
	defer rows.Close()
 
	batch, err := scanFeedBatchRows(rows, 1000)
	if err != nil {
		return nil, err
	}
	if len(batch.Items) == 0 {
		return nil, fmt.Errorf("merchant: product %d not found or not eligible", productID)
	}
 
	// Return a MerchantProduct assembled from the flat batch items.
	// The hydration service is the canonical place for this transform;
	// here we provide a convenience wrapper.
	return assembleMerchantProduct(productID, batch.Items), nil
}
 
// ---------------------------------------------------------------------------
// LoadVariant  (single variant hydration)
// ---------------------------------------------------------------------------
 
func (r *MerchantRepository) LoadVariant(
	ctx context.Context,
	variantID int64,
) (*merchant.MerchantVariant, error) {
 
	rows, err := r.db.Query(ctx, queryLoadSingleVariant, variantID)
	if err != nil {
		return nil, fmt.Errorf("merchant: load variant %d: %w", variantID, err)
	}
	defer rows.Close()
 
	batch, err := scanFeedBatchRows(rows, 1)
	if err != nil {
		return nil, err
	}
	if len(batch.Items) == 0 {
		return nil, fmt.Errorf("merchant: variant %d not found or not eligible", variantID)
	}
 
	// Re-constitute a MerchantVariant from the flat feed item.
	return feedItemToVariant(&batch.Items[0]), nil
}
 
// ---------------------------------------------------------------------------
// CountEligibleVariants
// ---------------------------------------------------------------------------
 
func (r *MerchantRepository) CountEligibleVariants(
	ctx context.Context,
	filter merchant.MerchantFeedFilter,
) (int64, error) {
 
	var count int64
	err := r.db.QueryRow(ctx, queryCountEligibleVariants,
		toNullBool(filter.ActiveOnly),
		toNullBool(filter.InStockOnly),
		toNullBool(filter.ExcludeDigital),
		toInt64Array(filter.CategoryIDs),
		toInt64Array(filter.BrandIDs),
		filter.UpdatedAfter,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("merchant: count eligible variants: %w", err)
	}
	return count, nil
}
 
// ---------------------------------------------------------------------------
// LoadUpdatedVariantIDs  (incremental feed support)
// ---------------------------------------------------------------------------
 
func (r *MerchantRepository) LoadUpdatedVariantIDs(
	ctx context.Context,
	since time.Time,
	limit int,
) ([]int64, error) {
 
	rows, err := r.db.Query(ctx, queryLoadUpdatedVariantIDs, since, limit)
	if err != nil {
		return nil, fmt.Errorf("merchant: load updated variant ids: %w", err)
	}
	defer rows.Close()
 
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("merchant: scan updated variant id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
 
// ---------------------------------------------------------------------------
// Row scanner
// ---------------------------------------------------------------------------
 
// scanFeedBatchRows scans the flat rows returned by the feed queries into
// MerchantFeedBatch.  The query always returns one row per variant.
func scanFeedBatchRows(rows pgx.Rows, batchSize int) (*merchant.MerchantFeedBatch, error) {
	items := make([]merchant.MerchantFeedItem, 0, batchSize)
	var lastVariantID int64
 
	for rows.Next() {
		var (
			// Identity
			variantID   int64
			productID   int64
			productSlug string
			sku         string
			gtin        string
			mpn         string
 
			// Core
			title           string
			description     string
			brandName       string
			condition       string
			isDigital       bool
 
			// Category
			categoryPath          string
			googleProductCategory string
 
			// Pricing
			basePrice       float64
			variantPrice    float64
			currency        string
			discountPercent float64
			discountType    string
			discountStartAt *time.Time
			discountEndAt   *time.Time
 
			// Inventory
			availableQty int64
			reservedQty  int64
			incomingQty  int64
 
			// Images (primary + up to 10 additional as JSON array from the DB)
			primaryImageURL    string
			additionalImageURLs []string
 
			// Physical
			weightKg float64
 
			// Variant attributes (as JSON or separate columns)
			attrColor   string
			attrSize    string
			attrGender  string
			attrAgeGroup string
			attrMaterial string
			attrPattern  string
 
			// Timestamps
			updatedAt time.Time
		)
 
		if err := rows.Scan(
			&variantID,
			&productID,
			&productSlug,
			&sku,
			&gtin,
			&mpn,
			&title,
			&description,
			&brandName,
			&condition,
			&isDigital,
			&categoryPath,
			&googleProductCategory,
			&basePrice,
			&variantPrice,
			&currency,
			&discountPercent,
			&discountType,
			&discountStartAt,
			&discountEndAt,
			&availableQty,
			&reservedQty,
			&incomingQty,
			&primaryImageURL,
			&additionalImageURLs,
			&weightKg,
			&attrColor,
			&attrSize,
			&attrGender,
			&attrAgeGroup,
			&attrMaterial,
			&attrPattern,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("merchant: scan feed row: %w", err)
		}
 
		lastVariantID = variantID
 
		// --- Availability ---
		avail := merchant.MerchantAvailability{
			AvailableQty: availableQty,
			ReservedQty:  reservedQty,
			IncomingQty:  incomingQty,
		}
		avail.Status = merchant.ResolveAvailability(availableQty, reservedQty, incomingQty)
 
		// --- Pricing ---
		cur := currency
		if cur == "" {
			cur = "KES"
		}
		regularMoney := merchant.MoneyFromDecimal(variantPrice, cur)
		pricing := merchant.MerchantPricing{
			Price:           regularMoney,
			DiscountPercent: discountPercent,
			DiscountType:    discountType,
		}
		if discountPercent > 0 {
			var saleCents int64
			if discountType == "percentage" {
				saleCents = int64(float64(regularMoney.AmountCents) * (1 - discountPercent/100))
			} else {
				// fixed discount treated as an amount in the store's currency
				saleCents = regularMoney.AmountCents - int64(discountPercent*100)
				if saleCents < 0 {
					saleCents = 0
				}
			}
			pricing.SalePrice = merchant.Money{AmountCents: saleCents, Currency: cur}
			pricing.SalePriceEffectiveStart = discountStartAt
			pricing.SalePriceEffectiveEnd = discountEndAt
		}
 
		// --- Condition ---
		cond := merchant.Condition(condition)
		if !cond.Valid() {
			cond = merchant.ConditionNew
		}
 
		// --- Images ---
		// Ensure HTTPS
		ensureHTTPS := func(u string) string {
			if len(u) > 7 && u[:7] == "http://" {
				return "https://" + u[7:]
			}
			return u
		}
		primaryImageURL = ensureHTTPS(primaryImageURL)
		for i, u := range additionalImageURLs {
			additionalImageURLs[i] = ensureHTTPS(u)
		}
 
		// --- Assemble MerchantFeedItem (flat wire format) ---
		item := merchant.MerchantFeedItem{
			ID:                    fmt.Sprintf("%d-%d", productID, variantID),
			ItemGroupID:           merchant.BuildItemGroupID(productID),
			Title:                 title,
			Description:           description,
			Brand:                 brandName,
			Condition:             string(cond),
			GTIN:                  gtin,
			MPN:                   mpn,
			ImageLink:             primaryImageURL,
			AdditionalImageLinks:  additionalImageURLs,
			GoogleProductCategory: googleProductCategory,
			ProductType:           categoryPath,
			Color:                 attrColor,
			Size:                  attrSize,
			Gender:                attrGender,
			AgeGroup:              attrAgeGroup,
			Material:              attrMaterial,
			Pattern:               attrPattern,
			Availability:          string(avail.Status),
			Price:                 pricing.Price.Format(),
			IdentifierExists:      boolToYesNo(gtin != "" || mpn != ""),
		}
 
		if pricing.HasSalePrice() {
			item.SalePrice = pricing.SalePrice.Format()
			item.SalePriceEffectiveDate = pricing.SalePriceEffectiveDateRange()
		}
 
		if weightKg > 0 {
			item.ShippingWeight = fmt.Sprintf("%.0f g", weightKg*1000)
		}
 
		// Link (will be fully resolved by the hydration service; placeholder here)
		item.Link = fmt.Sprintf("__STORE_URL__/products/%s?variant=%d", productSlug, variantID)
 
		_ = isDigital // available for eligibility filtering in hydration
		_ = basePrice
 
		items = append(items, item)
	}
 
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("merchant: iterate feed rows: %w", err)
	}
 
	return &merchant.MerchantFeedBatch{
		Items:      items,
		NextCursor: lastVariantID,
		HasMore:    len(items) == batchSize,
	}, nil
}
 
// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
 
func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
 
func toNullBool(b bool) *bool { return &b }
 
func toInt64Array(ids []int64) []int64 {
	if ids == nil {
		return []int64{}
	}
	return ids
}
 
// assembleMerchantProduct builds a MerchantProduct shell from flat feed items
// (used only by LoadProduct for backward compatibility with callers that need
// the product-level domain model).
func assembleMerchantProduct(productID int64, items []merchant.MerchantFeedItem) *merchant.MerchantProduct {
	if len(items) == 0 {
		return nil
	}
	first := items[0]
	return &merchant.MerchantProduct{
		ProductID:   productID,
		Name:        first.Title,
		BrandName:   first.Brand,
		CategoryPath: first.ProductType,
		GoogleProductCategory: first.GoogleProductCategory,
	}
}
 
// feedItemToVariant is a thin reconstitution used by LoadVariant.
func feedItemToVariant(item *merchant.MerchantFeedItem) *merchant.MerchantVariant {
	return &merchant.MerchantVariant{
		ID:                    item.ID,
		ItemGroupID:           item.ItemGroupID,
		Title:                 item.Title,
		Description:           item.Description,
		Link:                  item.Link,
		ImageLink:             item.ImageLink,
		AdditionalImageLinks:  item.AdditionalImageLinks,
		Brand:                 item.Brand,
		GTIN:                  item.GTIN,
		MPN:                   item.MPN,
		GoogleProductCategory: item.GoogleProductCategory,
		ProductType:           item.ProductType,
		Color:                 item.Color,
		Size:                  item.Size,
		Material:              item.Material,
		Pattern:               item.Pattern,
		Condition:             merchant.Condition(item.Condition),
	}
}