package discovery

import (
	"context"
	"errors"
	"testing"

	categorydomain "zentora-service/internal/domain/category"
	discoverydomain "zentora-service/internal/domain/discovery"
)

type stubCandidateRepository struct {
	called            bool
	req               *discoverydomain.FeedRequest
	result            []discoverydomain.Candidate
	err               error
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
	return s.result, s.err
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

func TestDiscoveryServiceGetFeedCandidatesValidatesRequest(t *testing.T) {
	svc := NewDiscoveryService(&stubCandidateRepository{}, &stubCategoryRepository{})

	_, err := svc.GetFeedCandidates(context.Background(), nil)
	if !errors.Is(err, discoverydomain.ErrInvalidRequest) {
		t.Fatalf("GetFeedCandidates() error = %v, want %v", err, discoverydomain.ErrInvalidRequest)
	}
}

func TestDiscoveryServiceGetFeedCandidatesChecksCategoryExists(t *testing.T) {
	categoryID := int64(42)
	candidateRepo := &stubCandidateRepository{}
	categoryRepo := &stubCategoryRepository{err: categorydomain.ErrNotFound}
	svc := NewDiscoveryService(candidateRepo, categoryRepo)

	_, err := svc.GetFeedCandidates(context.Background(), &discoverydomain.FeedRequest{
		FeedType:   discoverydomain.FeedCategory,
		CategoryID: &categoryID,
	})
	if !errors.Is(err, categorydomain.ErrNotFound) {
		t.Fatalf("GetFeedCandidates() error = %v, want %v", err, categorydomain.ErrNotFound)
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
	expected := []discoverydomain.Candidate{
		{ProductID: 101, Signals: map[string]float64{"category_score": 1}},
	}
	candidateRepo := &stubCandidateRepository{result: expected}
	categoryRepo := &stubCategoryRepository{category: &categorydomain.Category{ID: categoryID}}
	svc := NewDiscoveryService(candidateRepo, categoryRepo)

	got, err := svc.GetFeedCandidates(context.Background(), &discoverydomain.FeedRequest{
		FeedType:   discoverydomain.FeedCategory,
		CategoryID: &categoryID,
	})
	if err != nil {
		t.Fatalf("GetFeedCandidates() error = %v", err)
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
	if len(got) != len(expected) || got[0].ProductID != expected[0].ProductID {
		t.Fatalf("GetFeedCandidates() = %#v, want %#v", got, expected)
	}
}

func TestDiscoveryServiceGetFeedCandidatesSkipsCategoryLookupForNonCategoryFeed(t *testing.T) {
	expected := []discoverydomain.Candidate{
		{ProductID: 1, Signals: map[string]float64{"trending_score": 5}},
	}
	candidateRepo := &stubCandidateRepository{result: expected}
	categoryRepo := &stubCategoryRepository{}
	svc := NewDiscoveryService(candidateRepo, categoryRepo)

	got, err := svc.GetFeedCandidates(context.Background(), &discoverydomain.FeedRequest{
		FeedType: discoverydomain.FeedTrending,
	})
	if err != nil {
		t.Fatalf("GetFeedCandidates() error = %v", err)
	}
	if categoryRepo.called {
		t.Fatal("expected category repository not to be called")
	}
	if !candidateRepo.called {
		t.Fatal("expected discovery repository to be called")
	}
	if len(got) != len(expected) || got[0].ProductID != expected[0].ProductID {
		t.Fatalf("GetFeedCandidates() = %#v, want %#v", got, expected)
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
