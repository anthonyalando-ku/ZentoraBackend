package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"testing"
	"time"

	categorydomain "zentora-service/internal/domain/category"
	discoverydomain "zentora-service/internal/domain/discovery"

	"github.com/redis/go-redis/v9"
)

type stubCandidateRepository struct {
	called            bool
	req               *discoverydomain.FeedRequest
	reqs              []*discoverydomain.FeedRequest
	feedCalls         []discoverydomain.FeedType
	result            []discoverydomain.Candidate
	resultsByFeed     map[discoverydomain.FeedType][]discoverydomain.Candidate
	hydrateCalled     bool
	hydrateIDs        []int64
	hydrateResult     []discoverydomain.ProductCard
	hydrateErr        error
	err               error
	errByFeed         map[discoverydomain.FeedType]error
	suggestCalled     bool
	suggestReq        *discoverydomain.SuggestRequest
	suggestResult     []discoverydomain.Suggestion
	suggestErr        error
	searchEventCalled bool
	searchEvent       *discoverydomain.SearchEvent
	searchEventID     int64
	searchEventErr    error
	searchClickCalled bool
	searchClick       *discoverydomain.SearchClickEvent
	searchClickErr    error
}

func (s *stubCandidateRepository) GetFeedCandidates(_ context.Context, req *discoverydomain.FeedRequest) ([]discoverydomain.Candidate, error) {
	s.called = true
	s.req = req
	cloned := *req
	cloned.Filters.BrandIDs = append([]int64(nil), req.Filters.BrandIDs...)
	cloned.Filters.TagIDs = append([]int64(nil), req.Filters.TagIDs...)
	cloned.Filters.VariantAttributeValueIDs = append([]int64(nil), req.Filters.VariantAttributeValueIDs...)
	s.reqs = append(s.reqs, &cloned)
	s.feedCalls = append(s.feedCalls, req.FeedType)
	if err, ok := s.errByFeed[req.FeedType]; ok {
		return nil, err
	}
	if result, ok := s.resultsByFeed[req.FeedType]; ok {
		return result, nil
	}
	return s.result, s.err
}

func (s *stubCandidateRepository) HydrateProductCards(_ context.Context, productIDs []int64) ([]discoverydomain.ProductCard, error) {
	s.hydrateCalled = true
	s.hydrateIDs = append([]int64(nil), productIDs...)
	return s.hydrateResult, s.hydrateErr
}

func (s *stubCandidateRepository) Suggest(_ context.Context, req *discoverydomain.SuggestRequest) ([]discoverydomain.Suggestion, error) {
	s.suggestCalled = true
	s.suggestReq = req
	return s.suggestResult, s.suggestErr
}

func (s *stubCandidateRepository) TrackSearch(_ context.Context, event *discoverydomain.SearchEvent) (int64, error) {
	s.searchEventCalled = true
	s.searchEvent = event
	return s.searchEventID, s.searchEventErr
}

func (s *stubCandidateRepository) TrackSearchClick(_ context.Context, event *discoverydomain.SearchClickEvent) error {
	s.searchClickCalled = true
	s.searchClick = event
	return s.searchClickErr
}

type stubCategoryRepository struct {
	called     bool
	categoryID int64
	category   *categorydomain.Category
	err        error
}

func (s *stubCategoryRepository) GetCategoryByID(_ context.Context, id int64) (*categorydomain.Category, error) {
	s.called = true
	s.categoryID = id
	return s.category, s.err
}

type featuredExecutionProduct struct {
	card                     discoverydomain.ProductCard
	brandID                  int64
	tagIDs                   []int64
	variantAttributeValueIDs []int64
	isFeatured               bool
	homepageSectionTypes     []string
	createdAt                time.Time
}

type featuredExecutionRepository struct {
	products []featuredExecutionProduct
}

