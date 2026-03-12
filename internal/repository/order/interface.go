package orderrepo

import (
	"context"

	"zentora-service/internal/domain/order"

	"github.com/jackc/pgx/v5"
)

type Repository interface {
	CreateOrderTx(ctx context.Context, tx pgx.Tx, o *order.Order) error

	// GetOrderByID returns order + items.
	GetOrderByID(ctx context.Context, id int64) (*order.Order, error)

	// ListOrders returns orders (with items optional; here we return without items for listing),
	// and a total count for pagination.
	ListOrders(ctx context.Context, f order.ListFilter) ([]order.Order, int64, error)
}