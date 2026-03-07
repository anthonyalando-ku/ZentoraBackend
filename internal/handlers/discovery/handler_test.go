package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	discoverydomain "zentora-service/internal/domain/discovery"

	"github.com/gin-gonic/gin"
)

type stubDiscoveryService struct {
	feedReq        *discoverydomain.FeedRequest
	feedResult     []discoverydomain.ProductCard
	feedErr        error
	suggestReq     *discoverydomain.SuggestRequest
	suggestResult  []discoverydomain.Suggestion
	suggestErr     error
	searchEvent    *discoverydomain.SearchEvent
	searchEventID  int64
	searchEventErr error
	searchClick    *discoverydomain.SearchClickEvent
	searchClickErr error
}

func (s *stubDiscoveryService) GetFeed(_ context.Context, req *discoverydomain.FeedRequest) ([]discoverydomain.ProductCard, error) {
	s.feedReq = req
	return s.feedResult, s.feedErr
}

func (s *stubDiscoveryService) Suggest(_ context.Context, req *discoverydomain.SuggestRequest) ([]discoverydomain.Suggestion, error) {
	s.suggestReq = req
	return s.suggestResult, s.suggestErr
}

func (s *stubDiscoveryService) TrackSearch(_ context.Context, event *discoverydomain.SearchEvent) (int64, error) {
	s.searchEvent = event
	return s.searchEventID, s.searchEventErr
}

func (s *stubDiscoveryService) TrackSearchClick(_ context.Context, event *discoverydomain.SearchClickEvent) error {
	s.searchClick = event
	return s.searchClickErr
}

type stubMetricsRunner struct {
	called bool
	err    error
}

func (s *stubMetricsRunner) RunOnce(context.Context) error {
	s.called = true
	return s.err
}

func TestGetFeedCandidatesUsesAuthenticatedIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	discoverySvc := &stubDiscoveryService{
		feedResult: []discoverydomain.ProductCard{{ProductID: 11, Name: "Phone"}},
	}
	handler := NewHandler(discoverySvc, &stubMetricsRunner{})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/discovery/feed?feed_type=recommended&limit=5", nil)
	ctx.Request = req
	ctx.Set("identity_id", int64(44))

	handler.GetFeedCandidates(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if discoverySvc.feedReq == nil || discoverySvc.feedReq.UserID == nil || *discoverySvc.feedReq.UserID != 44 {
		t.Fatalf("feed request user_id = %#v, want authenticated identity", discoverySvc.feedReq)
	}
	if discoverySvc.feedReq.Limit != 5 {
		t.Fatalf("feed request limit = %d, want 5", discoverySvc.feedReq.Limit)
	}
}

func TestGetFeedParsesSharedFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	discoverySvc := &stubDiscoveryService{
		feedResult: []discoverydomain.ProductCard{{ProductID: 11, Name: "Phone"}},
	}
	handler := NewHandler(discoverySvc, &stubMetricsRunner{})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/discovery/feed?feed_type=trending&brand_ids=3,1&tag_ids=8,5&variant_attribute_value_ids=21,13&price_min=10&price_max=99.99&min_rating=4&discount_only=true&in_stock_only=true",
		nil,
	)
	ctx.Request = req

	handler.GetFeedCandidates(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if discoverySvc.feedReq == nil {
		t.Fatal("expected feed request to be passed to service")
	}
	if len(discoverySvc.feedReq.Filters.VariantAttributeValueIDs) != 2 {
		t.Fatalf("variant attribute ids = %#v, want parsed ids", discoverySvc.feedReq.Filters.VariantAttributeValueIDs)
	}
	if !discoverySvc.feedReq.Filters.DiscountOnly || !discoverySvc.feedReq.Filters.InStockOnly {
		t.Fatalf("filters = %#v, want discount and stock filters enabled", discoverySvc.feedReq.Filters)
	}
}

func TestTrackSearchUsesAuthenticatedIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	discoverySvc := &stubDiscoveryService{searchEventID: 91}
	handler := NewHandler(discoverySvc, &stubMetricsRunner{})

	body, _ := json.Marshal(map[string]any{
		"query":        "Earbuds",
		"result_count": 2,
		"results": []map[string]any{
			{"product_id": 7, "position": 1, "score": 0.9},
		},
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/search/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req
	ctx.Set("identity_id", int64(8))

	handler.TrackSearch(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if discoverySvc.searchEvent == nil || discoverySvc.searchEvent.UserID == nil || *discoverySvc.searchEvent.UserID != 8 {
		t.Fatalf("search event user_id = %#v, want authenticated identity", discoverySvc.searchEvent)
	}
}

func TestSuggestParsesPrefixAndLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	discoverySvc := &stubDiscoveryService{
		suggestResult: []discoverydomain.Suggestion{{Text: "earbuds"}},
	}
	handler := NewHandler(discoverySvc, &stubMetricsRunner{})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/discovery/suggest?prefix=ear&limit=3", nil)

	handler.Suggest(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if discoverySvc.suggestReq == nil || discoverySvc.suggestReq.Prefix != "ear" || discoverySvc.suggestReq.Limit != 3 {
		t.Fatalf("suggest request = %#v, want prefix and limit populated", discoverySvc.suggestReq)
	}
}

func TestTrackSearchClickUsesAuthenticatedIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	discoverySvc := &stubDiscoveryService{}
	handler := NewHandler(discoverySvc, &stubMetricsRunner{})

	body, _ := json.Marshal(map[string]any{
		"search_event_id": 15,
		"product_id":      6,
		"position":        2,
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/search/clicks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req
	ctx.Set("identity_id", int64(12))

	handler.TrackSearchClick(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if discoverySvc.searchClick == nil || discoverySvc.searchClick.UserID == nil || *discoverySvc.searchClick.UserID != 12 {
		t.Fatalf("search click = %#v, want authenticated identity", discoverySvc.searchClick)
	}
}

func TestRecomputeMetricsReturnsServerErrorOnFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	metrics := &stubMetricsRunner{err: errors.New("boom")}
	handler := NewHandler(&stubDiscoveryService{}, metrics)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/discovery/metrics/recompute", nil)

	handler.RecomputeMetrics(ctx)

	if !metrics.called {
		t.Fatal("expected metrics runner to be called")
	}
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}