func (r *featuredExecutionRepository) GetFeedCandidates(_ context.Context, req *discoverydomain.FeedRequest) ([]discoverydomain.Candidate, error) {
	if req.FeedType != discoverydomain.FeedFeatured {
		return nil, discoverydomain.ErrFeedNotImplemented
	}

	candidates := make([]discoverydomain.Candidate, 0, len(r.products))
	for _, product := range r.products {
		if !featuredExecutionMatchesSource(product) || !featuredExecutionMatchesFilters(product, req.Filters) {
			continue
		}

		score := 1.0
		for _, sectionType := range product.homepageSectionTypes {
			switch sectionType {
			case "custom":
				score = maxFloat(score, 1000)
			case "featured":
				score = maxFloat(score, 800)
			}
		}

		candidates = append(candidates, discoverydomain.Candidate{
			ProductID: product.card.ProductID,
			Signals: map[string]float64{
				"merchandising_score": score,
				"freshness_score":     float64(product.createdAt.Unix()),
			},
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if left.Signals["merchandising_score"] != right.Signals["merchandising_score"] {
			return left.Signals["merchandising_score"] > right.Signals["merchandising_score"]
		}
		if left.Signals["freshness_score"] != right.Signals["freshness_score"] {
			return left.Signals["freshness_score"] > right.Signals["freshness_score"]
		}
		return left.ProductID > right.ProductID
	})

	if len(candidates) > req.Limit {
		candidates = candidates[:req.Limit]
	}
	return candidates, nil
}

func (r *featuredExecutionRepository) HydrateProductCards(_ context.Context, productIDs []int64) ([]discoverydomain.ProductCard, error) {
	byID := make(map[int64]discoverydomain.ProductCard, len(r.products))
	for _, product := range r.products {
		byID[product.card.ProductID] = product.card
	}

	cards := make([]discoverydomain.ProductCard, 0, len(productIDs))
	for _, productID := range productIDs {
		card, ok := byID[productID]
		if !ok {
			continue
		}
		cards = append(cards, card)
	}
	return cards, nil
}

func (r *featuredExecutionRepository) Suggest(context.Context, *discoverydomain.SuggestRequest) ([]discoverydomain.Suggestion, error) {
	return nil, nil
}

func (r *featuredExecutionRepository) TrackSearch(context.Context, *discoverydomain.SearchEvent) (int64, error) {
	return 0, nil
}

func (r *featuredExecutionRepository) TrackSearchClick(context.Context, *discoverydomain.SearchClickEvent) error {
	return nil
}

func featuredExecutionMatchesSource(product featuredExecutionProduct) bool {
	if product.isFeatured {
		return true
	}
	for _, sectionType := range product.homepageSectionTypes {
		if sectionType == "featured" || sectionType == "custom" {
			return true
		}
	}
	return false
}

func featuredExecutionMatchesFilters(product featuredExecutionProduct, filters discoverydomain.FeedFilter) bool {
	if len(filters.BrandIDs) > 0 && !containsInt64(filters.BrandIDs, product.brandID) {
		return false
	}
	if len(filters.TagIDs) > 0 && !containsAnyInt64(product.tagIDs, filters.TagIDs) {
		return false
	}
	if filters.PriceMin != nil && product.card.Price < *filters.PriceMin {
		return false
	}
	if filters.PriceMax != nil && product.card.Price > *filters.PriceMax {
		return false
	}
	if filters.MinRating != nil && product.card.Rating < *filters.MinRating {
		return false
	}
	if filters.DiscountOnly && product.card.Discount <= 0 {
		return false
	}
	if filters.InStockOnly && product.card.InventoryStatus == discoverydomain.InventoryStatusOutOfStock {
		return false
	}
	if len(filters.VariantAttributeValueIDs) > 0 && !containsAllInt64(product.variantAttributeValueIDs, filters.VariantAttributeValueIDs) {
		return false
	}
	return true
}

func containsInt64(values []int64, target int64) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsAnyInt64(values []int64, targets []int64) bool {
	for _, target := range targets {
		if containsInt64(values, target) {
			return true
		}
	}
	return false
}

func containsAllInt64(values []int64, targets []int64) bool {
	for _, target := range targets {
		if !containsInt64(values, target) {
			return false
		}
	}
	return true
}

func maxFloat(left, right float64) float64 {
	if right > left {
		return right
	}
	return left
}

func assertInt64SliceEqual(t *testing.T, got, want []int64, message string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", message, got, want)
	}
}

func TestDiscoveryServiceGetFeedCandidatesValidatesRequest(t *testing.T) {
	svc := NewDiscoveryService(&stubCandidateRepository{}, &stubCategoryRepository{})

	_, err := svc.GetFeed(context.Background(), nil)
	if !errors.Is(err, discoverydomain.ErrInvalidRequest) {
		t.Fatalf("GetFeed() error = %v, want %v", err, discoverydomain.ErrInvalidRequest)
	}
}

func TestDiscoveryServiceGetFeedCandidatesChecksCategoryExists(t *testing.T) {
	categoryID := int64(42)
	candidateRepo := &stubCandidateRepository{}
	categoryRepo := &stubCategoryRepository{err: categorydomain.ErrNotFound}
	svc := NewDiscoveryService(candidateRepo, categoryRepo)

	_, err := svc.GetFeed(context.Background(), &discoverydomain.FeedRequest{
		FeedType:   discoverydomain.FeedCategory,
		CategoryID: &categoryID,
	})
	if !errors.Is(err, categorydomain.ErrNotFound) {
		t.Fatalf("GetFeed() error = %v, want %v", err, categorydomain.ErrNotFound)
	}
	if !categoryRepo.called {
		t.Fatal("expected category repository to be called")
	}
	if candidateRepo.called {
		t.Fatal("expected discovery repository not to be called when category lookup fails")
	}
}

