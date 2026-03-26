package homepage

import "time"

// SectionType represents the display intent of a homepage section.
type SectionType string

const (
	SectionTypeTrending SectionType = "trending"
	SectionTypeFeatured SectionType = "featured"
	SectionTypeCategory SectionType = "category"
	SectionTypeCustom   SectionType = "custom"
)

func (t SectionType) Valid() bool {
	switch t {
	case SectionTypeTrending, SectionTypeFeatured, SectionTypeCategory, SectionTypeCustom:
		return true
	}
	return false
}

// Section maps directly to the homepage_sections table.
type Section struct {
	ID          int64       `json:"id"`
	Title       *string     `json:"title"`
	Type        SectionType `json:"type"`
	ReferenceID *int64      `json:"reference_id"`
	SortOrder   int         `json:"sort_order"`
	IsActive    bool        `json:"is_active"`
}

// SectionWithProducts is the fully resolved section returned to the client.
// Products are populated by the service layer from the catalog repos.
type SectionWithProducts struct {
	Section
	Products []SectionProduct `json:"products"`
}

// SectionProduct is a lightweight product projection used inside a section.
// Keeping it narrow keeps the response small and cache-friendly.
type SectionProduct struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Slug         string   `json:"slug"`
	BasePrice    float64  `json:"base_price"`
	PrimaryImage string   `json:"primary_image"`
	Rating       float64  `json:"rating"`
	ReviewCount  int      `json:"review_count"`
	IsFeatured   bool     `json:"is_featured"`
	BrandName    string   `json:"brand_name,omitempty"`
	CacheKey     string   `json:"-"` // used by caching layer
}

// HomepageResponse is what the public homepage endpoint returns.
type HomepageResponse struct {
	Sections    []SectionWithProducts `json:"sections"`
	GeneratedAt time.Time             `json:"generated_at"`
}