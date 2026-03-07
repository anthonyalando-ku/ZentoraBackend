package discovery

import "errors"

var (
	ErrInvalidRequest     = errors.New("discovery request is required")
	ErrInvalidFeedType    = errors.New("feed_type is invalid")
	ErrCategoryRequired   = errors.New("category_id is required for category feeds")
	ErrInvalidCategoryID  = errors.New("category_id must be positive")
	ErrQueryRequired      = errors.New("query is required for search feeds")
	ErrFeedNotImplemented = errors.New("feed type is not implemented yet")
)