func TestDiscoveryServiceGetFeedCandidatesPassesCategoryFeedToRepository(t *testing.T) {
	categoryID := int64(7)
	candidateRepo := &stubCandidateRepository{
		result: []discoverydomain.Candidate{
			{ProductID: 101, Signals: map[string]float64{"category_score": 1}},
		},
		hydrateResult: []discoverydomain.ProductCard{
			{ProductID: 101, Name: "Phone", Slug: "phone"},
		},
	}
	categoryRepo := &stubCategoryRepository{category: &categorydomain.Category{ID: categoryID}}
	svc := NewDiscoveryService(candidateRepo, categoryRepo)

	got, err := svc.GetFeed(context.Background(), &discoverydomain.FeedRequest{
		FeedType:   discoverydomain.FeedCategory,
		CategoryID: &categoryID,
	})
	if err != nil {
		t.Fatalf("GetFeed() error = %v", err)
	}
	if !categoryRepo.called {
		t.Fatal("expected category repository to be called")
	}
	if categoryRepo.categoryID != categoryID {
		t.Fatalf("category lookup id = %d, want %d", categoryRepo.categoryID, categoryID)
	}
	if !candidateRepo.called {
		t.Fatal("expected discovery repository to be called")
	}
	if candidateRepo.req == nil || candidateRepo.req.CategoryID == nil || *candidateRepo.req.CategoryID != categoryID {
		t.Fatalf("repository request category_id = %v, want %d", candidateRepo.req, categoryID)
	}
	if !candidateRepo.hydrateCalled {
		t.Fatal("expected hydrate method to be called")
	}
	if len(got) != 1 || got[0].ProductID != 101 {
		t.Fatalf("GetFeed() = %#v, want hydrated product card", got)
	}
}

func TestDiscoveryServiceGetFeedCandidatesSkipsCategoryLookupForNonCategoryFeed(t *testing.T) {
	candidateRepo := &stubCandidateRepository{
		result: []discoverydomain.Candidate{
			{ProductID: 1, Signals: map[string]float64{"trending_score": 5}},
		},
		hydrateResult: []discoverydomain.ProductCard{
			{ProductID: 1, Name: "Laptop"},
		},
	}
	categoryRepo := &stubCategoryRepository{}
	svc := NewDiscoveryService(candidateRepo, categoryRepo)

	got, err := svc.GetFeed(context.Background(), &discoverydomain.FeedRequest{
		FeedType: discoverydomain.FeedTrending,
	})
	if err != nil {
		t.Fatalf("GetFeed() error = %v", err)
	}
	if categoryRepo.called {
		t.Fatal("expected category repository not to be called")
	}
	if !candidateRepo.called {
		t.Fatal("expected discovery repository to be called")
	}
	if !candidateRepo.hydrateCalled {
		t.Fatal("expected hydrate method to be called")
	}
	if len(got) != 1 || got[0].ProductID != 1 {
		t.Fatalf("GetFeed() = %#v, want hydrated product card", got)
	}
}

func TestDiscoveryServiceGetFeedCandidatesPassesEditorialFeedToRepository(t *testing.T) {
	candidateRepo := &stubCandidateRepository{
		result: []discoverydomain.Candidate{
			{ProductID: 77, Signals: map[string]float64{"merchandising_score": 15}},
		},
		hydrateResult: []discoverydomain.ProductCard{
			{ProductID: 77, Name: "Editorial Pick"},
		},
	}
	svc := NewDiscoveryService(candidateRepo, &stubCategoryRepository{})

	got, err := svc.GetFeed(context.Background(), &discoverydomain.FeedRequest{
		FeedType: discoverydomain.FeedEditorial,
	})
	if err != nil {
		t.Fatalf("GetFeed() error = %v", err)
	}
	if !candidateRepo.called {
		t.Fatal("expected discovery repository to be called")
	}
	if candidateRepo.req == nil || candidateRepo.req.FeedType != discoverydomain.FeedEditorial {
		t.Fatalf("repository request = %#v, want editorial feed", candidateRepo.req)
	}
	if !candidateRepo.hydrateCalled {
		t.Fatal("expected hydrate method to be called")
	}
	if len(got) != 1 || got[0].ProductID != 77 {
		t.Fatalf("GetFeed() = %#v, want hydrated editorial card", got)
	}
}

func TestDiscoveryServiceGetFeedCandidatesRequiresUserForRecommendedFeed(t *testing.T) {
	candidateRepo := &stubCandidateRepository{}
	svc := NewDiscoveryService(candidateRepo, &stubCategoryRepository{})

	_, err := svc.GetFeed(context.Background(), &discoverydomain.FeedRequest{
		FeedType: discoverydomain.FeedRecommended,
	})
	if !errors.Is(err, discoverydomain.ErrUserRequired) {
		t.Fatalf("GetFeed() error = %v, want %v", err, discoverydomain.ErrUserRequired)
	}
	if candidateRepo.called {
		t.Fatal("expected discovery repository not to be called when recommended feed is invalid")
	}
}

