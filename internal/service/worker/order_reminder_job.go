package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"zentora-service/internal/domain/order"
)

type pendingReminderRepo interface {
	ListPendingOrdersNeedingReminder(ctx context.Context, olderThan time.Duration) ([]order.PendingReminderCandidate, error)
	MarkReminderSent(ctx context.Context, id int64) error
}

type pendingReminderMailer interface {
	SendPendingOrderReminder(toEmail, customerName, orderNumber string, totalAmount float64, currency string, placedAt time.Time) error
}

// OrderReminderJobService periodically emails customers whose orders have
// stayed in "pending" status for too long, nudging them to log in and
// complete, update, or cancel the order.
type OrderReminderJobService struct {
	repo         pendingReminderRepo
	mailer       pendingReminderMailer
	pendingAfter time.Duration
	interval     time.Duration
	logger       *log.Logger
}

func NewOrderReminderJobService(
	repo pendingReminderRepo,
	mailer pendingReminderMailer,
	pendingAfter time.Duration,
	interval time.Duration,
	logger *log.Logger,
) *OrderReminderJobService {
	if pendingAfter <= 0 {
		pendingAfter = 24 * time.Hour
	}
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	if logger == nil {
		logger = log.Default()
	}

	return &OrderReminderJobService{
		repo:         repo,
		mailer:       mailer,
		pendingAfter: pendingAfter,
		interval:     interval,
		logger:       logger,
	}
}

func (s *OrderReminderJobService) RunOnce(ctx context.Context) error {
	candidates, err := s.repo.ListPendingOrdersNeedingReminder(ctx, s.pendingAfter)
	if err != nil {
		return fmt.Errorf("list pending orders needing reminder: %w", err)
	}

	for _, c := range candidates {
		if err := s.mailer.SendPendingOrderReminder(c.CustomerEmail, c.CustomerName, c.OrderNumber, c.TotalAmount, c.Currency, c.CreatedAt); err != nil {
			s.logger.Printf("pending order reminder failed: order=%s email=%s err=%v", c.OrderNumber, c.CustomerEmail, err)
			continue // leave reminder_sent_at unset so it's retried next tick
		}
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
