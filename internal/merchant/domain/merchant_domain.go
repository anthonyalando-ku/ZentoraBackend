// Package merchant defines the domain layer for the Google Merchant Center feed system.
// It is catalog-driven and deliberately independent of the discovery/ranking engine.
package merchant

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var (
	ErrNoVariants          = errors.New("merchant: product has no purchasable variants")
	ErrMissingTitle        = errors.New("merchant: item title is required")
	ErrMissingLink         = errors.New("merchant: item link is required")
	ErrMissingImageLink    = errors.New("merchant: item image_link is required")
	ErrMissingPrice        = errors.New("merchant: item price is required")
	ErrInvalidAvailability = errors.New("merchant: invalid availability value")
	ErrInvalidCondition    = errors.New("merchant: invalid condition value")
	ErrTitleTooLong        = errors.New("merchant: title exceeds 150 characters")
	ErrDescriptionTooLong  = errors.New("merchant: description exceeds 5000 characters")
)

// ---------------------------------------------------------------------------
// Enumerations
// ---------------------------------------------------------------------------

// Availability mirrors Google Merchant Center availability values.
type Availability string

const (
	AvailabilityInStock    Availability = "in_stock"
	AvailabilityOutOfStock Availability = "out_of_stock"
	AvailabilityPreorder   Availability = "preorder"
	AvailabilityBackorder  Availability = "backorder"
)

func (a Availability) Valid() bool {
	switch a {
	case AvailabilityInStock, AvailabilityOutOfStock, AvailabilityPreorder, AvailabilityBackorder:
		return true
	}
	return false
}

// Condition mirrors Google Merchant Center condition values.
type Condition string

const (
	ConditionNew         Condition = "new"
	ConditionRefurbished Condition = "refurbished"
	ConditionUsed        Condition = "used"
)

func (c Condition) Valid() bool {
	switch c {
	case ConditionNew, ConditionRefurbished, ConditionUsed:
		return true
	}
	return false
}

// AgeGroup mirrors Google Merchant Center age_group values.
type AgeGroup string

const (
	AgeGroupNewborn AgeGroup = "newborn"
	AgeGroupInfant  AgeGroup = "infant"
	AgeGroupToddler AgeGroup = "toddler"
	AgeGroupKids    AgeGroup = "kids"
	AgeGroupAdult   AgeGroup = "adult"
)

// Gender mirrors Google Merchant Center gender values.
type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
	GenderUnisex Gender = "unisex"
)

// EnergyEfficiencyClass mirrors Google Merchant Center values.
type EnergyEfficiencyClass string

const (
	EnergyClassAPlus3  EnergyEfficiencyClass = "A+++"
	EnergyClassAPlus2  EnergyEfficiencyClass = "A++"
	EnergyClassAPlus   EnergyEfficiencyClass = "A+"
	EnergyClassA       EnergyEfficiencyClass = "A"
	EnergyClassB       EnergyEfficiencyClass = "B"
	EnergyClassC       EnergyEfficiencyClass = "C"
	EnergyClassD       EnergyEfficiencyClass = "D"
	EnergyClassE       EnergyEfficiencyClass = "E"
	EnergyClassF       EnergyEfficiencyClass = "F"
	EnergyClassG       EnergyEfficiencyClass = "G"
)

// PickupMethod mirrors Google Merchant Center pickup_method values.
type PickupMethod string

const (
	PickupBuy      PickupMethod = "buy"
	PickupReserve  PickupMethod = "reserve"
	PickupShipToStore PickupMethod = "ship to store"
	PickupNotSupported PickupMethod = "not supported"
)

// FeedExportMode controls what the feed generator emits.
type FeedExportMode string

const (
	FeedExportFull        FeedExportMode = "full"
	FeedExportIncremental FeedExportMode = "incremental"
)

// ---------------------------------------------------------------------------
// Value objects
// ---------------------------------------------------------------------------

// Money holds a price value and its ISO 4217 currency code.
// Stored as int64 cents to avoid floating-point inaccuracies.
type Money struct {
	AmountCents int64  // e.g. 1999 = 19.99
	Currency    string // e.g. "KES"
}

// Format returns a Google Merchant Center compliant price string: "19.99 KES"
func (m Money) Format() string {
	if m.AmountCents == 0 {
		return fmt.Sprintf("0.00 %s", m.Currency)
	}
	whole := m.AmountCents / 100
	frac := m.AmountCents % 100
	return fmt.Sprintf("%d.%02d %s", whole, frac, m.Currency)
}

