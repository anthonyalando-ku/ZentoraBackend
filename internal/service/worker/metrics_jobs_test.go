package worker

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"
)

type stubMetricsRepository struct {
	calls []string

	productMetricsErr error
	affinityErr       error
	coViewsErr        error
}

func (s *stubMetricsRepository) RefreshProductMetrics(context.Context) error {
	s.calls = append(s.calls, "product_metrics")
	return s.productMetricsErr
}

func (s *stubMetricsRepository) RefreshUserCategoryAffinity(context.Context) error {
	s.calls = append(s.calls, "user_category_affinity")
	return s.affinityErr
}

func (s *stubMetricsRepository) RefreshProductCoViews(context.Context) error {
	s.calls = append(s.calls, "product_co_views")
	return s.coViewsErr
}

func TestMetricsJobServiceRunOnceRunsAllRefreshesInOrder(t *testing.T) {
	repo := &stubMetricsRepository{}
	svc := NewMetricsJobService(repo, time.Minute, log.New(&bytes.Buffer{}, "", 0))

	if err := svc.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}

	want := []string{"product_metrics", "user_category_affinity", "product_co_views"}
	if len(repo.calls) != len(want) {
		t.Fatalf("RunOnce() call count = %d, want %d", len(repo.calls), len(want))
	}
	for i := range want {
		if repo.calls[i] != want[i] {
			t.Fatalf("RunOnce() call %d = %q, want %q", i, repo.calls[i], want[i])
		}
	}
}

func TestMetricsJobServiceRunOnceStopsOnFirstError(t *testing.T) {
	repo := &stubMetricsRepository{affinityErr: errors.New("boom")}
	svc := NewMetricsJobService(repo, time.Minute, log.New(&bytes.Buffer{}, "", 0))

	err := svc.RunOnce(context.Background())
	if err == nil || !strings.Contains(err.Error(), "refresh user category affinity") {
		t.Fatalf("RunOnce() error = %v, want wrapped affinity error", err)
	}

	want := []string{"product_metrics", "user_category_affinity"}
	if len(repo.calls) != len(want) {
		t.Fatalf("RunOnce() call count = %d, want %d", len(repo.calls), len(want))
	}
	for i := range want {
		if repo.calls[i] != want[i] {
			t.Fatalf("RunOnce() call %d = %q, want %q", i, repo.calls[i], want[i])
		}
	}
}

func TestMetricsJobServiceStartReturnsNilWhenContextCancelled(t *testing.T) {
	repo := &stubMetricsRepository{}
	svc := NewMetricsJobService(repo, time.Hour, log.New(&bytes.Buffer{}, "", 0))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := svc.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if len(repo.calls) != 0 {
		t.Fatalf("Start() calls = %#v, want no work when context is already cancelled", repo.calls)
	}
}
