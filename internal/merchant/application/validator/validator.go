// Package validator provides production-grade validation for merchant feed items.
// It implements all Google Merchant Center requirements as of 2024 and is
// designed to be called before any item enters the XML serialiser.
package validator

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"zentora-service/internal/merchant/domain"
)

// ValidationResult holds all validation errors for a single feed item.
type ValidationResult struct {
	ItemID string
	Errors []ValidationError
}

// ValidationError represents a single field-level error.
type ValidationError struct {
	Field   string
	Code    string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Field, e.Message)
}

// IsValid returns true if there are no validation errors.
func (r *ValidationResult) IsValid() bool { return len(r.Errors) == 0 }

// Add appends a validation error.
func (r *ValidationResult) Add(field, code, msg string) {
	r.Errors = append(r.Errors, ValidationError{Field: field, Code: code, Message: msg})
}

// ---------------------------------------------------------------------------
// Validator
// ---------------------------------------------------------------------------

// Validator runs all GMC field checks.
type Validator struct {
	requireGTINOrMPN bool // stricter mode for branded products
}

// New returns a Validator with safe defaults.
func New(requireGTINOrMPN bool) *Validator {
	return &Validator{requireGTINOrMPN: requireGTINOrMPN}
}

var (
	gtinRegexp  = regexp.MustCompile(`^\d{8}$|^\d{12}$|^\d{13}$|^\d{14}$`)
	priceRegexp = regexp.MustCompile(`^\d+\.\d{2} [A-Z]{3}$`)
)

// Validate runs all checks on a MerchantFeedItem.
func (v *Validator) Validate(item *merchant.MerchantFeedItem) *ValidationResult {
	result := &ValidationResult{ItemID: item.ID}

	v.checkRequired(item, result)
	v.checkRecommended(item, result)
	v.checkOptional(item, result)

	return result
}

func (v *Validator) checkRequired(item *merchant.MerchantFeedItem, r *ValidationResult) {
	// id
	if item.ID == "" {
		r.Add("id", "MISSING_REQUIRED", "id is required")
	} else if utf8.RuneCountInString(item.ID) > 50 {
		r.Add("id", "TOO_LONG", "id must be ≤ 50 characters")
	}

	// title
	switch {
	case strings.TrimSpace(item.Title) == "":
		r.Add("title", "MISSING_REQUIRED", "title is required")
	case utf8.RuneCountInString(item.Title) > 150:
		r.Add("title", "TOO_LONG", "title must be ≤ 150 characters")
	}

	// description
	if utf8.RuneCountInString(item.Description) > 5000 {
		r.Add("description", "TOO_LONG", "description must be ≤ 5000 characters")
	}

	// link
	if !isValidHTTPSURL(item.Link) {
		r.Add("link", "INVALID_URL", "link must be a valid HTTPS URL")
	}

	// image_link
	if !isValidHTTPSURL(item.ImageLink) {
		r.Add("image_link", "INVALID_URL", "image_link must be a valid HTTPS URL")
	}

	// availability
	if !merchant.Availability(item.Availability).Valid() {
		r.Add("availability", "INVALID_VALUE",
			fmt.Sprintf("availability must be one of: in_stock, out_of_stock, preorder, backorder (got %q)", item.Availability))
	}

	// price
	if !priceRegexp.MatchString(item.Price) {
		r.Add("price", "INVALID_FORMAT", fmt.Sprintf("price must be in '0.00 CUR' format (got %q)", item.Price))
	}

	// condition
	if !merchant.Condition(item.Condition).Valid() {
		r.Add("condition", "INVALID_VALUE",
			fmt.Sprintf("condition must be one of: new, refurbished, used (got %q)", item.Condition))
	}

	// brand
	if strings.TrimSpace(item.Brand) == "" {
		r.Add("brand", "MISSING_REQUIRED", "brand is required")
	} else if utf8.RuneCountInString(item.Brand) > 70 {
		r.Add("brand", "TOO_LONG", "brand must be ≤ 70 characters")
	}
}

