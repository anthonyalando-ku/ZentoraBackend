package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"zentora-service/internal/domain/inventory"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InventoryRepository struct {
	db *pgxpool.Pool
}

func NewInventoryRepository(db *pgxpool.Pool) *InventoryRepository {
	return &InventoryRepository{db: db}
}

// ---- Locations ----

func (r *InventoryRepository) CreateLocation(ctx context.Context, l *inventory.Location) error {
	const q = `
		INSERT INTO inventory_locations (name, location_code, is_active)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`

	err := r.db.QueryRow(ctx, q, l.Name, l.LocationCode, l.IsActive).
		Scan(&l.ID, &l.CreatedAt)
	if err != nil {
		return mapInventoryError(err)
	}
	return nil
}

func (r *InventoryRepository) GetLocationByID(ctx context.Context, id int64) (*inventory.Location, error) {
	const q = `
		SELECT id, name, location_code, is_active, created_at
		FROM inventory_locations WHERE id = $1`
	return r.scanLocation(ctx, q, id)
}

func (r *InventoryRepository) UpdateLocation(ctx context.Context, l *inventory.Location) error {
	const q = `
		UPDATE inventory_locations
		SET name = $1, location_code = $2, is_active = $3
		WHERE id = $4`

	result, err := r.db.Exec(ctx, q, l.Name, l.LocationCode, l.IsActive, l.ID)
	if err != nil {
		return mapInventoryError(err)
	}
	if result.RowsAffected() == 0 {
		return inventory.ErrLocationNotFound
	}
	return nil
}

func (r *InventoryRepository) DeleteLocation(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM inventory_locations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete location: %w", err)
	}
	if result.RowsAffected() == 0 {
		return inventory.ErrLocationNotFound
	}
	return nil
}

func (r *InventoryRepository) ListLocations(ctx context.Context, f inventory.LocationFilter) ([]inventory.Location, error) {
	q := `SELECT id, name, location_code, is_active, created_at FROM inventory_locations`
	if f.ActiveOnly {
		q += ` WHERE is_active = TRUE`
	}
	q += ` ORDER BY name`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list locations: %w", err)
	}
	defer rows.Close()

	var out []inventory.Location
	for rows.Next() {
		var l inventory.Location
		if err := rows.Scan(&l.ID, &l.Name, &l.LocationCode, &l.IsActive, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan location: %w", err)
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate locations: %w", err)
	}
	return out, nil
}

func (r *InventoryRepository) scanLocation(ctx context.Context, query string, arg any) (*inventory.Location, error) {
	var l inventory.Location
	err := r.db.QueryRow(ctx, query, arg).
		Scan(&l.ID, &l.Name, &l.LocationCode, &l.IsActive, &l.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, inventory.ErrLocationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get location: %w", err)
	}
	return &l, nil
}

// ---- Items ----

// UpsertItem inserts or updates an inventory item.
// Accepts an optional tx; starts its own if nil.
func (r *InventoryRepository) UpsertItem(ctx context.Context, tx pgx.Tx, item *inventory.Item) error {
	if tx != nil {
		return upsertItemTx(ctx, tx, item)
	}
	localTx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = localTx.Rollback(ctx) }()
	if err := upsertItemTx(ctx, localTx, item); err != nil {
		return err
	}
	return localTx.Commit(ctx)
}

func upsertItemTx(ctx context.Context, tx pgx.Tx, item *inventory.Item) error {
	const q = `
		INSERT INTO inventory_items (variant_id, location_id, available_qty, reserved_qty, incoming_qty)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (variant_id, location_id) DO UPDATE
			SET available_qty = EXCLUDED.available_qty,
			    reserved_qty  = EXCLUDED.reserved_qty,
			    incoming_qty  = EXCLUDED.incoming_qty,
			    updated_at    = NOW()
		RETURNING id, updated_at`

	err := tx.QueryRow(ctx, q,
		item.VariantID, item.LocationID,
		item.AvailableQty, item.ReservedQty, item.IncomingQty,
	).Scan(&item.ID, &item.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert inventory item: %w", err)
	}
	return nil
}

