package discovery

import (
	"context"
	"errors"
	"testing"

	categorydomain "zentora-service/internal/domain/category"
	discoverydomain "zentora-service/internal/domain/discovery"
)

type stubCandidateRepository struct {
	called        bool
	req           *discoverydomain.FeedRequest
	result        []discoverydomain.Candidate
	err           error
	suggestCalled bool
	suggestReq    *discoverydomain.SuggestRequest
	suggestResult []discoverydomain.Suggestion
	suggestErr    error
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
