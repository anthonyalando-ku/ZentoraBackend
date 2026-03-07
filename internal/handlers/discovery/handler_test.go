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

type apiResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

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

func TestGetFeedReturnsFrontendReadyProductCards(t *testing.T) {
	gin.SetMode(gin.TestMode)

	discoverySvc := &stubDiscoveryService{
		feedResult: []discoverydomain.ProductCard{
			{
				ProductID:       11,
				Name:            "Phone",
				Slug:            "phone",
				PrimaryImage:    "https://cdn.example.com/p/11.jpg",
				Price:           149.99,
				Discount:        10,
				Rating:          4.3,
				ReviewCount:     17,
				InventoryStatus: discoverydomain.InventoryStatusLowStock,
				Brand:           "Zentora",
				Category:        "Electronics",
			},
		},
	}
	handler := NewHandler(discoverySvc, &stubMetricsRunner{})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/discovery/feed?feed_type=featured", nil)

	handler.GetFeedCandidates(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response apiResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	var payload struct {
		Items []discoverydomain.ProductCard `json:"items"`
	}
	if err := json.Unmarshal(response.Data, &payload); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(payload.Items))
	}
	card := payload.Items[0]
	if card.ProductID != 11 || card.Slug != "phone" || card.PrimaryImage == "" || card.Brand == "" || card.Category == "" {
		t.Fatalf("card = %#v, want frontend-ready product card fields", card)
	}
	if card.InventoryStatus != discoverydomain.InventoryStatusLowStock {
		t.Fatalf("inventory_status = %q, want %q", card.InventoryStatus, discoverydomain.InventoryStatusLowStock)
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

func TestSearchRequiresQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	discoverySvc := &stubDiscoveryService{}
	handler := NewHandler(discoverySvc, &stubMetricsRunner{})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/discovery/search", nil)

	handler.Search(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if discoverySvc.feedReq != nil {
		t.Fatalf("expected discovery service not to be called, got %#v", discoverySvc.feedReq)
	}
}

func TestSearchBuildsFeedSearchRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	discoverySvc := &stubDiscoveryService{
		feedResult: []discoverydomain.ProductCard{{ProductID: 11, Name: "Phone"}},
	}
	handler := NewHandler(discoverySvc, &stubMetricsRunner{})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/discovery/search?query=samsung+phone&limit=10&brand_ids=3,1",
		nil,
	)
	ctx.Request = req

	handler.Search(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if discoverySvc.feedReq == nil {
		t.Fatal("expected discovery service to receive feed request")
	}
	if discoverySvc.feedReq.FeedType != discoverydomain.FeedSearch {
		t.Fatalf("feed type = %q, want %q", discoverySvc.feedReq.FeedType, discoverydomain.FeedSearch)
	}
	if discoverySvc.feedReq.Query == nil || *discoverySvc.feedReq.Query != "samsung phone" {
		t.Fatalf("feed query = %#v, want trimmed search query", discoverySvc.feedReq.Query)
	}
	if discoverySvc.feedReq.Limit != 10 {
		t.Fatalf("feed limit = %d, want 10", discoverySvc.feedReq.Limit)
	}
	if len(discoverySvc.feedReq.Filters.BrandIDs) != 2 {
		t.Fatalf("brand filter count = %d, want 2", len(discoverySvc.feedReq.Filters.BrandIDs))
	}
	if discoverySvc.feedReq.Filters.BrandIDs[0] != 3 {
		t.Fatalf("first brand filter = %d, want 3", discoverySvc.feedReq.Filters.BrandIDs[0])
	}
	if discoverySvc.feedReq.Filters.BrandIDs[1] != 1 {
		t.Fatalf("second brand filter = %d, want 1", discoverySvc.feedReq.Filters.BrandIDs[1])
	}
}

func TestSearchReturnsFrontendReadyProductCards(t *testing.T) {
	gin.SetMode(gin.TestMode)

	discoverySvc := &stubDiscoveryService{
		feedResult: []discoverydomain.ProductCard{
			{
				ProductID:       25,
				Name:            "Wireless Earbuds",
				Slug:            "wireless-earbuds",
				PrimaryImage:    "https://cdn.example.com/p/25.jpg",
				Price:           89.50,
				Discount:        5,
				Rating:          4.8,
				ReviewCount:     203,
				InventoryStatus: discoverydomain.InventoryStatusInStock,
				Brand:           "Zentora Audio",
				Category:        "Audio",
			},
		},
	}
	handler := NewHandler(discoverySvc, &stubMetricsRunner{})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/discovery/search?query=earbuds", nil)

	handler.Search(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response apiResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	var payload struct {
		Query string                        `json:"query"`
		Items []discoverydomain.ProductCard `json:"items"`
	}
	if err := json.Unmarshal(response.Data, &payload); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if payload.Query != "earbuds" {
		t.Fatalf("query = %q, want %q", payload.Query, "earbuds")
	}
	if len(payload.Items) != 1 || payload.Items[0].ProductID != 25 || payload.Items[0].PrimaryImage == "" {
		t.Fatalf("items = %#v, want frontend-ready search product cards", payload.Items)
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
