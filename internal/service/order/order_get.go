package orderusecase
import (
	"context"
	"errors"
	"fmt"
	"zentora-service/internal/domain/order"
)


func (s *Service) GetOrderByID(ctx context.Context, id int64) (*order.Order, error) {
	o, err := s.orders.GetOrderByID(ctx, id)	
	if err != nil {
		if errors.Is(err, order.ErrNotFound) {
			return nil, order.ErrNotFound
		}	
		return nil, fmt.Errorf("get order by id: %w", err)
	}
	return o, nil
}

func (r *Service) ListOrders(ctx context.Context, f order.ListFilter) ([]order.Order, int64, error) {
	return r.orders.ListOrders(ctx, f)
}
