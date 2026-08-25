package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"zentora-service/internal/domain/order"
)

type pendingReminderRepo interface {
	ListPendingOrdersNeedingReminder(ctx context.Context, olderThan, renotifyAfter time.Duration) ([]order.PendingReminderCandidate, error)
	MarkReminderSent(ctx context.Context, id int64) error
}

type pendingReminderMailer interface {
	SendPendingOrdersAdminReminder(candidates []order.PendingReminderCandidate) error
}

// OrderReminderJobService periodically emails the store admin a digest of
// orders that have stayed "pending" too long, so they can log in and
// complete or update each one. It never emails the customer who placed it.
type OrderReminderJobService struct {
	repo          pendingReminderRepo
	mailer        pendingReminderMailer
	pendingAfter  time.Duration
	renotifyAfter time.Duration
	interval      time.Duration
	logger        *log.Logger
}

func NewOrderReminderJobService(
	repo pendingReminderRepo,
	mailer pendingReminderMailer,
	pendingAfter time.Duration,
	renotifyAfter time.Duration,
	interval time.Duration,
	logger *log.Logger,
) *OrderReminderJobService {
	if pendingAfter <= 0 {
		pendingAfter = 24 * time.Hour
	}
	if renotifyAfter <= 0 {
		renotifyAfter = 24 * time.Hour
	}
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	if logger == nil {
		logger = log.Default()
	}

	return &OrderReminderJobService{
		repo:          repo,
		mailer:        mailer,
		pendingAfter:  pendingAfter,
		renotifyAfter: renotifyAfter,
		interval:      interval,
		logger:        logger,
	}
}

func (s *OrderReminderJobService) RunOnce(ctx context.Context) error {
	candidates, err := s.repo.ListPendingOrdersNeedingReminder(ctx, s.pendingAfter, s.renotifyAfter)
	if err != nil {
		return fmt.Errorf("list pending orders needing reminder: %w", err)
	}
	if len(candidates) == 0 {
		return nil
	}

	if err := s.mailer.SendPendingOrdersAdminReminder(candidates); err != nil {
		s.logger.Printf("pending orders admin reminder failed: count=%d err=%v", len(candidates), err)
		return nil // leave reminder_sent_at unset so it's retried next tick
	}

	for _, c := range candidates {
		if err := s.repo.MarkReminderSent(ctx, c.OrderID); err != nil {
			s.logger.Printf("failed to mark reminder sent: order=%s err=%v", c.OrderNumber, err)
		}
	}

	return nil
}

func (s *OrderReminderJobService) Start(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}

	if err := s.RunOnce(ctx); err != nil {
		s.logger.Printf("pending order reminder run failed: %v", err)
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := s.RunOnce(ctx); err != nil {
				s.logger.Printf("pending order reminder run failed: %v", err)
			}
		}
	}
}