func (m Money) IsZero() bool { return m.AmountCents == 0 }

// MoneyFromDecimal converts a DECIMAL(12,2) value (as float64 from pgx) to Money.
func MoneyFromDecimal(amount float64, currency string) Money {
	return Money{
		AmountCents: int64(amount * 100),
		Currency:    currency,
	}
}

// ---------------------------------------------------------------------------
// MerchantImage
// ---------------------------------------------------------------------------

// MerchantImage represents a product image suitable for the Merchant feed.
type MerchantImage struct {
	URL       string
	IsPrimary bool
	SortOrder int
}

// EnsureHTTPS normalises the URL to HTTPS (Google rejects HTTP images).
func (img *MerchantImage) EnsureHTTPS() {
	if strings.HasPrefix(img.URL, "http://") {
		img.URL = "https://" + img.URL[7:]
	}
}

// ---------------------------------------------------------------------------
// MerchantShipping
// ---------------------------------------------------------------------------

// MerchantShipping represents a single shipping option for the feed.
type MerchantShipping struct {
	Country string
	Service string
	PriceAmount Money
}

// Format returns the Google Merchant-compliant shipping annotation.
// e.g. "KE:Standard Shipping:10.00 KES"
func (s MerchantShipping) Format() string {
	return fmt.Sprintf("%s:%s:%s", s.Country, s.Service, s.PriceAmount.Format())
}

// ---------------------------------------------------------------------------
// MerchantTax
// ---------------------------------------------------------------------------

// MerchantTax represents tax configuration for a feed item.
type MerchantTax struct {
	Country  string
	Rate     float64 // decimal, e.g. 0.16 for 16%
	TaxShip  bool
}

// ---------------------------------------------------------------------------
// MerchantPricing
// ---------------------------------------------------------------------------

// MerchantPricing holds the resolved pricing for one feed item.
type MerchantPricing struct {
	Price                    Money
	SalePrice                Money  // zero if no active discount
	SalePriceEffectiveStart  *time.Time
	SalePriceEffectiveEnd    *time.Time
	DiscountPercent          float64 // 0–100
	DiscountType             string  // "percentage" | "fixed"
	CostOfGoodsSold          Money  // optional
}

// HasSalePrice returns true when a genuine sale price is set.
func (p MerchantPricing) HasSalePrice() bool {
	return !p.SalePrice.IsZero() && p.SalePrice.AmountCents < p.Price.AmountCents
}

// SalePriceEffectiveDateRange formats the GMC-required date range string.
// Format: "2024-01-01T00:00:00+00:00/2024-02-01T00:00:00+00:00"
func (p MerchantPricing) SalePriceEffectiveDateRange() string {
	if p.SalePriceEffectiveStart == nil || p.SalePriceEffectiveEnd == nil {
		return ""
	}
	return fmt.Sprintf("%s/%s",
		p.SalePriceEffectiveStart.UTC().Format(time.RFC3339),
		p.SalePriceEffectiveEnd.UTC().Format(time.RFC3339),
	)
}

// ---------------------------------------------------------------------------
// MerchantAvailability
// ---------------------------------------------------------------------------

// MerchantAvailability holds the resolved availability state for one variant.
type MerchantAvailability struct {
	Status          Availability
	AvailableQty    int64
	ReservedQty     int64
	IncomingQty     int64
	AvailabilityDate *time.Time // for preorder
}

// NetQty returns the net sellable quantity.
func (a MerchantAvailability) NetQty() int64 {
	return a.AvailableQty - a.ReservedQty
}

// ResolveAvailability maps inventory state to Google Merchant availability.
// Callers can override with explicit status from product configuration.
func ResolveAvailability(availableQty, reservedQty, incomingQty int64) Availability {
	net := availableQty - reservedQty
	if net > 0 {
		return AvailabilityInStock
	}
	if incomingQty > 0 {
		return AvailabilityPreorder
	}
	return AvailabilityOutOfStock
}

// ---------------------------------------------------------------------------
// MerchantCustomLabels
// ---------------------------------------------------------------------------

// MerchantCustomLabels holds up to 5 custom label slots for campaign segmentation.
type MerchantCustomLabels struct {
	Label0 string // e.g. "sale"
	Label1 string // e.g. "new-arrival"
	Label2 string // e.g. "top-rated"
	Label3 string // e.g. "high-margin"
	Label4 string // e.g. "clearance"
}

