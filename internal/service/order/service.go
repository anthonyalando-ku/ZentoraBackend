package orderusecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"zentora-service/internal/domain/cart"
	"zentora-service/internal/domain/discount"
	"zentora-service/internal/domain/order"
	"zentora-service/internal/domain/product"
	"zentora-service/internal/domain/user"
	"zentora-service/internal/domain/variant"
	"zentora-service/internal/repository/postgres"
	orderrepo "zentora-service/internal/repository/order"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CartRepo interface {
	GetActiveCartWithItemsForUser(ctx context.Context, userID int64) (*cart.Cart, error)
	ClearCart(ctx context.Context, cartID int64) error
	MarkCartConverted(ctx context.Context, cartID int64) error
}

type ProductRepo interface {
	GetProductByID(ctx context.Context, id int64) (*product.Product, error)
}

type VariantRepo interface {
	GetVariantByID(ctx context.Context, id int64) (*variant.Variant, error)
}

type InventoryRepo interface {
	GetItemsByVariant(ctx context.Context, variantID int64) ([]any, error) // not used
	GetStockSummary(ctx context.Context, variantID int64) (any, error)     // not used
	Reserve(ctx context.Context, tx pgx.Tx, variantID, locationID int64, qty int) error
	AdjustAvailable(ctx context.Context, tx pgx.Tx, variantID, locationID int64, delta int) error
}

type AddressRepo interface {
	GetAddressByID(ctx context.Context, id int64) (*user.Address, error)
	ListAddressesByUser(ctx context.Context, userID int64) ([]user.Address, error)
}

type DiscountRepo interface {
	// For product discount resolution, your product repo already links discount_targets.
	// Here we only record redemption if needed.
	RecordRedemption(ctx context.Context, tx pgx.Tx, red *discount.DiscountRedemption) error
}

type Service struct {
	db            *pgxpool.Pool
	orders        orderrepo.Repository
	carts         CartRepo
	products      ProductRepo
	variants      VariantRepo
	inventory     *postgres.InventoryRepository
	addresses     AddressRepo
	discounts     DiscountRepo
}

func NewService(
	db *pgxpool.Pool,
	orders orderrepo.Repository,
	carts CartRepo,
	products ProductRepo,
	variants VariantRepo,
	inventory *postgres.InventoryRepository,
	addresses AddressRepo,
	discounts DiscountRepo,
) *Service {
	return &Service{
		db:        db,
		orders:    orders,
		carts:     carts,
		products:  products,
		variants:  variants,
		inventory: inventory,
		addresses: addresses,
		discounts: discounts,
	}
}

func genOrderNumber() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "ZNT-" + hex.EncodeToString(b)
}

func (s *Service) CreateGuestOrder(ctx context.Context, req *order.CreateGuestOrderRequest) (*order.Order, error) {
	if req == nil || len(req.Items) == 0 {
		return nil, order.ErrInvalidInput
	}
	if req.Shipping.FullName == "" || req.Shipping.Phone == "" || req.Shipping.Country == "" || req.Shipping.City == "" || req.Shipping.AddressLine1 == "" {
		return nil, order.ErrInvalidInput
	}

	pm := order.PaymentPayOnDelivery
	if req.PaymentMethod != nil {
		pm = *req.PaymentMethod
	}
	_ = pm // stored later when payments module is implemented

	return s.createOrderFromItems(ctx, nil, nil, req.Items, req.Shipping)
}

func (s *Service) CreateUserOrder(ctx context.Context, userID int64, req *order.CreateUserOrderRequest) (*order.Order, error) {
	if userID <= 0 || req == nil {
		return nil, order.ErrInvalidInput
	}

	ship, err := s.resolveShippingForUser(ctx, userID, req.AddressID)
	if err != nil {
		return nil, err
	}

	// Scenario A: order from cart
	if req.CartID != nil && *req.CartID > 0 {
		c, err := s.carts.GetActiveCartWithItemsForUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		if c == nil || c.ID != *req.CartID {
			return nil, order.ErrCartNotFound
		}

		items := make([]order.CreateItem, 0, len(c.Items))
		for _, it := range c.Items {
			items = append(items, order.CreateItem{
				ProductID: it.ProductID,
				VariantID: it.VariantID,
				Quantity:  it.Quantity,
			})
		}

		o, err := s.createOrderFromItems(ctx, &userID, req.CartID, items, ship)
		if err != nil {
			return nil, err
		}

		// clear cart after successful order (outside of tx OK since order is already committed,
		// but ideally you’d do it in same tx by moving cart tables into same DB and using tx handle).
		_ = s.carts.ClearCart(ctx, *req.CartID)
		_ = s.carts.MarkCartConverted(ctx, *req.CartID)
		return o, nil
	}

	// Scenario B: direct item order
	if len(req.Items) == 0 {
		return nil, order.ErrInvalidInput
	}
	return s.createOrderFromItems(ctx, &userID, nil, req.Items, ship)
}

