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

	return nil
}

type Candidate struct {
	ProductID int64
	Signals   map[string]float64
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
