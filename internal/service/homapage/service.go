package homepage

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"zentora-service/internal/domain/homepage"
	//pgRepo "zentora-service/internal/repository/postgres"

	"github.com/redis/go-redis/v9"
)

// ─── Cache TTLs ───────────────────────────────────────────────────────────────

const (
	// homepageCacheTTL is the TTL for the fully assembled homepage response.
	// Low enough to reflect admin changes promptly; high enough to absorb traffic spikes.
	homepageCacheTTL = 2 * time.Minute

	// sectionCacheTTL is per-section TTL when sections are cached individually.
	sectionCacheTTL = 5 * time.Minute

	// cacheKeyHomepage is the Redis key for the full homepage snapshot.
	cacheKeyHomepage = "homepage:full"

	// cacheKeySectionPrefix is the prefix for per-section keys.
	cacheKeySectionPrefix = "homepage:section:"
)

// ─── Interfaces ───────────────────────────────────────────────────────────────

type homepageRepo interface {
	ListActiveSections(ctx context.Context) ([]homepage.Section, error)
	ListSections(ctx context.Context, f homepage.ListFilter) ([]homepage.Section, error)
	GetSectionByID(ctx context.Context, id int64) (*homepage.Section, error)
	CreateSection(ctx context.Context, s *homepage.Section) error
	UpdateSection(ctx context.Context, s *homepage.Section) error
	DeleteSection(ctx context.Context, id int64) error
	ReorderSections(ctx context.Context, items []homepage.ReorderItem) error
	ToggleActive(ctx context.Context, id int64, active bool) error
}

// catalogReader is the minimal surface the service needs from the catalog repos.
// Keeps the homepage package decoupled from the full CatalogService.
type catalogReader interface {
	GetSectionProducts(ctx context.Context, sectionType homepage.SectionType, referenceID *int64, limit int) ([]homepage.SectionProduct, error)
}

type cacheClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

// ─── Service ──────────────────────────────────────────────────────────────────

type HomepageService struct {
	repo    homepageRepo
	catalog catalogReader
	cache   cacheClient
}

// NewHomepageService wires the service.
// cache is variadic so callers can omit it (falls back to no-cache mode).
func NewHomepageService(
	repo homepageRepo,
	catalog catalogReader,
	cache ...cacheClient,
) *HomepageService {
	svc := &HomepageService{
		repo:    repo,
		catalog: catalog,
	}
	if len(cache) > 0 {
		svc.cache = cache[0]
	}
	return svc
}

// ─── Public homepage ──────────────────────────────────────────────────────────

// GetHomepage returns the fully assembled homepage (all active sections + their
// products). It is the highest-traffic endpoint so it is aggressively cached.
func (s *HomepageService) GetHomepage(ctx context.Context) (*homepage.HomepageResponse, error) {
	// 1. Try full-page cache hit
	if resp, ok := s.getCachedHomepage(ctx); ok {
		return resp, nil
	}

	// 2. Load section rows (tiny table, fast query)
	sections, err := s.repo.ListActiveSections(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active sections: %w", err)
	}

	// 3. Resolve products for each section; try per-section cache first
	assembled := make([]homepage.SectionWithProducts, 0, len(sections))
	for _, sec := range sections {
		swp, err := s.resolveSectionWithProducts(ctx, sec)
		if err != nil {
			// Non-fatal: skip broken sections rather than failing the whole page
			continue
		}
		assembled = append(assembled, *swp)
	}

	resp := &homepage.HomepageResponse{
		Sections:    assembled,
		GeneratedAt: time.Now().UTC(),
	}

	// 4. Cache the full response
	s.setCachedHomepage(ctx, resp)

	return resp, nil
}

// GetSectionByID returns a single section with its products (admin + public use).
func (s *HomepageService) GetSectionByID(ctx context.Context, id int64) (*homepage.SectionWithProducts, error) {
	sec, err := s.repo.GetSectionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.resolveSectionWithProducts(ctx, *sec)
}

// resolveSectionWithProducts resolves products for one section.
// Checks per-section cache first, then falls back to the catalog reader.
func (s *HomepageService) resolveSectionWithProducts(ctx context.Context, sec homepage.Section) (*homepage.SectionWithProducts, error) {
	swp := &homepage.SectionWithProducts{Section: sec}

	// Check per-section product cache
	if products, ok := s.getCachedSectionProducts(ctx, sec.ID); ok {
		swp.Products = products
		return swp, nil
	}

	// Default product limit per section
	const defaultLimit = 20

	products, err := s.catalog.GetSectionProducts(ctx, sec.Type, sec.ReferenceID, defaultLimit)
	if err != nil {
		return nil, fmt.Errorf("get products for section %d: %w", sec.ID, err)
	}

	swp.Products = products
	s.setCachedSectionProducts(ctx, sec.ID, products)

	return swp, nil
}

// ─── Admin write operations ───────────────────────────────────────────────────

func (s *HomepageService) ListSections(ctx context.Context, f homepage.ListFilter) ([]homepage.Section, error) {
	return s.repo.ListSections(ctx, f)
}