func (v *Validator) checkRecommended(item *merchant.MerchantFeedItem, r *ValidationResult) {
	// gtin — format validation only; existence is optional
	if item.GTIN != "" && !gtinRegexp.MatchString(item.GTIN) {
		r.Add("gtin", "INVALID_FORMAT", "gtin must be 8, 12, 13, or 14 digits")
	}

	// item_group_id — required when variants exist
	if item.ItemGroupID == "" {
		r.Add("item_group_id", "MISSING_RECOMMENDED", "item_group_id is required for variant products")
	}

	// sale_price — must have same currency as price
	if item.SalePrice != "" {
		if !priceRegexp.MatchString(item.SalePrice) {
			r.Add("sale_price", "INVALID_FORMAT", "sale_price format must match price format")
		} else {
			priceCur := extractCurrency(item.Price)
			saleCur := extractCurrency(item.SalePrice)
			if priceCur != saleCur {
				r.Add("sale_price", "CURRENCY_MISMATCH", "sale_price currency must match price currency")
			}
		}
	}

	// additional_image_link — max 10 and all HTTPS
	if len(item.AdditionalImageLinks) > 10 {
		r.Add("additional_image_link", "TOO_MANY", "maximum 10 additional_image_links")
	}
	for i, u := range item.AdditionalImageLinks {
		if !isValidHTTPSURL(u) {
			r.Add("additional_image_link",
				"INVALID_URL",
				fmt.Sprintf("additional_image_link[%d] must be a valid HTTPS URL", i))
		}
	}

	// identifier_exists
	if item.IdentifierExists != "yes" && item.IdentifierExists != "no" {
		r.Add("identifier_exists", "INVALID_VALUE", "identifier_exists must be 'yes' or 'no'")
	}

	// Strict mode: brand + no GTIN/MPN requires identifier_exists=no
	if v.requireGTINOrMPN && item.Brand != "" &&
		item.GTIN == "" && item.MPN == "" &&
		item.IdentifierExists != "no" {
		r.Add("identifier_exists", "MISSING_IDENTIFIER",
			"products with a brand must supply gtin or mpn, or set identifier_exists=no")
	}

	// custom_label length
	for i, label := range []string{
		item.CustomLabel0, item.CustomLabel1,
		item.CustomLabel2, item.CustomLabel3,
		item.CustomLabel4,
	} {
		if utf8.RuneCountInString(label) > 100 {
			r.Add(fmt.Sprintf("custom_label_%d", i), "TOO_LONG",
				"custom_label must be ≤ 100 characters")
		}
	}
}

func (v *Validator) checkOptional(item *merchant.MerchantFeedItem, r *ValidationResult) {
	// mobile_link
	if item.MobileLink != "" && !isValidHTTPSURL(item.MobileLink) {
		r.Add("mobile_link", "INVALID_URL", "mobile_link must be a valid HTTPS URL")
	}

	// ads_redirect
	if item.AdsRedirect != "" && !isValidHTTPSURL(item.AdsRedirect) {
		r.Add("ads_redirect", "INVALID_URL", "ads_redirect must be a valid HTTPS URL")
	}

	// multipack
	if item.Multipack < 0 {
		r.Add("multipack", "INVALID_VALUE", "multipack must be ≥ 0")
	}

	// adult
	if item.Adult != "" && item.Adult != "yes" && item.Adult != "no" {
		r.Add("adult", "INVALID_VALUE", "adult must be 'yes' or 'no'")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func isValidHTTPSURL(raw string) bool {
	if raw == "" {
		return false
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}
	return u.Scheme == "https" && u.Host != ""
}

func extractCurrency(priceStr string) string {
	parts := strings.Fields(priceStr)
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

// ValidateBatch validates a slice of feed items and returns a summary.
type BatchValidationSummary struct {
	Total   int
	Valid   int
	Invalid int
	Results []ValidationResult
}

func ValidateBatch(items []merchant.MerchantFeedItem, v *Validator) BatchValidationSummary {
	summary := BatchValidationSummary{Total: len(items)}
	for i := range items {
		res := v.Validate(&items[i])
		if res.IsValid() {
			summary.Valid++
		} else {
			summary.Invalid++
			summary.Results = append(summary.Results, *res)
		}
	}
	return summary
}