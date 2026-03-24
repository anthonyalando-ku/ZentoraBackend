package orderusecase

import (
	"context"
	"errors"
	"fmt"

	"zentora-service/internal/domain/order"
)

// Admin: get by order number
func (s *Service) GetOrderByNumber(ctx context.Context, orderNumber string) (*order.Order, error) {
	o, err := s.orders.GetOrderByNumber(ctx, orderNumber)
	if err != nil {
		if errors.Is(err, order.ErrNotFound) {
			return nil, order.ErrNotFound
		}
		if errors.Is(err, order.ErrInvalidInput) {
			return nil, order.ErrInvalidInput
		}
		return nil, fmt.Errorf("get order by number: %w", err)
	}
	return o, nil
}

// Admin: update order status (mark completed, cancelled, shipped, delivered, etc.)
func (s *Service) UpdateOrderStatus(ctx context.Context, id int64, status order.OrderStatus) (*order.Order, error) {
	// basic validation + basic transition rules (optional)
	switch status {
	case order.OrderStatusPending, order.OrderStatusCompleted, order.OrderStatusCancelled, order.OrderStatusShipped, order.OrderStatusDelivered:
		// ok
	default:
		return nil, order.ErrInvalidInput
	}

	o, err := s.orders.UpdateOrderStatus(ctx, id, status)
	if err != nil {
		if errors.Is(err, order.ErrNotFound) {
			return nil, order.ErrNotFound
		}
		if errors.Is(err, order.ErrInvalidInput) {
			return nil, order.ErrInvalidInput
		}
		return nil, fmt.Errorf("update order status: %w", err)
	}
	return o, nil
}

// Admin: stats for dashboard
func (s *Service) GetOrderStats(ctx context.Context) (*order.OrderStatsResponse, error) {
	stats, err := s.orders.OrderStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get order stats: %w", err)
	}
	return stats, nil
}