func TestDiscoveryServiceGetFeedCandidatesPassesAlsoViewedFeedToRepository(t *testing.T) {
	sessionID := "  session-9  "
	candidateRepo := &stubCandidateRepository{
		result: []discoverydomain.Candidate{
			{ProductID: 22, Signals: map[string]float64{"co_view_score": 0.7}},
		},
		hydrateResult: []discoverydomain.ProductCard{
			{ProductID: 22, Name: "Camera"},
		},
	}
	svc := NewDiscoveryService(candidateRepo, &stubCategoryRepository{})

	got, err := svc.GetFeed(context.Background(), &discoverydomain.FeedRequest{
		FeedType:  discoverydomain.FeedAlsoViewed,
		SessionID: &sessionID,
	})
	if err != nil {
		t.Fatalf("GetFeed() error = %v", err)
	}
	if !candidateRepo.called {
		t.Fatal("expected discovery repository to be called")
	}
	if candidateRepo.req == nil || candidateRepo.req.SessionID == nil || *candidateRepo.req.SessionID != "session-9" {
		t.Fatalf("repository request session_id = %#v, want %q", candidateRepo.req, "session-9")
	}
	if len(got) != 1 || got[0].ProductID != 22 {
		t.Fatalf("GetFeed() = %#v, want hydrated product card", got)
	}
}

func TestDiscoveryServiceGetFeedBackfillsRecommendedWithTrendingAndBestSellers(t *testing.T) {
	userID := int64(55)
	candidateRepo := &stubCandidateRepository{
		resultsByFeed: map[discoverydomain.FeedType][]discoverydomain.Candidate{
			discoverydomain.FeedRecommended: {
				{ProductID: 1},
				{ProductID: 2},
			},
			discoverydomain.FeedTrending: {
				{ProductID: 2},
				{ProductID: 3},
			},
			discoverydomain.FeedBestSellers: {
				{ProductID: 4},
			},
		},
		hydrateResult: []discoverydomain.ProductCard{
			{ProductID: 1, Name: "One"},
			{ProductID: 2, Name: "Two"},
			{ProductID: 3, Name: "Three"},
			{ProductID: 4, Name: "Four"},
		},
	}
	svc := NewDiscoveryService(candidateRepo, &stubCategoryRepository{})

	got, err := svc.GetFeed(context.Background(), &discoverydomain.FeedRequest{
		FeedType: discoverydomain.FeedRecommended,
		UserID:   &userID,
		Limit:    4,
	})
	if err != nil {
		t.Fatalf("GetFeed() error = %v", err)
	}
	if len(candidateRepo.feedCalls) != 3 {
		t.Fatalf("feed call count = %d, want 3", len(candidateRepo.feedCalls))
	}
	if candidateRepo.feedCalls[0] != discoverydomain.FeedRecommended || candidateRepo.feedCalls[1] != discoverydomain.FeedTrending || candidateRepo.feedCalls[2] != discoverydomain.FeedBestSellers {
		t.Fatalf("feed calls = %#v, want recommended -> trending -> best_sellers", candidateRepo.feedCalls)
	}
	assertInt64SliceEqual(t, candidateRepo.hydrateIDs, []int64{1, 2, 3, 4}, "hydrate ids")
	if len(got) != 4 || got[3].ProductID != 4 {
		t.Fatalf("GetFeed() = %#v, want 4 hydrated cards", got)
	}
}

func TestDiscoveryServiceGetFeedBackfillsAlsoViewedUntilTrendingFillsLimit(t *testing.T) {
	sessionID := "session-7"
	candidateRepo := &stubCandidateRepository{
		resultsByFeed: map[discoverydomain.FeedType][]discoverydomain.Candidate{
			discoverydomain.FeedAlsoViewed: nil,
			discoverydomain.FeedTrending: {
				{ProductID: 11},
				{ProductID: 12},
				{ProductID: 13},
			},
		},
		hydrateResult: []discoverydomain.ProductCard{
			{ProductID: 11, Name: "Eleven"},
			{ProductID: 12, Name: "Twelve"},
			{ProductID: 13, Name: "Thirteen"},
		},
	}
	svc := NewDiscoveryService(candidateRepo, &stubCategoryRepository{})

	got, err := svc.GetFeed(context.Background(), &discoverydomain.FeedRequest{
		FeedType:  discoverydomain.FeedAlsoViewed,
		SessionID: &sessionID,
		Limit:     3,
	})
	if err != nil {
		t.Fatalf("GetFeed() error = %v", err)
	}
	if len(candidateRepo.feedCalls) != 2 {
		t.Fatalf("feed call count = %d, want 2", len(candidateRepo.feedCalls))
	}
	if candidateRepo.feedCalls[0] != discoverydomain.FeedAlsoViewed || candidateRepo.feedCalls[1] != discoverydomain.FeedTrending {
		t.Fatalf("feed calls = %#v, want also_viewed -> trending", candidateRepo.feedCalls)
	}
	assertInt64SliceEqual(t, candidateRepo.hydrateIDs, []int64{11, 12, 13}, "hydrate ids")
	if len(got) != 3 || got[0].ProductID != 11 {
		t.Fatalf("GetFeed() = %#v, want hydrated backfill cards", got)
	}
}