// AsSlice returns non-empty labels as an ordered slice.
func (l MerchantCustomLabels) AsSlice() []string {
	var out []string
	for _, v := range []string{l.Label0, l.Label1, l.Label2, l.Label3, l.Label4} {
		out = append(out, v)
	}
	return out
}

// ---------------------------------------------------------------------------
// MerchantVariant  (core unit)
// ---------------------------------------------------------------------------

// MerchantVariant is the primary unit of a Merchant feed entry.
// Each purchasable variant becomes one <item> in the XML feed.
type MerchantVariant struct {
	// --- Identity ---
	ID          string // feed item id: "<productID>-<variantID>"
	ItemGroupID string // parent grouping: "product-<productID>"
	SKU         string
	GTIN        string // barcode
	MPN         string // manufacturer part number

	// --- Product context ---
	ProductID   int64
	VariantID   int64
	ProductSlug string
	VariantName string // e.g. "Blue / XL"

	// --- Core required fields ---
	Title       string
	Description string
	Link        string
	MobileLink  string

	// --- Images ---
	ImageLink            string
	AdditionalImageLinks []string

	// --- Availability & pricing ---
	Availability MerchantAvailability
	Pricing      MerchantPricing
	Condition    Condition

	// --- Classification ---
	Brand               string
	GoogleProductCategory string // GMC taxonomy ID or path
	ProductType         string   // store's own category path
	CategoryPath        string   // full breadcrumb path

	// --- Variant dimensions ---
	Color    string
	Size     string
	Gender   Gender
	AgeGroup AgeGroup
	Material string
	Pattern  string

	// --- Physical ---
	ShippingWeightGrams float64
	Multipack           int // number of identical items in one pack

	// --- Merchant-specific ---
	Shipping    []MerchantShipping
	Tax         []MerchantTax
	CustomLabels MerchantCustomLabels

	// --- Optional ---
	AdultOnly             bool
	IdentifierExists      bool
	ExpirationDate        *time.Time
	EnergyEfficiencyClass EnergyEfficiencyClass
	UnitPricingMeasure    string // e.g. "100ml"
	UnitPricingBaseMeasure string // e.g. "100ml"
	AdsRedirect           string
	PickupMethod          PickupMethod
	PickupSLA             string

	// --- Feed metadata ---
	UpdatedAt time.Time
}

// FeedItemID formats the canonical Merchant Center item id.
func (v *MerchantVariant) FeedItemID() string {
	return fmt.Sprintf("%d-%d", v.ProductID, v.VariantID)
}

// Validate runs Google Merchant Center validation on required fields.
func (v *MerchantVariant) Validate() error {
	if strings.TrimSpace(v.Title) == "" {
		return ErrMissingTitle
	}
	if utf8.RuneCountInString(v.Title) > 150 {
		return ErrTitleTooLong
	}
	if utf8.RuneCountInString(v.Description) > 5000 {
		return ErrDescriptionTooLong
	}
	if strings.TrimSpace(v.Link) == "" {
		return ErrMissingLink
	}
	if strings.TrimSpace(v.ImageLink) == "" {
		return ErrMissingImageLink
	}
	if v.Pricing.Price.IsZero() {
		return ErrMissingPrice
	}
	if !v.Availability.Status.Valid() {
		return ErrInvalidAvailability
	}
	if !v.Condition.Valid() {
		return ErrInvalidCondition
	}
	return nil
}

// NormalizeImages ensures all image URLs are HTTPS and deduplicated.
func (v *MerchantVariant) NormalizeImages() {
	if strings.HasPrefix(v.ImageLink, "http://") {
		v.ImageLink = "https://" + v.ImageLink[7:]
	}
	seen := map[string]struct{}{v.ImageLink: {}}
	deduped := v.AdditionalImageLinks[:0]
	for _, u := range v.AdditionalImageLinks {
		if strings.HasPrefix(u, "http://") {
			u = "https://" + u[7:]
		}
		if _, exists := seen[u]; !exists {
			seen[u] = struct{}{}
			deduped = append(deduped, u)
		}
	}
	// GMC allows up to 10 additional images
	if len(deduped) > 10 {
		deduped = deduped[:10]
	}
	v.AdditionalImageLinks = deduped
}

// NormalizeTitle truncates and cleans the title for GMC compliance.
func (v *MerchantVariant) NormalizeTitle() {
	v.Title = strings.TrimSpace(v.Title)
	runes := []rune(v.Title)
	if len(runes) > 150 {
		v.Title = string(runes[:150])
	}
}

