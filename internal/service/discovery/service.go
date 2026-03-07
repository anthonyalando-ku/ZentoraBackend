package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	categorydomain "zentora-service/internal/domain/category"
	discoverydomain "zentora-service/internal/domain/discovery"

	"github.com/redis/go-redis/v9"
)

type candidateRepository interface {
	GetFeedCandidates(ctx context.Context, req *discoverydomain.FeedRequest) ([]discoverydomain.Candidate, error)
	HydrateProductCards(ctx context.Context, productIDs []int64) ([]discoverydomain.ProductCard, error)
	Suggest(ctx context.Context, req *discoverydomain.SuggestRequest) ([]discoverydomain.Suggestion, error)
	TrackSearch(ctx context.Context, event *discoverydomain.SearchEvent) (int64, error)
	TrackSearchClick(ctx context.Context, event *discoverydomain.SearchClickEvent) error
}

type categoryRepository interface {
	GetCategoryByID(ctx context.Context, id int64) (*categorydomain.Category, error)
}

type DiscoveryService struct {
	discoveryRepo candidateRepository
	categoryRepo  categoryRepository
	cache         cacheClient
}

type cacheClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
}

const feedCacheTTL = 2 * time.Minute

func NewDiscoveryService(discoveryRepo candidateRepository, categoryRepo categoryRepository, cache ...cacheClient) *DiscoveryService {
	svc := &DiscoveryService{
		discoveryRepo: discoveryRepo,
		categoryRepo:  categoryRepo,
	}
	if len(cache) > 0 {
		svc.cache = cache[0]
	}
	return svc
}

func (s *DiscoveryService) GetFeed(ctx context.Context, req *discoverydomain.FeedRequest) ([]discoverydomain.ProductCard, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if req.FeedType == discoverydomain.FeedCategory {
		if _, err := s.categoryRepo.GetCategoryByID(ctx, *req.CategoryID); err != nil {
			return nil, fmt.Errorf("get category: %w", err)
		}
	}

	if cached, ok := s.getCachedFeed(ctx, req); ok {
		return cached, nil
	}

	candidates, err := s.discoveryRepo.GetFeedCandidates(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get discovery feed candidates: %w", err)
	}
	productIDs := make([]int64, 0, len(candidates))
	for _, candidate := range candidates {
		productIDs = append(productIDs, candidate.ProductID)
	}

	cards, err := s.discoveryRepo.HydrateProductCards(ctx, productIDs)
	if err != nil {
		return nil, fmt.Errorf("hydrate discovery feed: %w", err)
	}
	s.storeCachedFeed(ctx, req, cards)
	return cards, nil
}

func (s *DiscoveryService) Suggest(ctx context.Context, req *discoverydomain.SuggestRequest) ([]discoverydomain.Suggestion, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	suggestions, err := s.discoveryRepo.Suggest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("suggest discovery terms: %w", err)
	}
	return suggestions, nil
}

func (s *DiscoveryService) TrackSearch(ctx context.Context, event *discoverydomain.SearchEvent) (int64, error) {
	if err := event.Validate(); err != nil {
		return 0, err
	}

	eventID, err := s.discoveryRepo.TrackSearch(ctx, event)
	if err != nil {
		return 0, fmt.Errorf("track search: %w", err)
	}
	return eventID, nil
}

func (s *DiscoveryService) TrackSearchClick(ctx context.Context, event *discoverydomain.SearchClickEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}

	if err := s.discoveryRepo.TrackSearchClick(ctx, event); err != nil {
		return fmt.Errorf("track search click: %w", err)
	}
	return nil
}

func (s *DiscoveryService) getCachedFeed(ctx context.Context, req *discoverydomain.FeedRequest) ([]discoverydomain.ProductCard, bool) {
	if s.cache == nil || !isCacheableFeed(req) {
		return nil, false
	}

	key := buildFeedCacheKey(req)
	payload, err := s.cache.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}

	var cards []discoverydomain.ProductCard
	if err := json.Unmarshal(payload, &cards); err != nil {
		return nil, false
	}
	return cards, true
}

func (s *DiscoveryService) storeCachedFeed(ctx context.Context, req *discoverydomain.FeedRequest, cards []discoverydomain.ProductCard) {
	if s.cache == nil || !isCacheableFeed(req) || len(cards) == 0 {
		return
	}

	payload, err := json.Marshal(cards)
	if err != nil {
		return
	}
	_ = s.cache.Set(ctx, buildFeedCacheKey(req), payload, feedCacheTTL).Err()
}

func isCacheableFeed(req *discoverydomain.FeedRequest) bool {
	switch req.FeedType {
	case discoverydomain.FeedTrending,
		discoverydomain.FeedBestSellers,
		discoverydomain.FeedDeals,
		discoverydomain.FeedNewArrivals,
		discoverydomain.FeedHighlyRated,
		discoverydomain.FeedMostWishlisted,
		discoverydomain.FeedFeatured:
		return true
	case discoverydomain.FeedRecommended:
		return req.UserID != nil
	default:
		return false
	}
}

func buildFeedCacheKey(req *discoverydomain.FeedRequest) string {
	parts := []string{
		"discovery:feed",
		string(req.FeedType),
		"limit=" + strconv.Itoa(req.Limit),
	}
	if req.UserID != nil {
		parts = append(parts, "user="+strconv.FormatInt(*req.UserID, 10))
	}
	if req.SessionID != nil {
		parts = append(parts, "session="+*req.SessionID)
	}
	if req.CategoryID != nil {
		parts = append(parts, "category="+strconv.FormatInt(*req.CategoryID, 10))
	}
	if req.Query != nil {
		parts = append(parts, "query="+strings.ToLower(*req.Query))
	}

	appendIDs := func(label string, values []int64) {
		if len(values) == 0 {
			return
		}
		copied := append([]int64(nil), values...)
		sort.Slice(copied, func(i, j int) bool { return copied[i] < copied[j] })
		encoded := make([]string, 0, len(copied))
		for _, value := range copied {
			encoded = append(encoded, strconv.FormatInt(value, 10))
		}
		parts = append(parts, label+"="+strings.Join(encoded, ","))
	}

	appendIDs("brands", req.Filters.BrandIDs)
	appendIDs("tags", req.Filters.TagIDs)
	if req.Filters.PriceMin != nil {
		parts = append(parts, "price_min="+strconv.FormatFloat(*req.Filters.PriceMin, 'f', -1, 64))
	}
	if req.Filters.PriceMax != nil {
		parts = append(parts, "price_max="+strconv.FormatFloat(*req.Filters.PriceMax, 'f', -1, 64))
	}
	if req.Filters.MinRating != nil {
		parts = append(parts, "min_rating="+strconv.FormatFloat(*req.Filters.MinRating, 'f', -1, 64))
	}
	if req.Filters.DiscountOnly {
		parts = append(parts, "discount_only=true")
	}
	if req.Filters.InStockOnly {
		parts = append(parts, "in_stock_only=true")
	}
	return strings.Join(parts, ":")
}
