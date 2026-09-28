package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strings"
	"testing"
	"time"
	"zentora-service/internal/domain/delivery"
	"zentora-service/internal/domain/order"
	"zentora-service/internal/domain/product"
	"zentora-service/internal/domain/user"
	"zentora-service/internal/domain/variant"
	"zentora-service/internal/repository/postgres"
	orderusecase "zentora-service/internal/service/order"
)

// Use an explicitly configured disposable localhost PostgreSQL cluster only.
func TestDeliveryMigrationAndOrderPersistence(t *testing.T) {
	url := os.Getenv("DELIVERY_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set DELIVERY_TEST_DATABASE_URL to a disposable local PostgreSQL database")
	}
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Host != "127.0.0.1" && cfg.ConnConfig.Host != "localhost" {
		t.Fatal("test database must be local")
	}
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("delivery_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	initial, err := os.ReadFile("../../db/migrations/001_svc_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	// Exercise the exact existing shipping schema rather than an approximation.
	sql := string(initial)
	start := strings.Index(sql, "CREATE TABLE shipping_methods (")
	end := strings.Index(sql[start:], "CREATE TABLE orders (") + start
	if start < 0 || end <= start {
		t.Fatal("shipping schema not found")
	}
	if _, err := db.Exec(ctx, sql[start:end]); err != nil {
		t.Fatal(err)
	}
	// Reuse order column definitions, omitting unrelated foreign-key dependencies.
	for _, table := range []string{"orders", "order_items"} {
		start := strings.Index(sql, "CREATE TABLE "+table+" (")
		end := strings.Index(sql[start:], ");") + start + 2
		lines := strings.Split(sql[start:end], "\n")
		var kept []string
		for _, line := range lines {
			if !strings.Contains(line, "FOREIGN KEY") {
				kept = append(kept, line)
			}
		}
		statement := strings.Join(kept, "\n")
		before := strings.TrimRight(strings.TrimSuffix(statement, ");"), "\r\n \t,")
		if _, err := db.Exec(ctx, before+"\n);"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(ctx, `INSERT INTO shipping_methods (name) VALUES ('Legacy');
 INSERT INTO orders (order_number,status,subtotal,total_amount,shipping_fee,currency,shipping_full_name,shipping_phone,shipping_country,shipping_city,shipping_address_line_1)
 VALUES ('OLD','pending',100,107,7,'KES','Test','000','Kenya','Nairobi','Test');`); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../db/migrations/009_delivery_information.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	repo := postgres.NewDeliveryRepository(db)
	policy, err := repo.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if policy.NairobiIndicativeFee != 300 || policy.PricingMode != "informational" {
		t.Fatalf("bad seed: %+v", policy)
	}
	orders := postgres.NewOrderRepository(db)
	old, err := orders.GetOrderByNumber(ctx, "OLD")
	if err != nil {
		t.Fatal(err)
	}
	if old.DeliveryInformation != nil || old.ShippingFee != 7 || old.TotalAmount != 107 {
		t.Fatal("migration changed history")
	}
	for i, city := range []string{"Nairobi", "Mombasa"} {
		o := &order.Order{OrderNumber: fmt.Sprintf("NEW-%d", i), Status: order.OrderStatusPending, Subtotal: 4500, TotalAmount: 4500, Currency: "KES", Shipping: order.ShippingInfo{FullName: "Test", Phone: "000", Country: "Kenya", City: city, AddressLine1: "Test"}, DeliveryInformation: delivery.NewSnapshot(policy)}
		if err := orders.CreateOrderTx(ctx, nil, o); err != nil {
			t.Fatal(err)
		}
		read, err := orders.GetOrderByID(ctx, o.ID)
		if err != nil {
			t.Fatal(err)
		}
		if read.TotalAmount != 4500 || read.ShippingFee != 0 || read.DeliveryInformation.ConfirmedFee != nil {
			t.Fatal("delivery changed order amount")
		}
		updated, err := orders.UpdateOrderStatus(ctx, o.ID, order.OrderStatusCompleted)
		if err != nil {
			t.Fatal(err)
		}
		if updated.DeliveryInformation.Notice != policy.Message {
			t.Fatal("status update lost snapshot")
		}
	}
	saved, err := repo.Update(ctx, delivery.Update{ExpectedVersion: 1, NairobiIndicativeFee: 400}, 42)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Version != 2 || saved.NairobiIndicativeFee != 400 {
		t.Fatal("update failed")
	}
	if _, err := repo.Update(ctx, delivery.Update{ExpectedVersion: 1, NairobiIndicativeFee: 500}, 42); !errors.Is(err, delivery.ErrConflict) {
		t.Fatal("stale update accepted")
	}
	read, err := orders.GetOrderByNumber(ctx, "NEW-0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(read.DeliveryInformation.Notice, "KES 300") {
		t.Fatal("policy edit rewrote historical snapshot")
	}
	rows, _, err := orders.ListOrders(ctx, order.ListFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected three persisted orders, got %d", len(rows))
	}
	// Exercise both real checkout service paths, including inventory and item persistence.
	if _, err := db.Exec(ctx, `CREATE TABLE inventory_items (id BIGSERIAL PRIMARY KEY, variant_id BIGINT, location_id BIGINT, available_qty INT, reserved_qty INT, updated_at TIMESTAMP);
 INSERT INTO inventory_items (variant_id,location_id,available_qty,reserved_qty) VALUES (1,1,100,0)`); err != nil {
		t.Fatal(err)
	}
	svc := orderusecase.NewService(db, orders, nil, testProducts{}, testVariants{}, postgres.NewInventoryRepository(db), testAddresses{}, nil, nil, nil)
	svc.SetDeliveryReader(repo)
	for _, city := range []string{"Nairobi", "Mombasa"} {
		guest, err := svc.CreateGuestOrder(ctx, &order.CreateGuestOrderRequest{Items: []order.CreateItem{{ProductID: 1, VariantID: 1, Quantity: 2}}, Shipping: order.ShippingInfo{FullName: "Test", Phone: "000", Country: "Kenya", City: city, AddressLine1: "Test"}})
		if err != nil {
			t.Fatal(err)
		}
		if guest.TotalAmount != 4500 || guest.ShippingFee != 0 || guest.DeliveryInformation.ConfirmedFee != nil {
			t.Fatal("guest checkout charged informational fee")
		}
	}
	registered, err := svc.CreateUserOrder(ctx, 42, &order.CreateUserOrderRequest{Items: []order.CreateItem{{ProductID: 1, VariantID: 1, Quantity: 2}}})
	if err != nil {
		t.Fatal(err)
	}
	if registered.TotalAmount != 4500 || registered.ShippingFee != 0 || len(registered.Items) != 1 {
		t.Fatal("registered checkout amount incorrect")
	}
	// Missing policy must still allow checkout, storing an honest generic notice.
	if _, err := db.Exec(ctx, `UPDATE shipping_methods SET is_active=false WHERE code='standard-delivery'`); err != nil {
		t.Fatal(err)
	}
	fallback, err := svc.CreateUserOrder(ctx, 42, &order.CreateUserOrderRequest{Items: []order.CreateItem{{ProductID: 1, VariantID: 1, Quantity: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	if fallback.DeliveryInformation.PolicyVersion != nil || fallback.TotalAmount != 2250 || fallback.DeliveryInformation.Notice != delivery.FallbackNotice {
		t.Fatal("invalid fallback checkout")
	}
}

type testProducts struct{}

func (testProducts) GetProductByID(context.Context, int64) (*product.Product, error) {
	return &product.Product{ID: 1, Name: "Test", Slug: "test"}, nil
}

type testVariants struct{}

func (testVariants) GetVariantByID(context.Context, int64) (*variant.Variant, error) {
	return &variant.Variant{ID: 1, ProductID: 1, Price: 2250, IsActive: true}, nil
}

type testAddresses struct{}

func (testAddresses) GetAddressByID(context.Context, int64) (*user.Address, error) {
	return &user.Address{ID: 1, UserID: 42, FullName: "Test", PhoneNumber: "000", Country: "Kenya", City: "Kisumu", AddressLine1: "Test", IsDefault: true}, nil
}
func (s testAddresses) ListAddressesByUser(ctx context.Context, id int64) ([]user.Address, error) {
	a, _ := s.GetAddressByID(ctx, id)
	return []user.Address{*a}, nil
}