func TestDiscoveryServiceGetFeedBackfillPreservesFiltersAndRemainingLimit(t *testing.T) {
	userID := int64(9)
	priceMin := 10.0
	minRating := 4.5
	req := &discoverydomain.FeedRequest{
		FeedType: discoverydomain.FeedRecommended,
		UserID:   &userID,
		Limit:    4,
		Filters: discoverydomain.FeedFilter{
			BrandIDs:                 []int64{5, 1},
			TagIDs:                   []int64{7},
			VariantAttributeValueIDs: []int64{21, 13},
			PriceMin:                 &priceMin,
			MinRating:                &minRating,
			DiscountOnly:             true,
			InStockOnly:              true,
		},
	}
	candidateRepo := &stubCandidateRepository{
		resultsByFeed: map[discoverydomain.FeedType][]discoverydomain.Candidate{
			discoverydomain.FeedRecommended: {
				{ProductID: 1},
			},
			discoverydomain.FeedTrending: {
				{ProductID: 2},
				{ProductID: 3},
				{ProductID: 4},
			},
		},
		hydrateResult: []discoverydomain.ProductCard{
			{ProductID: 1, Name: "One"},
			{ProductID: 2, Name: "Two"},
			{ProductID: 3, Name: "Three"},
			{ProductID: 4, Name: "Four"},
		},
	}

	svc := NewDiscoveryService(candidateRepo, &stubCategoryRepository{})
	if _, err := svc.GetFeed(context.Background(), req); err != nil {
		t.Fatalf("GetFeed() error = %v", err)
	}

	if len(candidateRepo.reqs) != 2 {
		t.Fatalf("request count = %d, want 2", len(candidateRepo.reqs))
	}
	fallbackReq := candidateRepo.reqs[1]
	if fallbackReq.FeedType != discoverydomain.FeedTrending {
		t.Fatalf("fallback feed type = %q, want %q", fallbackReq.FeedType, discoverydomain.FeedTrending)
	}
	if fallbackReq.Limit != 3 {
		t.Fatalf("fallback limit = %d, want 3", fallbackReq.Limit)
	}
	if fallbackReq.UserID == nil || *fallbackReq.UserID != userID {
		t.Fatalf("fallback user_id = %#v, want %d", fallbackReq.UserID, userID)
	}
	assertInt64SliceEqual(t, fallbackReq.Filters.BrandIDs, []int64{1, 5}, "fallback brand ids")
	assertInt64SliceEqual(t, fallbackReq.Filters.TagIDs, []int64{7}, "fallback tag ids")
	assertInt64SliceEqual(t, fallbackReq.Filters.VariantAttributeValueIDs, []int64{13, 21}, "fallback variant attribute ids")
	if fallbackReq.Filters.PriceMin == nil || *fallbackReq.Filters.PriceMin != priceMin {
		t.Fatalf("fallback price min = %#v, want %v", fallbackReq.Filters.PriceMin, priceMin)
	}
	if fallbackReq.Filters.MinRating == nil || *fallbackReq.Filters.MinRating != minRating {
		t.Fatalf("fallback min rating = %#v, want %v", fallbackReq.Filters.MinRating, minRating)
	}
	if !fallbackReq.Filters.DiscountOnly || !fallbackReq.Filters.InStockOnly {
		t.Fatalf("fallback filters = %#v, want boolean filters preserved", fallbackReq.Filters)
	}
	assertInt64SliceEqual(t, req.Filters.BrandIDs, []int64{1, 5}, "original request brand ids")
}

type stubCacheClient struct {
	values map[string][]byte
	gets   int
	sets   int
}

func (s *stubCacheClient) Get(_ context.Context, key string) *redis.StringCmd {
	s.gets++
	if s.values == nil {
		s.values = make(map[string][]byte)
	}
	if value, ok := s.values[key]; ok {
		return redis.NewStringResult(string(value), nil)
	}
	return redis.NewStringResult("", redis.Nil)
}

