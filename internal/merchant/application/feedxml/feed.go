// Package feedxml defines the XML-serializable structs for Google Merchant
// Center RSS 2.0 / Atom feed format.
//
// These types are intentionally decoupled from the domain models to allow the
// serialization layer to evolve independently (e.g. future Facebook/TikTok
// catalogue feeds that use different XML namespaces).
//
// Usage:
//
//	feed := feedxml.BuildRSSFeed(metadata, items, "https://zentora.com")
//	enc := xml.NewEncoder(w)
//	enc.Indent("", "  ")
//	enc.Encode(feed)
package feedxml

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"zentora-service/internal/merchant/domain"
)

// ---------------------------------------------------------------------------
// RSS 2.0 root structures
// ---------------------------------------------------------------------------

// RSSFeed is the top-level Google Merchant Center XML feed document.
type RSSFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	GNS     string     `xml:"xmlns:g,attr"`
	Channel RSSChannel `xml:"channel"`
}
 
// RSSChannel maps to the <channel> element.
type RSSChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []RSSItem `xml:"item"`
}
 
// RSSItem is one <item> element — one per purchasable variant.
//
// Required fields are value types (always emitted).
// Every optional/recommended field is *string so a nil pointer produces no
// XML element at all. This is the only approach that works reliably with
// encoding/xml — struct-based omitempty does not suppress empty structs
// the same way pointer-based omitempty does.
type RSSItem struct {
	// ---- Required (always present) -----------------------------------------
	ID           string `xml:"g:id"`
	Title        string `xml:"title"`
	Description  string `xml:"description"`
	Link         string `xml:"link"`
	ImageLink    string `xml:"g:image_link"`
	Availability string `xml:"g:availability"`
	Price        string `xml:"g:price"`
	Condition    string `xml:"g:condition"`
	Brand        string `xml:"g:brand"`
 
	// ---- Strongly recommended (omitted when nil) ----------------------------
	GTIN                   *string  `xml:"g:gtin"`
	MPN                    *string  `xml:"g:mpn"`
	ItemGroupID            *string  `xml:"g:item_group_id"`
	SalePrice              *string  `xml:"g:sale_price"`
	SalePriceEffectiveDate *string  `xml:"g:sale_price_effective_date"`
	GoogleProductCategory  *string  `xml:"g:google_product_category"`
	ProductType            *string  `xml:"g:product_type"`
	Color                  *string  `xml:"g:color"`
	Size                   *string  `xml:"g:size"`
	Gender                 *string  `xml:"g:gender"`
	AgeGroup               *string  `xml:"g:age_group"`
	Material               *string  `xml:"g:material"`
	Pattern                *string  `xml:"g:pattern"`
	ShippingWeight         *string  `xml:"g:shipping_weight"`
	IdentifierExists       *string  `xml:"g:identifier_exists"`
	AdditionalImageLinks   []string `xml:"g:additional_image_link"`
	Shippings              []RSSShipping `xml:"g:shipping"`
 
	// Custom labels — only emitted when non-empty
	CustomLabel0 *string `xml:"g:custom_label_0"`
	CustomLabel1 *string `xml:"g:custom_label_1"`
	CustomLabel2 *string `xml:"g:custom_label_2"`
	CustomLabel3 *string `xml:"g:custom_label_3"`
	CustomLabel4 *string `xml:"g:custom_label_4"`
 
	// ---- Optional (omitted when nil) ----------------------------------------
	MobileLink             *string `xml:"g:mobile_link"`
	EnergyEfficiencyClass  *string `xml:"g:energy_efficiency_class"`
	UnitPricingMeasure     *string `xml:"g:unit_pricing_measure"`
	UnitPricingBaseMeasure *string `xml:"g:unit_pricing_base_measure"`
	Multipack              *int    `xml:"g:multipack"`
	Adult                  *string `xml:"g:adult"`
	ExpirationDate         *string `xml:"g:expiration_date"`
	AvailabilityDate       *string `xml:"g:availability_date"`
	CostOfGoodsSold        *string `xml:"g:cost_of_goods_sold"`
	AdsRedirect            *string `xml:"g:ads_redirect"`
	PickupMethod           *string `xml:"g:pickup_method"`
	PickupSLA              *string `xml:"g:pickup_sla"`
}
 
// RSSShipping represents the structured <g:shipping> element.
type RSSShipping struct {
	Country string `xml:"g:country"`
	Service string `xml:"g:service"`
	Price   string `xml:"g:price"`
}
 
// ---------------------------------------------------------------------------
// Builder
// ---------------------------------------------------------------------------
 
