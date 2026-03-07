package discovery

import "strings"

type FeedType string

const (
	FeedTrending       FeedType = "trending"
	FeedBestSellers    FeedType = "best_sellers"
	FeedRecommended    FeedType = "recommended"
	FeedCategory       FeedType = "category"
	FeedDeals          FeedType = "deals"
	FeedNewArrivals    FeedType = "new_arrivals"
	FeedHighlyRated    FeedType = "highly_rated"
	FeedMostWishlisted FeedType = "most_wishlisted"
	FeedAlsoViewed     FeedType = "also_viewed"
	FeedFeatured       FeedType = "featured"
	FeedEditorial      FeedType = "editorial"
	FeedSearch         FeedType = "search"
)

const (
	DefaultFeedLimit    = 20
	MaxFeedLimit        = 100
	DefaultSuggestLimit = 10
	MaxSuggestLimit     = 20
)

type FeedFilter struct {
	BrandIDs     []int64
	TagIDs       []int64
	PriceMin     *float64
	PriceMax     *float64
	MinRating    *float64
	DiscountOnly bool
	InStockOnly  bool
}

type FeedRequest struct {
	FeedType   FeedType
	UserID     *int64
	SessionID  *string
	CategoryID *int64
	Query      *string
	Limit      int
	Filters    FeedFilter
}

func (r *FeedRequest) Validate() error {
	if r == nil {
		return ErrInvalidRequest
	}

	r.FeedType = FeedType(strings.TrimSpace(string(r.FeedType)))
	switch r.FeedType {
	case FeedTrending, FeedBestSellers, FeedRecommended, FeedCategory, FeedDeals,
		FeedNewArrivals, FeedHighlyRated, FeedMostWishlisted, FeedAlsoViewed,
		FeedFeatured, FeedEditorial, FeedSearch:
	default:
		return ErrInvalidFeedType
	}

	if r.Limit <= 0 {
		r.Limit = DefaultFeedLimit
	}
	if r.Limit > MaxFeedLimit {
		r.Limit = MaxFeedLimit
	}

	if r.UserID != nil && *r.UserID <= 0 {
		return ErrInvalidUserID
	}

	if r.SessionID != nil {
		trimmed := strings.TrimSpace(*r.SessionID)
		if trimmed == "" {
			r.SessionID = nil
		} else if trimmed != *r.SessionID {
			r.SessionID = &trimmed
		}
	}

	if r.CategoryID != nil && *r.CategoryID <= 0 {
		return ErrInvalidCategoryID
	}
	if r.FeedType == FeedCategory && r.CategoryID == nil {
		return ErrCategoryRequired
	}

	if r.Query != nil {
		trimmed := strings.TrimSpace(*r.Query)
		if trimmed != *r.Query {
			r.Query = &trimmed
		}
	}
	if r.FeedType == FeedSearch && (r.Query == nil || *r.Query == "") {
		return ErrQueryRequired
	}
	if r.FeedType == FeedRecommended && r.UserID == nil {
		return ErrUserRequired
	}
	if r.FeedType == FeedAlsoViewed && r.UserID == nil && r.SessionID == nil {
		return ErrUserOrSessionRequired
	}

	return nil
}

type Candidate struct {
	ProductID int64
	Signals   map[string]float64
}

type InventoryStatus string

const (
	InventoryStatusInStock    InventoryStatus = "in_stock"
	InventoryStatusLowStock   InventoryStatus = "low_stock"
	InventoryStatusOutOfStock InventoryStatus = "out_of_stock"
)

type ProductCard struct {
	ProductID       int64           `json:"product_id"`
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	PrimaryImage    string          `json:"primary_image"`
	Price           float64         `json:"price"`
	Discount        float64         `json:"discount"`
	Rating          float64         `json:"rating"`
	ReviewCount     int             `json:"review_count"`
	InventoryStatus InventoryStatus `json:"inventory_status"`
	Brand           string          `json:"brand"`
	Category        string          `json:"category"`
}

type SuggestionType string

const (
	SuggestionTypeProduct  SuggestionType = "product"
	SuggestionTypeCategory SuggestionType = "category"
	SuggestionTypeBrand    SuggestionType = "brand"
	SuggestionTypeQuery    SuggestionType = "query"
)

type SuggestRequest struct {
	Prefix string
	Limit  int
}

func (r *SuggestRequest) Validate() error {
	if r == nil {
		return ErrInvalidRequest
	}

	r.Prefix = strings.TrimSpace(r.Prefix)
	if r.Prefix == "" {
		return ErrPrefixRequired
	}

	if r.Limit <= 0 {
		r.Limit = DefaultSuggestLimit
	}
	if r.Limit > MaxSuggestLimit {
		r.Limit = MaxSuggestLimit
	}

	return nil
}

type Suggestion struct {
	Text            string
	Type            SuggestionType
	ReferenceID     *int64
	PopularityScore float64
}

type SearchResultPosition struct {
	ProductID int64
	Position  int
	Score     float64
}

type SearchEvent struct {
	Query           string
	NormalizedQuery string
	UserID          *int64
	SessionID       *string
	ResultCount     int
	Results         []SearchResultPosition
}

func (e *SearchEvent) Validate() error {
	if e == nil {
		return ErrInvalidRequest
	}

	e.Query = strings.TrimSpace(e.Query)
	if e.Query == "" {
		return ErrQueryRequired
	}
	e.NormalizedQuery = strings.ToLower(e.Query)

	if e.UserID != nil && *e.UserID <= 0 {
		return ErrInvalidUserID
	}

	if e.SessionID != nil {
		trimmed := strings.TrimSpace(*e.SessionID)
		if trimmed == "" {
			e.SessionID = nil
		} else {
			e.SessionID = &trimmed
		}
	}

	if e.ResultCount < 0 {
		return ErrNegativeResultCount
	}

	for _, result := range e.Results {
		if result.ProductID <= 0 {
			return ErrInvalidProductID
		}
		if result.Position <= 0 {
			return ErrInvalidPosition
		}
	}

	return nil
}

type SearchClickEvent struct {
	SearchEventID int64
	ProductID     int64
	Position      int
	UserID        *int64
	SessionID     *string
}

func (e *SearchClickEvent) Validate() error {
	if e == nil {
		return ErrInvalidRequest
	}

	if e.SearchEventID <= 0 {
		return ErrInvalidSearchEvent
	}
	if e.ProductID <= 0 {
		return ErrInvalidProductID
	}
	if e.Position <= 0 {
		return ErrInvalidPosition
	}
	if e.UserID != nil && *e.UserID <= 0 {
		return ErrInvalidUserID
	}

	if e.SessionID != nil {
		trimmed := strings.TrimSpace(*e.SessionID)
		if trimmed == "" {
			e.SessionID = nil
		} else {
			e.SessionID = &trimmed
		}
	}

	return nil
}
