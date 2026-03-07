package discovery

import "testing"

func TestFeedRequestValidateDefaultsLimit(t *testing.T) {
	req := &FeedRequest{FeedType: FeedTrending}

	if err := req.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if req.Limit != DefaultFeedLimit {
		t.Fatalf("Validate() limit = %d, want %d", req.Limit, DefaultFeedLimit)
	}
}

func TestFeedRequestValidateCapsLimit(t *testing.T) {
	req := &FeedRequest{FeedType: FeedTrending, Limit: MaxFeedLimit + 50}

	if err := req.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if req.Limit != MaxFeedLimit {
		t.Fatalf("Validate() limit = %d, want %d", req.Limit, MaxFeedLimit)
	}
}

func TestFeedRequestValidateRequiresCategoryID(t *testing.T) {
	req := &FeedRequest{FeedType: FeedCategory}

	if err := req.Validate(); err != ErrCategoryRequired {
		t.Fatalf("Validate() error = %v, want %v", err, ErrCategoryRequired)
	}
}

func TestFeedRequestValidateRejectsInvalidCategoryID(t *testing.T) {
	categoryID := int64(0)
	req := &FeedRequest{FeedType: FeedCategory, CategoryID: &categoryID}

	if err := req.Validate(); err != ErrInvalidCategoryID {
		t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidCategoryID)
	}
}

func TestFeedRequestValidateTrimsQuery(t *testing.T) {
	query := "  sneakers  "
	req := &FeedRequest{FeedType: FeedSearch, Query: &query}

	if err := req.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if req.Query == nil || *req.Query != "sneakers" {
		t.Fatalf("Validate() query = %v, want sneakers", req.Query)
	}
}

func TestFeedRequestValidateRejectsBlankSearchQuery(t *testing.T) {
	query := "   "
	req := &FeedRequest{FeedType: FeedSearch, Query: &query}

	if err := req.Validate(); err != ErrQueryRequired {
		t.Fatalf("Validate() error = %v, want %v", err, ErrQueryRequired)
	}
}

func TestFeedRequestValidateAllowsAlsoViewedFeedType(t *testing.T) {
	req := &FeedRequest{FeedType: FeedAlsoViewed}

	if err := req.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestFeedRequestValidateRejectsUnknownFeedType(t *testing.T) {
	req := &FeedRequest{FeedType: "unknown"}

	if err := req.Validate(); err != ErrInvalidFeedType {
		t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidFeedType)
	}
}
