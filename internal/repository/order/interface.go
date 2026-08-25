package orderrepo

import (
	"context"
	"time"

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

	GetOrderByNumber(ctx context.Context, orderNumber string) (*order.Order, error)
	UpdateOrderStatus(ctx context.Context, id int64, status order.OrderStatus) (*order.Order, error)
	OrderStats(ctx context.Context) (*order.OrderStatsResponse, error)

	// ListPendingOrdersNeedingReminder returns registered-user orders still
	// pending after olderThan that haven't had a reminder sent yet.
	ListPendingOrdersNeedingReminder(ctx context.Context, olderThan time.Duration) ([]order.PendingReminderCandidate, error)
	// MarkReminderSent records that a pending-order reminder was emailed, so
	// the next worker tick does not notify the customer again for it.
	MarkReminderSent(ctx context.Context, orderID int64) error
}