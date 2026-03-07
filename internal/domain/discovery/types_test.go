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
	sessionID := "session-1"
	req := &FeedRequest{FeedType: FeedAlsoViewed, SessionID: &sessionID}

	if err := req.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestFeedRequestValidateRequiresUserIDForRecommended(t *testing.T) {
	req := &FeedRequest{FeedType: FeedRecommended}

	if err := req.Validate(); err != ErrUserRequired {
		t.Fatalf("Validate() error = %v, want %v", err, ErrUserRequired)
	}
}

func TestFeedRequestValidateRequiresIdentityForAlsoViewed(t *testing.T) {
	req := &FeedRequest{FeedType: FeedAlsoViewed}

	if err := req.Validate(); err != ErrUserOrSessionRequired {
		t.Fatalf("Validate() error = %v, want %v", err, ErrUserOrSessionRequired)
	}
}

func TestFeedRequestValidateTrimsSessionID(t *testing.T) {
	userID := int64(5)
	sessionID := "  session-42  "
	req := &FeedRequest{FeedType: FeedAlsoViewed, UserID: &userID, SessionID: &sessionID}

	if err := req.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if req.SessionID == nil || *req.SessionID != "session-42" {
		t.Fatalf("Validate() session_id = %v, want session-42", req.SessionID)
	}
}

func TestFeedRequestValidateRejectsUnknownFeedType(t *testing.T) {
	req := &FeedRequest{FeedType: "unknown"}

	if err := req.Validate(); err != ErrInvalidFeedType {
		t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidFeedType)
	}
}

func TestSuggestRequestValidateDefaultsLimitAndTrimsPrefix(t *testing.T) {
	req := &SuggestRequest{Prefix: "  elec  "}

	if err := req.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if req.Prefix != "elec" {
		t.Fatalf("Validate() prefix = %q, want %q", req.Prefix, "elec")
	}
	if req.Limit != DefaultSuggestLimit {
		t.Fatalf("Validate() limit = %d, want %d", req.Limit, DefaultSuggestLimit)
	}
}

func TestSuggestRequestValidateCapsLimit(t *testing.T) {
	req := &SuggestRequest{Prefix: "elec", Limit: MaxSuggestLimit + 10}

	if err := req.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if req.Limit != MaxSuggestLimit {
		t.Fatalf("Validate() limit = %d, want %d", req.Limit, MaxSuggestLimit)
	}
}

func TestSuggestRequestValidateRejectsBlankPrefix(t *testing.T) {
	req := &SuggestRequest{Prefix: "   "}

	if err := req.Validate(); err != ErrPrefixRequired {
		t.Fatalf("Validate() error = %v, want %v", err, ErrPrefixRequired)
	}
}

func TestSearchEventValidateNormalizesQueryAndSession(t *testing.T) {
	sessionID := "  session-1  "
	req := &SearchEvent{
		Query:       "  Wireless Earbuds  ",
		SessionID:   &sessionID,
		ResultCount: 2,
		Results: []SearchResultPosition{
			{ProductID: 1, Position: 1, Score: 0.9},
		},
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if req.Query != "Wireless Earbuds" {
		t.Fatalf("Validate() query = %q, want %q", req.Query, "Wireless Earbuds")
	}
	if req.NormalizedQuery != "wireless earbuds" {
		t.Fatalf("Validate() normalized query = %q, want %q", req.NormalizedQuery, "wireless earbuds")
	}
	if req.SessionID == nil || *req.SessionID != "session-1" {
		t.Fatalf("Validate() session_id = %v, want session-1", req.SessionID)
	}
}

func TestSearchEventValidateRejectsNegativeResultCount(t *testing.T) {
	req := &SearchEvent{Query: "earbuds", ResultCount: -1}

	if err := req.Validate(); err != ErrNegativeResultCount {
		t.Fatalf("Validate() error = %v, want %v", err, ErrNegativeResultCount)
	}
}

func TestSearchEventValidateRejectsInvalidResultPositionProductID(t *testing.T) {
	req := &SearchEvent{
		Query: "earbuds",
		Results: []SearchResultPosition{
			{ProductID: 0, Position: 1, Score: 0.4},
		},
	}

	if err := req.Validate(); err != ErrInvalidProductID {
		t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidProductID)
	}
}

func TestSearchClickEventValidateRejectsInvalidSearchEventID(t *testing.T) {
	req := &SearchClickEvent{SearchEventID: 0, ProductID: 1, Position: 1}

	if err := req.Validate(); err != ErrInvalidSearchEvent {
		t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidSearchEvent)
	}
}

func TestSearchClickEventValidateTrimsBlankSessionToNil(t *testing.T) {
	sessionID := "   "
	req := &SearchClickEvent{
		SearchEventID: 1,
		ProductID:     2,
		Position:      1,
		SessionID:     &sessionID,
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if req.SessionID != nil {
		t.Fatalf("Validate() session_id = %v, want nil", req.SessionID)
	}
}