// NormalizeDescription truncates the description for GMC compliance.
func (v *MerchantVariant) NormalizeDescription() {
	v.Description = strings.TrimSpace(v.Description)
	runes := []rune(v.Description)
	if len(runes) > 5000 {
		v.Description = string(runes[:5000])
	}
}

// ---------------------------------------------------------------------------
// MerchantProduct  (product-level view, container for variants)
// ---------------------------------------------------------------------------

// MerchantProduct is an intermediate hydration model that holds all
// product-level data before it is exploded into per-variant feed items.
// It is NOT exported directly to the XML feed.
type MerchantProduct struct {
	ProductID   int64
	Name        string
	Slug        string
	Description string
	BrandName   string
	BrandSlug   string
	BasePrice   float64
	Currency    string
	Condition   Condition
	IsDigital   bool
	Rating      float64
	ReviewCount int

	// Category hydration
	PrimaryCategoryID   int64
	PrimaryCategoryName string
	CategoryPath        string
	GoogleProductCategory string

	// Images
	Images []MerchantImage

	// Variants (populated by repository)
	Variants []MerchantProductVariant

	UpdatedAt time.Time
}

// MerchantProductVariant is the raw variant data joined from the DB.
// It is transformed into MerchantVariant during hydration.
type MerchantProductVariant struct {
	VariantID   int64
	SKU         string
	Price       float64
	WeightGrams float64
	IsActive    bool

	// Inventory (aggregated across locations)
	AvailableQty int64
	ReservedQty  int64
	IncomingQty  int64

	// Discount resolved at variant level
	DiscountPercent         float64
	DiscountType            string
	DiscountStartsAt        *time.Time
	DiscountEndsAt          *time.Time

	// Attributes as key→value map (e.g. "color"→"Blue", "size"→"XL")
	Attributes map[string]string

	// Optional overrides
	GTIN string
	MPN  string
}

// ---------------------------------------------------------------------------
// MerchantFeedItem  (final, XML-ready unit)
// ---------------------------------------------------------------------------

// MerchantFeedItem is the wire-format DTO consumed by the XML generator.
// It is a flat representation of MerchantVariant with serialization helpers.
type MerchantFeedItem struct {
	// Required
	ID           string
	Title        string
	Description  string
	Link         string
	ImageLink    string
	Availability string
	Price        string // formatted: "19.99 KES"
	Condition    string
	Brand        string

	// Strongly recommended
	GTIN                       string
	MPN                        string
	ItemGroupID                string
	SalePrice                  string // "" if no sale
	SalePriceEffectiveDate     string
	AdditionalImageLinks       []string
	GoogleProductCategory      string
	ProductType                string
	Color                      string
	Size                       string
	Gender                     string
	AgeGroup                   string
	Material                   string
	Pattern                    string
	ShippingWeight             string // "250 g"
	Shipping                   []string
	Tax                        []MerchantTax
	IdentifierExists           string // "yes" | "no"
	CustomLabel0               string
	CustomLabel1               string
	CustomLabel2               string
	CustomLabel3               string
	CustomLabel4               string

	// Optional
	MobileLink             string
	EnergyEfficiencyClass  string
	UnitPricingMeasure     string
	UnitPricingBaseMeasure string
	Multipack              int
	Adult                  string // "yes" | "no"
	ExpirationDate         string
	AvailabilityDate       string
	CostOfGoodsSold        string
	AdsRedirect            string
	PickupMethod           string
	PickupSLA              string
}