func (r *InventoryRepository) GetItemByID(ctx context.Context, id int64) (*inventory.Item, error) {
	const q = `
		SELECT id, variant_id, location_id, available_qty, reserved_qty, incoming_qty, updated_at
		FROM inventory_items WHERE id = $1`
	return r.scanItem(ctx, q, id)
}

func (r *InventoryRepository) GetItemByVariantAndLocation(ctx context.Context, variantID, locationID int64) (*inventory.Item, error) {
	const q = `
		SELECT id, variant_id, location_id, available_qty, reserved_qty, incoming_qty, updated_at
		FROM inventory_items WHERE variant_id = $1 AND location_id = $2`
	return r.scanItem(ctx, q, variantID, locationID)
}

func (r *InventoryRepository) GetItemsByVariant(ctx context.Context, variantID int64) ([]inventory.ItemWithLocation, error) {
	const q = `
		SELECT ii.id, ii.variant_id, ii.location_id, ii.available_qty, ii.reserved_qty,
		       ii.incoming_qty, ii.updated_at, il.name, il.location_code
		FROM inventory_items ii
		JOIN inventory_locations il ON il.id = ii.location_id
		WHERE ii.variant_id = $1
		ORDER BY il.name`

	rows, err := r.db.Query(ctx, q, variantID)
	if err != nil {
		return nil, fmt.Errorf("get items by variant: %w", err)
	}
	defer rows.Close()

	var out []inventory.ItemWithLocation
	for rows.Next() {
		var i inventory.ItemWithLocation
		if err := rows.Scan(
			&i.ID, &i.VariantID, &i.LocationID,
			&i.AvailableQty, &i.ReservedQty, &i.IncomingQty,
			&i.UpdatedAt, &i.LocationName, &i.LocationCode,
		); err != nil {
			return nil, fmt.Errorf("scan inventory item: %w", err)
		}
		out = append(out, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inventory items: %w", err)
	}
	return out, nil
}

func (r *InventoryRepository) GetStockSummary(ctx context.Context, variantID int64) (*inventory.StockSummary, error) {
	const q = `
		SELECT
			$1::BIGINT,
			COALESCE(SUM(available_qty), 0),
			COALESCE(SUM(reserved_qty), 0),
			COALESCE(SUM(incoming_qty), 0)
		FROM inventory_items
		WHERE variant_id = $1`

	var s inventory.StockSummary
	err := r.db.QueryRow(ctx, q, variantID).
		Scan(&s.VariantID, &s.AvailableQty, &s.ReservedQty, &s.IncomingQty)
	if err != nil {
		return nil, fmt.Errorf("get stock summary: %w", err)
	}
	return &s, nil
}

// AdjustAvailable atomically increments/decrements available_qty by delta.
// A negative delta that would make available_qty < 0 returns ErrInsufficientStock.
func (r *InventoryRepository) AdjustAvailable(ctx context.Context, tx pgx.Tx, variantID, locationID int64, delta int) error {
	if tx != nil {
		return adjustAvailableTx(ctx, tx, variantID, locationID, delta)
	}
	localTx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = localTx.Rollback(ctx) }()
	if err := adjustAvailableTx(ctx, localTx, variantID, locationID, delta); err != nil {
		return err
	}
	return localTx.Commit(ctx)
}

