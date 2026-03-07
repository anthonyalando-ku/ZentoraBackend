package discovery

import (
	"context"
	"fmt"

	categorydomain "zentora-service/internal/domain/category"
	discoverydomain "zentora-service/internal/domain/discovery"
)

type candidateRepository interface {
	GetFeedCandidates(ctx context.Context, req *discoverydomain.FeedRequest) ([]discoverydomain.Candidate, error)
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