func (s *stubCacheClient) Set(_ context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	s.sets++
	if s.values == nil {
		s.values = make(map[string][]byte)
	}
	switch typed := value.(type) {
	case []byte:
		s.values[key] = append([]byte(nil), typed...)
	default:
		encoded, _ := json.Marshal(typed)
		s.values[key] = encoded
	}
	return redis.NewStatusResult("OK", nil)
}

func TestDiscoveryServiceGetFeedUsesCacheForHotFeeds(t *testing.T) {
	cache := &stubCacheClient{}
	req := &discoverydomain.FeedRequest{FeedType: discoverydomain.FeedTrending}
	if err := req.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	key := buildFeedCacheKey(req)
	cached := []discoverydomain.ProductCard{{ProductID: 33, Name: "Cached"}}
	payload, _ := json.Marshal(cached)
	cache.values = map[string][]byte{key: payload}

	candidateRepo := &stubCandidateRepository{}
	svc := NewDiscoveryService(candidateRepo, &stubCategoryRepository{}, cache)

	got, err := svc.GetFeed(context.Background(), req)
	if err != nil {
		t.Fatalf("GetFeed() error = %v", err)
	}
	if candidateRepo.called {
		t.Fatal("expected candidate repository not to be called on cache hit")
	}
	if len(got) != 1 || got[0].ProductID != 33 {
		t.Fatalf("GetFeed() = %#v, want cached cards", got)
	}
}

func TestDiscoveryServiceGetFeedStoresHotFeedInCache(t *testing.T) {
	cache := &stubCacheClient{}
	candidateRepo := &stubCandidateRepository{
		result:        []discoverydomain.Candidate{{ProductID: 88}},
		hydrateResult: []discoverydomain.ProductCard{{ProductID: 88, Name: "Fresh"}},
	}
	svc := NewDiscoveryService(candidateRepo, &stubCategoryRepository{}, cache)

	got, err := svc.GetFeed(context.Background(), &discoverydomain.FeedRequest{FeedType: discoverydomain.FeedTrending})
	if err != nil {
		t.Fatalf("GetFeed() error = %v", err)
	}
	if len(got) != 1 || got[0].ProductID != 88 {
		t.Fatalf("GetFeed() = %#v, want hydrated cards", got)
	}
	if cache.sets == 0 {
		t.Fatal("expected feed result to be stored in cache")
	}
}

func TestDiscoveryServiceGetFeedStoresEditorialFeedInCache(t *testing.T) {
	cache := &stubCacheClient{}
	candidateRepo := &stubCandidateRepository{
		result:        []discoverydomain.Candidate{{ProductID: 91}},
		hydrateResult: []discoverydomain.ProductCard{{ProductID: 91, Name: "Editorial"}},
	}
	svc := NewDiscoveryService(candidateRepo, &stubCategoryRepository{}, cache)

	req := &discoverydomain.FeedRequest{FeedType: discoverydomain.FeedEditorial}
	got, err := svc.GetFeed(context.Background(), req)
	if err != nil {
		t.Fatalf("GetFeed() error = %v", err)
	}
	if len(got) != 1 || got[0].ProductID != 91 {
		t.Fatalf("GetFeed() = %#v, want hydrated editorial cards", got)
	}
	if cache.sets == 0 {
		t.Fatal("expected editorial feed result to be stored in cache")
	}
}