// FromVariant converts a hydrated MerchantVariant into a MerchantFeedItem.
func FeedItemFromVariant(v *MerchantVariant) MerchantFeedItem {
	item := MerchantFeedItem{
		ID:                    v.FeedItemID(),
		Title:                 v.Title,
		Description:           v.Description,
		Link:                  v.Link,
		ImageLink:             v.ImageLink,
		AdditionalImageLinks:  v.AdditionalImageLinks,
		Availability:          string(v.Availability.Status),
		Price:                 v.Pricing.Price.Format(),
		Condition:             string(v.Condition),
		Brand:                 v.Brand,
		GTIN:                  v.GTIN,
		MPN:                   v.MPN,
		ItemGroupID:           v.ItemGroupID,
		GoogleProductCategory: v.GoogleProductCategory,
		ProductType:           v.ProductType,
		Color:                 v.Color,
		Size:                  v.Size,
		Gender:                string(v.Gender),
		AgeGroup:              string(v.AgeGroup),
		Material:              v.Material,
		Pattern:               v.Pattern,
		MobileLink:            v.MobileLink,
		EnergyEfficiencyClass: string(v.EnergyEfficiencyClass),
		UnitPricingMeasure:    v.UnitPricingMeasure,
		UnitPricingBaseMeasure: v.UnitPricingBaseMeasure,
		Multipack:             v.Multipack,
		AdsRedirect:           v.AdsRedirect,
		PickupMethod:          string(v.PickupMethod),
		PickupSLA:             v.PickupSLA,
		CustomLabel0:          v.CustomLabels.Label0,
		CustomLabel1:          v.CustomLabels.Label1,
		CustomLabel2:          v.CustomLabels.Label2,
		CustomLabel3:          v.CustomLabels.Label3,
		CustomLabel4:          v.CustomLabels.Label4,
	}

	if v.Pricing.HasSalePrice() {
		item.SalePrice = v.Pricing.SalePrice.Format()
		item.SalePriceEffectiveDate = v.Pricing.SalePriceEffectiveDateRange()
	}

	if !v.Pricing.CostOfGoodsSold.IsZero() {
		item.CostOfGoodsSold = v.Pricing.CostOfGoodsSold.Format()
	}

	if v.ShippingWeightGrams > 0 {
		item.ShippingWeight = fmt.Sprintf("%.0f g", v.ShippingWeightGrams)
	}

	for _, s := range v.Shipping {
		item.Shipping = append(item.Shipping, s.Format())
	}
	item.Tax = v.Tax

	item.IdentifierExists = "yes"
	if v.GTIN == "" && v.MPN == "" {
		item.IdentifierExists = "no"
	}

	if v.AdultOnly {
		item.Adult = "yes"
	} else {
		item.Adult = "no"
	}

	if v.ExpirationDate != nil {
		item.ExpirationDate = v.ExpirationDate.UTC().Format("2006-01-02")
	}

	if v.Availability.AvailabilityDate != nil {
		item.AvailabilityDate = v.Availability.AvailabilityDate.UTC().Format(time.RFC3339)
	}

	return item
}

// ---------------------------------------------------------------------------
// MerchantFeedBatch  (pagination / batching)
// ---------------------------------------------------------------------------

// MerchantFeedBatch is the result of a single page fetch from the repository.
type MerchantFeedBatch struct {
	Items      []MerchantFeedItem
	TotalCount int64  // total eligible items (for progress tracking)
	NextCursor int64  // last variant_id in this batch (for keyset pagination)
	HasMore    bool
}

// ---------------------------------------------------------------------------
// MerchantFeedMetadata
// ---------------------------------------------------------------------------

// MerchantFeedMetadata records housekeeping state for each feed generation run.
type MerchantFeedMetadata struct {
	FeedID          string
	GeneratedAt     time.Time
	ExportMode      FeedExportMode
	TotalItems      int64
	ValidItems      int64
	InvalidItems    int64
	DurationSeconds float64
	SchemaVersion   string
}

// ---------------------------------------------------------------------------
// Feed request / filter
// ---------------------------------------------------------------------------

// MerchantFeedFilter constrains which products appear in the feed.
type MerchantFeedFilter struct {
	// Eligibility
	InStockOnly   bool
	ActiveOnly    bool // always true in production
	ExcludeDigital bool

	// Scope (nil = no restriction)
	CategoryIDs []int64
	BrandIDs    []int64

	// Incremental mode
	UpdatedAfter *time.Time

	// Pagination
	Cursor int64 // last seen variant_id (keyset)
	Limit  int   // batch size, e.g. 500
}

// DefaultFeedFilter returns safe production defaults.
func DefaultFeedFilter() MerchantFeedFilter {
	limit := 500
	return MerchantFeedFilter{
		InStockOnly:    false,
		ActiveOnly:     true,
		ExcludeDigital: false,
		Limit:          limit,
	}
}

// ---------------------------------------------------------------------------
// URL builder (SEO)
// ---------------------------------------------------------------------------

// BuildProductURL returns the canonical HTTPS product URL.
// baseURL example: "https://zentora.com"
func BuildProductURL(baseURL, slug string) string {
	return fmt.Sprintf("%s/products/%s", strings.TrimRight(baseURL, "/"), slug)
}

// BuildVariantURL appends a variant query param to the canonical URL.
func BuildVariantURL(baseURL, slug string, variantID int64) string {
	return fmt.Sprintf("%s/products/%s?variant=%d", strings.TrimRight(baseURL, "/"), slug, variantID)
}

// BuildItemGroupID formats the item_group_id for a product family.
func BuildItemGroupID(productID int64) string {
	return fmt.Sprintf("product-%d", productID)
}