// BuildRSSFeed converts a slice of normalised MerchantFeedItems into a
// serialisable RSSFeed. normaliseItem in feedgen must have run first.
func BuildRSSFeed(
	meta merchant.MerchantFeedMetadata,
	items []merchant.MerchantFeedItem,
	storeURL, storeTitle string,
) *RSSFeed {
	rssItems := make([]RSSItem, 0, len(items))
	for i := range items {
		rssItems = append(rssItems, toRSSItem(&items[i]))
	}
 
	return &RSSFeed{
		Version: "2.0",
		GNS:     "http://base.google.com/ns/1.0",
		Channel: RSSChannel{
			Title:       storeTitle,
			Link:        storeURL,
			Description: fmt.Sprintf("Google Merchant Center feed – generated %s", meta.GeneratedAt.Format(time.RFC3339)),
			Items:       rssItems,
		},
	}
}
 
// toRSSItem converts one MerchantFeedItem to an RSSItem.
// A field is only populated (non-nil pointer) when it carries real data.
func toRSSItem(item *merchant.MerchantFeedItem) RSSItem {
	ri := RSSItem{
		// Required — always set.
		ID:           item.ID,
		Title:        item.Title,
		Description:  item.Description,
		Link:         item.Link,
		ImageLink:    item.ImageLink,
		Availability: item.Availability,
		Price:        item.Price,
		Condition:    item.Condition,
		Brand:        item.Brand,
	}
 
	// Strongly recommended — only when non-empty.
	ri.GTIN                   = ps(item.GTIN)
	ri.MPN                    = ps(item.MPN)
	ri.ItemGroupID             = ps(item.ItemGroupID)
	ri.SalePrice               = ps(item.SalePrice)
	ri.SalePriceEffectiveDate  = ps(item.SalePriceEffectiveDate)
	ri.GoogleProductCategory   = ps(item.GoogleProductCategory)
	ri.ProductType             = ps(item.ProductType)
	ri.Color                   = ps(item.Color)
	ri.Size                    = ps(item.Size)
	ri.Gender                  = ps(item.Gender)
	ri.AgeGroup                = ps(item.AgeGroup)
	ri.Material                = ps(item.Material)
	ri.Pattern                 = ps(item.Pattern)
	ri.ShippingWeight          = ps(item.ShippingWeight)
	ri.IdentifierExists        = ps(item.IdentifierExists)
 
	// Custom labels — only when the label has content.
	ri.CustomLabel0 = ps(item.CustomLabel0)
	ri.CustomLabel1 = ps(item.CustomLabel1)
	ri.CustomLabel2 = ps(item.CustomLabel2)
	ri.CustomLabel3 = ps(item.CustomLabel3)
	ri.CustomLabel4 = ps(item.CustomLabel4)
 
	// Optional — only when non-empty.
	ri.MobileLink             = ps(item.MobileLink)
	ri.EnergyEfficiencyClass  = ps(item.EnergyEfficiencyClass)
	ri.UnitPricingMeasure     = ps(item.UnitPricingMeasure)
	ri.UnitPricingBaseMeasure = ps(item.UnitPricingBaseMeasure)
	ri.Adult                  = ps(item.Adult)
	ri.ExpirationDate         = ps(item.ExpirationDate)
	ri.AvailabilityDate       = ps(item.AvailabilityDate)
	ri.CostOfGoodsSold        = ps(item.CostOfGoodsSold)
	ri.AdsRedirect            = ps(item.AdsRedirect)
	ri.PickupMethod           = ps(item.PickupMethod)
	ri.PickupSLA              = ps(item.PickupSLA)
 
	// Multipack — only for genuine multi-unit packs (value > 1).
	if item.Multipack > 1 {
		n := item.Multipack
		ri.Multipack = &n
	}
 
	// Additional images — already deduplicated and validated by normaliseItem.
	ri.AdditionalImageLinks = item.AdditionalImageLinks
 
	// Shipping — only when entries exist.
	for _, s := range item.Shipping {
		if parsed := parseShippingString(s); parsed.Country != "" {
			ri.Shippings = append(ri.Shippings, parsed)
		}
	}
 
	return ri
}
 
// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
 
// ps returns a pointer to s if s is non-empty, otherwise nil.
// A nil pointer field is entirely omitted by encoding/xml.
func ps(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
 
// parseShippingString parses "KE:Standard Shipping:10.00 KES" into RSSShipping.
func parseShippingString(s string) RSSShipping {
	// We split on ":" with a limit of 3 so the price "10.00 KES" is kept intact.
	parts := splitN(s, ":", 3)
	if len(parts) != 3 {
		return RSSShipping{}
	}
	return RSSShipping{
		Country: parts[0],
		Service: parts[1],
		Price:   parts[2],
	}
}
 
func splitN(s, sep string, n int) []string {
	var out []string
	for len(out) < n-1 {
		i := strings.Index(s, sep)
		if i < 0 {
			break
		}
		out = append(out, s[:i])
		s = s[i+len(sep):]
	}
	out = append(out, s)
	return out
}