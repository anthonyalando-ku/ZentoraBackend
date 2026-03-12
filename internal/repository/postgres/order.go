package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"zentora-service/internal/domain/order"
	orderrepo "zentora-service/internal/repository/order"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{db: db}
}

var _ orderrepo.Repository = (*OrderRepository)(nil)

func (r *OrderRepository) CreateOrderTx(ctx context.Context, tx pgx.Tx, o *order.Order) error {
	if tx != nil {
		return r.createTx(ctx, tx, o)
	}

	localTx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = localTx.Rollback(ctx) }()
	if err := r.createTx(ctx, localTx, o); err != nil {
		return err
	}
	return localTx.Commit(ctx)
}

func (r *OrderRepository) createTx(ctx context.Context, tx pgx.Tx, o *order.Order) error {
	const insOrder = `
		INSERT INTO orders (
			user_id, cart_id, order_number, status, subtotal, discount_amount, tax_amount, shipping_fee, total_amount, currency, shipping_method_id,
			shipping_full_name, shipping_phone, shipping_country, shipping_county, shipping_city, shipping_area, shipping_postal_code,
			shipping_address_line_1, shipping_address_line_2
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,
			$12,$13,$14,$15,$16,$17,$18,$19,$20
		)
		RETURNING id, created_at, updated_at
	`

	if err := tx.QueryRow(ctx, insOrder,
		o.UserID, o.CartID, o.OrderNumber, o.Status,
		o.Subtotal, o.DiscountAmount, o.TaxAmount, o.ShippingFee, o.TotalAmount, o.Currency, o.ShippingMethodID,
		o.Shipping.FullName, o.Shipping.Phone, o.Shipping.Country, o.Shipping.County, o.Shipping.City, o.Shipping.Area, o.Shipping.PostalCode,
		o.Shipping.AddressLine1, o.Shipping.AddressLine2,
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	const insItem = `
		INSERT INTO order_items (
			order_id, product_id, variant_id,
			product_name, product_slug, variant_sku, variant_name, image_url,
			unit_price, quantity, discount_amount, tax_rate, total_price, currency
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14
		)
		RETURNING id
	`
	for i := range o.Items {
		it := &o.Items[i]
		it.OrderID = o.ID

		if err := tx.QueryRow(ctx, insItem,
			it.OrderID, it.ProductID, it.VariantID,
			it.ProductName, it.ProductSlug, it.VariantSKU, it.VariantName, it.ImageURL,
			it.UnitPrice, it.Quantity, it.DiscountAmount, it.TaxRate, it.TotalPrice, it.Currency,
		).Scan(&it.ID); err != nil {
			return fmt.Errorf("insert order item: %w", err)
		}
	}

	return nil
}

func (r *OrderRepository) GetOrderByID(ctx context.Context, id int64) (*order.Order, error) {
	if id <= 0 {
		return nil, order.ErrInvalidInput
	}

	const q = `
		SELECT
			id, user_id, cart_id, order_number, status,
			subtotal, discount_amount, tax_amount, shipping_fee, total_amount, currency, shipping_method_id,
			shipping_full_name, shipping_phone, shipping_country, shipping_county, shipping_city, shipping_area, shipping_postal_code,
			shipping_address_line_1, shipping_address_line_2,
			created_at, updated_at
		FROM orders
		WHERE id = $1
	`

	var o order.Order
	if err := r.db.QueryRow(ctx, q, id).Scan(
		&o.ID, &o.UserID, &o.CartID, &o.OrderNumber, &o.Status,
		&o.Subtotal, &o.DiscountAmount, &o.TaxAmount, &o.ShippingFee, &o.TotalAmount, &o.Currency, &o.ShippingMethodID,
		&o.Shipping.FullName, &o.Shipping.Phone, &o.Shipping.Country, &o.Shipping.County, &o.Shipping.City, &o.Shipping.Area, &o.Shipping.PostalCode,
		&o.Shipping.AddressLine1, &o.Shipping.AddressLine2,
		&o.CreatedAt, &o.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, order.ErrNotFound
		}
		return nil, fmt.Errorf("get order: %w", err)
	}

	items, err := r.getOrderItems(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return &o, nil
}

func (r *OrderRepository) getOrderItems(ctx context.Context, orderID int64) ([]order.OrderItem, error) {
	const q = `
		SELECT
			id, order_id, product_id, variant_id,
			product_name, product_slug, variant_sku, variant_name, image_url,
			unit_price, quantity, discount_amount, tax_rate, total_price, currency
		FROM order_items
		WHERE order_id = $1
		ORDER BY id ASC
	`

	rows, err := r.db.Query(ctx, q, orderID)
	if err != nil {
		return nil, fmt.Errorf("list order items: %w", err)
	}
	defer rows.Close()

	out := make([]order.OrderItem, 0, 8)
	for rows.Next() {
		var it order.OrderItem
		if err := rows.Scan(
			&it.ID, &it.OrderID, &it.ProductID, &it.VariantID,
			&it.ProductName, &it.ProductSlug, &it.VariantSKU, &it.VariantName, &it.ImageURL,
			&it.UnitPrice, &it.Quantity, &it.DiscountAmount, &it.TaxRate, &it.TotalPrice, &it.Currency,
		); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("order items rows: %w", err)
	}
	return out, nil
}

// ListOrders implements a dynamic filter usable by user and admin.
// It returns "order headers" (no items) + total count.
// For "order details", use GetOrderByID.
func (r *OrderRepository) ListOrders(ctx context.Context, f order.ListFilter) ([]order.Order, int64, error) {
	where, args := buildOrdersWhere(f)

	// count
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM orders`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	sortCol := "created_at"
	switch f.SortBy {
	case "id", "created_at", "total_amount":
		sortCol = f.SortBy
	}
	sortDir := "DESC"
	if !f.SortDesc {
		sortDir = "ASC"
	}

	args = append(args, limit, offset)
	limitPos := len(args) - 1
	offsetPos := len(args)

	q := fmt.Sprintf(`
		SELECT
			id, user_id, cart_id, order_number, status,
			subtotal, discount_amount, tax_amount, shipping_fee, total_amount, currency, shipping_method_id,
			shipping_full_name, shipping_phone, shipping_country, shipping_county, shipping_city, shipping_area, shipping_postal_code,
			shipping_address_line_1, shipping_address_line_2,
			created_at, updated_at
		FROM orders
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, sortCol, sortDir, limitPos, offsetPos)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	out := make([]order.Order, 0, limit)
	for rows.Next() {
		var o order.Order
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.CartID, &o.OrderNumber, &o.Status,
			&o.Subtotal, &o.DiscountAmount, &o.TaxAmount, &o.ShippingFee, &o.TotalAmount, &o.Currency, &o.ShippingMethodID,
			&o.Shipping.FullName, &o.Shipping.Phone, &o.Shipping.Country, &o.Shipping.County, &o.Shipping.City, &o.Shipping.Area, &o.Shipping.PostalCode,
			&o.Shipping.AddressLine1, &o.Shipping.AddressLine2,
			&o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan order: %w", err)
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list orders rows: %w", err)
	}

	return out, total, nil
}

func buildOrdersWhere(f order.ListFilter) (string, []any) {
	clauses := make([]string, 0, 8)
	args := make([]any, 0, 8)
	i := 1

	add := func(expr string, v any) {
		clauses = append(clauses, fmt.Sprintf(expr, i))
		args = append(args, v)
		i++
	}

	if f.OrderID != nil && *f.OrderID > 0 {
		add("id = $%d", *f.OrderID)
	}
	if f.OrderNumber != nil && strings.TrimSpace(*f.OrderNumber) != "" {
		add("order_number = $%d", strings.TrimSpace(*f.OrderNumber))
	}
	if f.UserID != nil && *f.UserID > 0 {
		add("user_id = $%d", *f.UserID)
	}
	if f.CartID != nil && *f.CartID > 0 {
		add("cart_id = $%d", *f.CartID)
	}
	if len(f.Statuses) > 0 {
		// status = ANY($n)
		clauses = append(clauses, fmt.Sprintf("status = ANY($%d)", i))
		args = append(args, f.Statuses)
		i++
	}
	if f.CreatedFrom != nil {
		add("created_at >= $%d", *f.CreatedFrom)
	}
	if f.CreatedTo != nil {
		add("created_at <= $%d", *f.CreatedTo)
	}

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}