package discovery

import "errors"

var (
	ErrInvalidRequest        = errors.New("discovery request is required")
	ErrInvalidFeedType       = errors.New("feed_type is invalid")
	ErrCategoryRequired      = errors.New("category_id is required for category feeds")
	ErrInvalidCategoryID     = errors.New("category_id must be positive")
	ErrInvalidUserID         = errors.New("user_id must be positive")
	ErrInvalidFilterID       = errors.New("filter ids must be positive")
	ErrInvalidPriceRange     = errors.New("price_min cannot be greater than price_max")
	ErrInvalidRatingFilter   = errors.New("min_rating must be between 0 and 5")
	ErrUserRequired          = errors.New("user_id is required for recommended feeds")
	ErrUserOrSessionRequired = errors.New("user_id or session_id is required for also_viewed feeds")
	ErrInvalidSearchEvent    = errors.New("search_event_id must be positive")
	ErrInvalidProductID      = errors.New("product_id must be positive")
	ErrInvalidPosition       = errors.New("position must be positive")
	ErrNegativeResultCount   = errors.New("result_count must be non-negative")
	ErrQueryRequired         = errors.New("query is required for search feeds")
	ErrPrefixRequired        = errors.New("prefix is required for suggestions")
	ErrFeedNotImplemented    = errors.New("feed type is not implemented yet")
)