func adjustAvailableTx(ctx context.Context, tx pgx.Tx, variantID, locationID int64, delta int) error {
	const q = `
		UPDATE inventory_items
		SET available_qty = available_qty + $1,
		    updated_at    = NOW()
		WHERE variant_id = $2 AND location_id = $3
		  AND available_qty + $1 >= 0
		RETURNING id`

	var id int64
	err := tx.QueryRow(ctx, q, delta, variantID, locationID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		// Either the row doesn't exist or the check failed.
		var exists bool
		_ = tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM inventory_items WHERE variant_id = $1 AND location_id = $2)`,
			variantID, locationID,
		).Scan(&exists)
		if !exists {
			return inventory.ErrItemNotFound
		}
		return inventory.ErrInsufficientStock
	}
	if err != nil {
		return fmt.Errorf("adjust available qty: %w", err)
	}
	return nil
}

// Reserve moves qty from available_qty to reserved_qty atomically.
func (r *InventoryRepository) Reserve(ctx context.Context, tx pgx.Tx, variantID, locationID int64, qty int) error {
	if tx != nil {
		return reserveTx(ctx, tx, variantID, locationID, qty)
	}
	localTx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = localTx.Rollback(ctx) }()
	if err := reserveTx(ctx, localTx, variantID, locationID, qty); err != nil {
		return err
	}
	return localTx.Commit(ctx)
}

func reserveTx(ctx context.Context, tx pgx.Tx, variantID, locationID int64, qty int) error {
	const q = `
		UPDATE inventory_items
		SET available_qty = available_qty - $1,
		    reserved_qty  = reserved_qty  + $1,
		    updated_at    = NOW()
		WHERE variant_id = $2 AND location_id = $3
		  AND available_qty >= $1
		RETURNING id`

	var id int64
	err := tx.QueryRow(ctx, q, qty, variantID, locationID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		_ = tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM inventory_items WHERE variant_id = $1 AND location_id = $2)`,
			variantID, locationID,
		).Scan(&exists)
		if !exists {
			return inventory.ErrItemNotFound
		}
		return inventory.ErrInsufficientStock
	}
	if err != nil {
		return fmt.Errorf("reserve qty: %w", err)
	}
	return nil
}

// Release moves qty back from reserved_qty to available_qty.
func (r *InventoryRepository) Release(ctx context.Context, tx pgx.Tx, variantID, locationID int64, qty int) error {
	if tx != nil {
		return releaseTx(ctx, tx, variantID, locationID, qty)
	}
	localTx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = localTx.Rollback(ctx) }()
	if err := releaseTx(ctx, localTx, variantID, locationID, qty); err != nil {
		return err
	}
	return localTx.Commit(ctx)
}

func releaseTx(ctx context.Context, tx pgx.Tx, variantID, locationID int64, qty int) error {
	const q = `
		UPDATE inventory_items
		SET reserved_qty  = reserved_qty  - $1,
		    available_qty = available_qty + $1,
		    updated_at    = NOW()
		WHERE variant_id = $2 AND location_id = $3
		  AND reserved_qty >= $1
		RETURNING id`

	var id int64
	err := tx.QueryRow(ctx, q, qty, variantID, locationID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return inventory.ErrItemNotFound
	}
	if err != nil {
		return fmt.Errorf("release qty: %w", err)
	}
	return nil
}

func (r *InventoryRepository) DeleteItem(ctx context.Context, variantID, locationID int64) error {
	result, err := r.db.Exec(ctx,
		`DELETE FROM inventory_items WHERE variant_id = $1 AND location_id = $2`,
		variantID, locationID,
	)
	if err != nil {
		return fmt.Errorf("delete inventory item: %w", err)
	}
	if result.RowsAffected() == 0 {
		return inventory.ErrItemNotFound
	}
	return nil
}

func (r *InventoryRepository) scanItem(ctx context.Context, query string, args ...any) (*inventory.Item, error) {
	var i inventory.Item
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&i.ID, &i.VariantID, &i.LocationID,
		&i.AvailableQty, &i.ReservedQty, &i.IncomingQty,
		&i.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, inventory.ErrItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get inventory item: %w", err)
	}
	return &i, nil
}

func mapInventoryError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		if strings.Contains(pgErr.ConstraintName, "location_code") {
			return inventory.ErrLocationCodeConflict
		}
		if strings.Contains(pgErr.ConstraintName, "inventory_items") {
			return inventory.ErrItemConflict
		}
	}
	_ = sql.ErrNoRows
	return fmt.Errorf("inventory repository: %w", err)
}