func (s *Service) resolveShippingForUser(ctx context.Context, userID int64, addressID *int64) (order.ShippingInfo, error) {
	if addressID != nil && *addressID > 0 {
		a, err := s.addresses.GetAddressByID(ctx, *addressID)
		if err != nil {
			return order.ShippingInfo{}, order.ErrAddressNotFound
		}
		if a.UserID != userID {
			return order.ShippingInfo{}, order.ErrAddressNotFound
		}
		return mapAddressToShipping(a), nil
	}

	addrs, err := s.addresses.ListAddressesByUser(ctx, userID)
	if err != nil {
		return order.ShippingInfo{}, err
	}
	if len(addrs) == 0 {
		return order.ShippingInfo{}, order.ErrAddressNotFound
	}

	// pick default if exists else first
	var a *user.Address
	for i := range addrs {
		if addrs[i].IsDefault {
			a = &addrs[i]
			break
		}
	}
	if a == nil {
		a = &addrs[0]
	}
	return mapAddressToShipping(a), nil
}

func mapAddressToShipping(a *user.Address) order.ShippingInfo {
	return order.ShippingInfo{
		FullName:     a.FullName,
		Phone:        a.PhoneNumber,
		Country:      a.Country,
		County:       a.County,
		City:         a.City,
		Area:         a.Area,
		PostalCode:   a.PostalCode,
		AddressLine1: a.AddressLine1,
		AddressLine2: a.AddressLine2,
	}
}

func (s *Service) createOrderFromItems(
	ctx context.Context,
	userID *int64,
	cartID *int64,
	items []order.CreateItem,
	shipping order.ShippingInfo,
) (*order.Order, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()

	o := &order.Order{
		UserID:      userID,
		CartID:      cartID,
		OrderNumber: genOrderNumber(),
		Status:      order.OrderStatusPending,
		Currency:    "KES",
		Shipping:    shipping,
		CreatedAt:   now,
		UpdatedAt:   now,
		Items:       make([]order.OrderItem, 0, len(items)),
	}

	// Reserve/adjust inventory + compute pricing per item
	// Strategy for throughput:
	// - Reserve stock per variant (atomic SQL) inside tx.
	// - Use a single location for now: choose the first inventory row for the variant.
	//   (Later: allocation algorithm.)
	for _, it := range items {
		if it.ProductID <= 0 || it.VariantID <= 0 || it.Quantity <= 0 {
			return nil, order.ErrInvalidInput
		}

		p, err := s.products.GetProductByID(ctx, it.ProductID)
		if err != nil {
			return nil, order.ErrProductNotFound
		}
		v, err := s.variants.GetVariantByID(ctx, it.VariantID)
		if err != nil {
			return nil, order.ErrVariantNotFound
		}
		// basic validation
		if v.ProductID != p.ID || !v.IsActive {
			return nil, order.ErrInvalidInput
		}

		// pick a location row and reserve there
		var locationID int64
		if err := tx.QueryRow(ctx, `SELECT location_id FROM inventory_items WHERE variant_id=$1 ORDER BY location_id ASC LIMIT 1`, it.VariantID).Scan(&locationID); err != nil {
			return nil, order.ErrOutOfStock
		}

		if err := s.inventory.Reserve(ctx, tx, it.VariantID, locationID, it.Quantity); err != nil {
			return nil, order.ErrOutOfStock
		}

		unitPrice := v.Price
		lineSubtotal := unitPrice * float64(it.Quantity)

		orderItem := order.OrderItem{
			ProductID:      p.ID,
			VariantID:      &v.ID,
			ProductName:    p.Name,
			ProductSlug:    &p.Slug,
			UnitPrice:      unitPrice,
			Quantity:       it.Quantity,
			DiscountAmount: 0,
			TaxRate:        0,
			TotalPrice:     lineSubtotal,
			Currency:       o.Currency,
		}

		o.Subtotal += lineSubtotal
		o.TotalAmount += lineSubtotal
		o.Items = append(o.Items, orderItem)
	}

	// TODO: discount application per product (needs discount resolution logic from discount_targets)
	// If you already have "GetProductDiscount" somewhere, plug it here and then:
	// - update o.DiscountAmount, o.TotalAmount
	// - call discounts.RecordRedemption(tx,...)

	if err := s.orders.CreateOrderTx(ctx, tx, o); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return o, nil
}