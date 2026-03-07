package worker

import (
	"context"
	"fmt"
	"log"
	"time"
)

type metricsRepository interface {
	RefreshProductMetrics(ctx context.Context) error
	RefreshUserCategoryAffinity(ctx context.Context) error
	RefreshProductCoViews(ctx context.Context) error
}

type MetricsJobService struct {
	repo     metricsRepository
	interval time.Duration
	logger   *log.Logger
}

func NewMetricsJobService(repo metricsRepository, interval time.Duration, logger *log.Logger) *MetricsJobService {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	if logger == nil {
		logger = log.Default()
	}

	return &MetricsJobService{
		repo:     repo,
		interval: interval,
		logger:   logger,
	}
}

func (s *MetricsJobService) RunOnce(ctx context.Context) error {
	if err := s.repo.RefreshProductMetrics(ctx); err != nil {
		return fmt.Errorf("refresh product metrics: %w", err)
	}
	if err := s.repo.RefreshUserCategoryAffinity(ctx); err != nil {
		return fmt.Errorf("refresh user category affinity: %w", err)
	}
	if err := s.repo.RefreshProductCoViews(ctx); err != nil {
		return fmt.Errorf("refresh product co-views: %w", err)
	}
	return nil
}

func (s *MetricsJobService) Start(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}

	if err := s.RunOnce(ctx); err != nil {
		s.logger.Printf("metrics computation failed: %v", err)
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := s.RunOnce(ctx); err != nil {
				s.logger.Printf("metrics computation failed: %v", err)
			}
		}
	}
}
