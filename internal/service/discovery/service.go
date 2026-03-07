package discovery

import (
	"context"
	"fmt"

	categorydomain "zentora-service/internal/domain/category"
	discoverydomain "zentora-service/internal/domain/discovery"
)

type candidateRepository interface {
	GetFeedCandidates(ctx context.Context, req *discoverydomain.FeedRequest) ([]discoverydomain.Candidate, error)
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
}

func NewDiscoveryService(discoveryRepo candidateRepository, categoryRepo categoryRepository) *DiscoveryService {
	return &DiscoveryService{
		discoveryRepo: discoveryRepo,
		categoryRepo:  categoryRepo,
	}
}

func (s *DiscoveryService) GetFeedCandidates(ctx context.Context, req *discoverydomain.FeedRequest) ([]discoverydomain.Candidate, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if req.FeedType == discoverydomain.FeedCategory {
		if _, err := s.categoryRepo.GetCategoryByID(ctx, *req.CategoryID); err != nil {
			return nil, fmt.Errorf("get category: %w", err)
		}
	}

	candidates, err := s.discoveryRepo.GetFeedCandidates(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get discovery feed candidates: %w", err)
	}
	return candidates, nil
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