func (s *HomepageService) CreateSection(ctx context.Context, req *homepage.CreateSectionRequest) (*homepage.Section, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	sec := &homepage.Section{
		Title:       req.Title,
		Type:        req.Type,
		ReferenceID: req.ReferenceID,
		SortOrder:   req.SortOrder,
		IsActive:    true,
	}
	if req.IsActive != nil {
		sec.IsActive = *req.IsActive
	}

	if err := s.repo.CreateSection(ctx, sec); err != nil {
		return nil, fmt.Errorf("create section: %w", err)
	}

	// Any write must bust the full homepage cache
	s.bustHomepageCache(ctx)

	return sec, nil
}

func (s *HomepageService) UpdateSection(ctx context.Context, id int64, req *homepage.UpdateSectionRequest) (*homepage.Section, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	sec, err := s.repo.GetSectionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		sec.Title = req.Title
	}
	if req.Type != nil {
		sec.Type = *req.Type
	}
	if req.ReferenceID != nil {
		sec.ReferenceID = req.ReferenceID
	}
	if req.SortOrder != nil {
		sec.SortOrder = *req.SortOrder
	}
	if req.IsActive != nil {
		sec.IsActive = *req.IsActive
	}

	if err := s.repo.UpdateSection(ctx, sec); err != nil {
		return nil, fmt.Errorf("update section: %w", err)
	}

	s.bustSectionCache(ctx, id)
	s.bustHomepageCache(ctx)

	return sec, nil
}

func (s *HomepageService) DeleteSection(ctx context.Context, id int64) error {
	if err := s.repo.DeleteSection(ctx, id); err != nil {
		return err
	}
	s.bustSectionCache(ctx, id)
	s.bustHomepageCache(ctx)
	return nil
}

func (s *HomepageService) ReorderSections(ctx context.Context, req *homepage.ReorderRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}
	if err := s.repo.ReorderSections(ctx, req.Items); err != nil {
		return fmt.Errorf("reorder sections: %w", err)
	}
	s.bustHomepageCache(ctx)
	return nil
}

func (s *HomepageService) ToggleActive(ctx context.Context, id int64, active bool) error {
	if err := s.repo.ToggleActive(ctx, id, active); err != nil {
		return err
	}
	s.bustSectionCache(ctx, id)
	s.bustHomepageCache(ctx)
	return nil
}

// ─── Cache helpers ────────────────────────────────────────────────────────────

func (s *HomepageService) getCachedHomepage(ctx context.Context) (*homepage.HomepageResponse, bool) {
	if s.cache == nil {
		return nil, false
	}
	payload, err := s.cache.Get(ctx, cacheKeyHomepage).Bytes()
	if err != nil {
		return nil, false
	}
	var resp homepage.HomepageResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		return nil, false
	}
	return &resp, true
}

func (s *HomepageService) setCachedHomepage(ctx context.Context, resp *homepage.HomepageResponse) {
	if s.cache == nil || resp == nil {
		return
	}
	payload, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_ = s.cache.Set(ctx, cacheKeyHomepage, payload, homepageCacheTTL).Err()
}

func (s *HomepageService) getCachedSectionProducts(ctx context.Context, sectionID int64) ([]homepage.SectionProduct, bool) {
	if s.cache == nil {
		return nil, false
	}
	key := sectionCacheKey(sectionID)
	payload, err := s.cache.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	var products []homepage.SectionProduct
	if err := json.Unmarshal(payload, &products); err != nil {
		return nil, false
	}
	return products, true
}

func (s *HomepageService) setCachedSectionProducts(ctx context.Context, sectionID int64, products []homepage.SectionProduct) {
	if s.cache == nil || len(products) == 0 {
		return
	}
	payload, err := json.Marshal(products)
	if err != nil {
		return
	}
	_ = s.cache.Set(ctx, sectionCacheKey(sectionID), payload, sectionCacheTTL).Err()
}

// bustHomepageCache invalidates the full-page snapshot.
// Called after every write that can change what the homepage looks like.
func (s *HomepageService) bustHomepageCache(ctx context.Context) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Del(ctx, cacheKeyHomepage).Err()
}

// bustSectionCache invalidates the per-section product cache.
func (s *HomepageService) bustSectionCache(ctx context.Context, sectionID int64) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Del(ctx, sectionCacheKey(sectionID)).Err()
}

func sectionCacheKey(id int64) string {
	return cacheKeySectionPrefix + strconv.FormatInt(id, 10)
}

// buildSectionProductsCacheKey builds a deterministic key that encodes the
// factors that affect which products are returned for a section.
// Currently only sectionID matters; extend here if filters are added later.
func buildSectionProductsCacheKey(sectionID int64, sectionType homepage.SectionType, referenceID *int64) string {
	parts := []string{
		cacheKeySectionPrefix,
		strconv.FormatInt(sectionID, 10),
		string(sectionType),
	}
	if referenceID != nil {
		parts = append(parts, "ref="+strconv.FormatInt(*referenceID, 10))
	}
	return strings.Join(parts, ":")
}