func TestDiscoveryServiceGetFeedReturnsFrontendReadyHydratedCards(t *testing.T) {
	candidateRepo := &stubCandidateRepository{
		result: []discoverydomain.Candidate{{ProductID: 71}},
		hydrateResult: []discoverydomain.ProductCard{
			{
				ProductID:       71,
				Name:            "Flagship Phone",
				Slug:            "flagship-phone",
				PrimaryImage:    "https://cdn.example.com/products/71.jpg",
				Price:           499.99,
				Discount:        12.50,
				Rating:          4.7,
				ReviewCount:     124,
				InventoryStatus: discoverydomain.InventoryStatusInStock,
				Brand:           "Zentora",
				Category:        "Phones",
			},
		},
	}
	svc := NewDiscoveryService(candidateRepo, &stubCategoryRepository{})

	got, err := svc.GetFeed(context.Background(), &discoverydomain.FeedRequest{FeedType: discoverydomain.FeedFeatured})
	if err != nil {
		t.Fatalf("GetFeed() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("GetFeed() card count = %d, want 1", len(got))
	}
	card := got[0]
	if card.ProductID != 71 || card.Name == "" || card.Slug == "" || card.PrimaryImage == "" || card.Brand == "" || card.Category == "" {
		t.Fatalf("GetFeed() card = %#v, want frontend-ready product card fields", card)
	}
	if card.Price != 499.99 || card.Discount != 12.50 || card.Rating != 4.7 || card.ReviewCount != 124 {
		t.Fatalf("GetFeed() numeric fields = %#v, want hydrated pricing and rating fields", card)
	}
	if card.InventoryStatus != discoverydomain.InventoryStatusInStock {
		t.Fatalf("GetFeed() inventory_status = %q, want %q", card.InventoryStatus, discoverydomain.InventoryStatusInStock)
	}
}

func TestDiscoveryServiceGetFeedExecutesFeaturedPipelineWithHomepageSectionsAndFilters(t *testing.T) {
	priceMin := 50.0
	priceMax := 150.0
	minRating := 4.5
	now := time.Unix(1_700_000_000, 0)

	repo := &featuredExecutionRepository{
		products: []featuredExecutionProduct{
			{
				card: discoverydomain.ProductCard{
					ProductID:       101,
					Name:            "Flagship Phone",
					Slug:            "flagship-phone",
					PrimaryImage:    "https://cdn.example.com/products/101.jpg",
					Price:           129.99,
					Discount:        15,
					Rating:          4.9,
					ReviewCount:     120,
					InventoryStatus: discoverydomain.InventoryStatusInStock,
					Brand:           "Zentora",
					Category:        "Phones",
				},
				brandID:                  1,
				tagIDs:                   []int64{10, 20},
				variantAttributeValueIDs: []int64{100, 101},
				isFeatured:               true,
				createdAt:                now.Add(-2 * time.Hour),
			},
			{
				card: discoverydomain.ProductCard{
					ProductID:       202,
					Name:            "Curated Earbuds",
					Slug:            "curated-earbuds",
					PrimaryImage:    "https://cdn.example.com/products/202.jpg",
					Price:           89.50,
					Discount:        5,
					Rating:          4.8,
					ReviewCount:     88,
					InventoryStatus: discoverydomain.InventoryStatusLowStock,
					Brand:           "Zentora Audio",
					Category:        "Audio",
				},
				brandID:                  1,
				tagIDs:                   []int64{10},
				variantAttributeValueIDs: []int64{100},
				homepageSectionTypes:     []string{"custom"},
				createdAt:                now.Add(-1 * time.Hour),
			},
			{
				card: discoverydomain.ProductCard{
					ProductID:       303,
					Name:            "Wrong Brand Tablet",
					Slug:            "wrong-brand-tablet",
					PrimaryImage:    "https://cdn.example.com/products/303.jpg",
					Price:           119.99,
					Discount:        20,
					Rating:          4.7,
					ReviewCount:     40,
					InventoryStatus: discoverydomain.InventoryStatusInStock,
					Brand:           "Other Brand",
					Category:        "Tablets",
				},
				brandID:                  2,
				tagIDs:                   []int64{10},
				variantAttributeValueIDs: []int64{100},
				isFeatured:               true,
				createdAt:                now,
			},
			{
				card: discoverydomain.ProductCard{
					ProductID:       404,
					Name:            "Out Of Stock Camera",
					Slug:            "out-of-stock-camera",
					PrimaryImage:    "https://cdn.example.com/products/404.jpg",
					Price:           99.99,
					Discount:        12,
					Rating:          4.6,
					ReviewCount:     12,
					InventoryStatus: discoverydomain.InventoryStatusOutOfStock,
					Brand:           "Zentora",
					Category:        "Cameras",
				},
				brandID:                  1,
				tagIDs:                   []int64{10},
				variantAttributeValueIDs: []int64{100},
				homepageSectionTypes:     []string{"featured"},
				createdAt:                now.Add(-30 * time.Minute),
			},
		},
	}
	svc := NewDiscoveryService(repo, &stubCategoryRepository{})

	got, err := svc.GetFeed(context.Background(), &discoverydomain.FeedRequest{
		FeedType: discoverydomain.FeedFeatured,
		Limit:    5,
		Filters: discoverydomain.FeedFilter{
			BrandIDs:                 []int64{1},
			TagIDs:                   []int64{10},
			VariantAttributeValueIDs: []int64{100},
			PriceMin:                 &priceMin,
			PriceMax:                 &priceMax,
			MinRating:                &minRating,
			DiscountOnly:             true,
			InStockOnly:              true,
		},
	})
	if err != nil {
		t.Fatalf("GetFeed() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("GetFeed() card count = %d, want 2 eligible featured cards", len(got))
	}
	assertInt64SliceEqual(t, []int64{got[0].ProductID, got[1].ProductID}, []int64{202, 101}, "featured feed ids")

	if got[0].PrimaryImage == "" || got[0].Brand == "" || got[0].Category == "" {
		t.Fatalf("first card = %#v, want hydrated image, brand, and category fields", got[0])
	}
	if got[1].InventoryStatus != discoverydomain.InventoryStatusInStock {
		t.Fatalf("second card inventory_status = %q, want %q", got[1].InventoryStatus, discoverydomain.InventoryStatusInStock)
	}
	if got[0].Discount <= 0 || got[0].Price <= 0 || got[0].Rating < minRating {
		t.Fatalf("first card pricing/rating fields = %#v, want hydrated discount, price, and rating", got[0])
	}
}

func TestDiscoveryServiceSuggestValidatesRequest(t *testing.T) {
	svc := NewDiscoveryService(&stubCandidateRepository{}, &stubCategoryRepository{})

	_, err := svc.Suggest(context.Background(), &discoverydomain.SuggestRequest{Prefix: "   "})
	if !errors.Is(err, discoverydomain.ErrPrefixRequired) {
		t.Fatalf("Suggest() error = %v, want %v", err, discoverydomain.ErrPrefixRequired)
	}
}

func TestDiscoveryServiceSuggestDelegatesToRepository(t *testing.T) {
	expected := []discoverydomain.Suggestion{
		{
			Text:            "electronics",
			Type:            discoverydomain.SuggestionTypeCategory,
			PopularityScore: 3.5,
		},
	}
	candidateRepo := &stubCandidateRepository{suggestResult: expected}
	svc := NewDiscoveryService(candidateRepo, &stubCategoryRepository{})

	got, err := svc.Suggest(context.Background(), &discoverydomain.SuggestRequest{Prefix: "  elec  "})
	if err != nil {
		t.Fatalf("Suggest() error = %v", err)
	}
	if !candidateRepo.suggestCalled {
		t.Fatal("expected discovery repository suggest method to be called")
	}
	if candidateRepo.suggestReq == nil || candidateRepo.suggestReq.Prefix != "elec" {
		t.Fatalf("repository suggest prefix = %#v, want %q", candidateRepo.suggestReq, "elec")
	}
	if len(got) != len(expected) || got[0].Text != expected[0].Text {
		t.Fatalf("Suggest() = %#v, want %#v", got, expected)
	}
}

func TestDiscoveryServiceTrackSearchValidatesRequest(t *testing.T) {
	svc := NewDiscoveryService(&stubCandidateRepository{}, &stubCategoryRepository{})

	_, err := svc.TrackSearch(context.Background(), &discoverydomain.SearchEvent{Query: "   "})
	if !errors.Is(err, discoverydomain.ErrQueryRequired) {
		t.Fatalf("TrackSearch() error = %v, want %v", err, discoverydomain.ErrQueryRequired)
	}
}

func TestDiscoveryServiceTrackSearchDelegatesToRepository(t *testing.T) {
	candidateRepo := &stubCandidateRepository{searchEventID: 99}
	svc := NewDiscoveryService(candidateRepo, &stubCategoryRepository{})

	eventID, err := svc.TrackSearch(context.Background(), &discoverydomain.SearchEvent{
		Query:       "  Earbuds  ",
		ResultCount: 1,
		Results: []discoverydomain.SearchResultPosition{
			{ProductID: 4, Position: 1, Score: 0.8},
		},
	})
	if err != nil {
		t.Fatalf("TrackSearch() error = %v", err)
	}
	if !candidateRepo.searchEventCalled {
		t.Fatal("expected search tracking repository method to be called")
	}
	if candidateRepo.searchEvent == nil || candidateRepo.searchEvent.NormalizedQuery != "earbuds" {
		t.Fatalf("repository search event = %#v, want normalized query earbuds", candidateRepo.searchEvent)
	}
	if eventID != 99 {
		t.Fatalf("TrackSearch() eventID = %d, want %d", eventID, 99)
	}
}

func TestDiscoveryServiceTrackSearchClickValidatesRequest(t *testing.T) {
	svc := NewDiscoveryService(&stubCandidateRepository{}, &stubCategoryRepository{})

	err := svc.TrackSearchClick(context.Background(), &discoverydomain.SearchClickEvent{
		SearchEventID: 1,
		ProductID:     0,
		Position:      1,
	})
	if !errors.Is(err, discoverydomain.ErrInvalidProductID) {
		t.Fatalf("TrackSearchClick() error = %v, want %v", err, discoverydomain.ErrInvalidProductID)
	}
}

func TestDiscoveryServiceTrackSearchClickDelegatesToRepository(t *testing.T) {
	candidateRepo := &stubCandidateRepository{}
	svc := NewDiscoveryService(candidateRepo, &stubCategoryRepository{})

	err := svc.TrackSearchClick(context.Background(), &discoverydomain.SearchClickEvent{
		SearchEventID: 10,
		ProductID:     5,
		Position:      2,
	})
	if err != nil {
		t.Fatalf("TrackSearchClick() error = %v", err)
	}
	if !candidateRepo.searchClickCalled {
		t.Fatal("expected search click repository method to be called")
	}
	if candidateRepo.searchClick == nil || candidateRepo.searchClick.Position != 2 {
		t.Fatalf("repository search click = %#v, want position 2", candidateRepo.searchClick)
	}
}
