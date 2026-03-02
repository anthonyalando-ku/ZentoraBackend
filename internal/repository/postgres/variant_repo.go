package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"zentora-service/internal/domain/variant"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VariantRepository struct {
	db *pgxpool.Pool
}

func NewVariantRepository(db *pgxpool.Pool) *VariantRepository {
	return &VariantRepository{db: db}
}

func (r *VariantRepository) CreateVariant(ctx context.Context, tx pgx.Tx, v *variant.Variant) error {
	if tx != nil {
		return insertVariantTx(ctx, tx, v)
	}

	localTx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = localTx.Rollback(ctx) }()

	if err := insertVariantTx(ctx, localTx, v); err != nil {
		return err
	}
	return localTx.Commit(ctx)
}

func (r *VariantRepository) GetVariantByID(ctx context.Context, id int64) (*variant.Variant, error) {
	const q = `
		SELECT id, product_id, sku, price, weight, is_active, created_at
		FROM product_variants WHERE id = $1`
	return scanVariantOne(ctx, r.db, q, id)
}

func (r *VariantRepository) GetVariantBySKU(ctx context.Context, sku string) (*variant.Variant, error) {
	const q = `
		SELECT id, product_id, sku, price, weight, is_active, created_at
		FROM product_variants WHERE sku = $1`
	return scanVariantOne(ctx, r.db, q, sku)
}

func (r *VariantRepository) UpdateVariant(ctx context.Context, v *variant.Variant) error {
	const q = `
		UPDATE product_variants
		SET sku = $1, price = $2, weight = $3, is_active = $4
		WHERE id = $5`

	result, err := r.db.Exec(ctx, q, v.SKU, v.Price, v.Weight, v.IsActive, v.ID)
	if err != nil {
		return mapVariantError(err)
	}
	if result.RowsAffected() == 0 {
		return variant.ErrNotFound
	}
	return nil
}

func (r *VariantRepository) DeleteVariant(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM product_variants WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete variant: %w", err)
	}
	if result.RowsAffected() == 0 {
		return variant.ErrNotFound
	}
	return nil
}

func (r *VariantRepository) ListVariantsByProduct(ctx context.Context, productID int64, activeOnly bool) ([]variant.Variant, error) {
	q := `
		SELECT id, product_id, sku, price, weight, is_active, created_at
		FROM product_variants
		WHERE product_id = $1`
	if activeOnly {
		q += ` AND is_active = TRUE`
	}
	q += ` ORDER BY id`

	rows, err := r.db.Query(ctx, q, productID)
	if err != nil {
		return nil, fmt.Errorf("list variants: %w", err)
	}
	defer rows.Close()
	return scanVariants(rows)
}

func (r *VariantRepository) SetVariantAttributeValues(ctx context.Context, tx pgx.Tx, variantID int64, valueIDs []int64) error {
	if tx != nil {
		return setVariantAttributeValuesTx(ctx, tx, variantID, valueIDs)
	}

	localTx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = localTx.Rollback(ctx) }()

	if err := setVariantAttributeValuesTx(ctx, localTx, variantID, valueIDs); err != nil {
		return err
	}
	return localTx.Commit(ctx)
}

func (r *VariantRepository) GetVariantAttributeValues(ctx context.Context, variantID int64) ([]int64, error) {
	rows, err := r.db.Query(ctx,
		`SELECT attribute_value_id FROM variant_attribute_values WHERE variant_id = $1 ORDER BY attribute_value_id`,
		variantID,
	)
	if err != nil {
		return nil, fmt.Errorf("get variant attribute values: %w", err)
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan variant attribute value: %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate variant attribute values: %w", err)
	}
	return out, nil
}

// CreateVariantWithAttributesTx creates a variant and links its attribute
// values inside the provided transaction. Exported for use by ProductRepository.
func CreateVariantWithAttributesTx(ctx context.Context, tx pgx.Tx, v *variant.Variant, valueIDs []int64) error {
	if err := insertVariantTx(ctx, tx, v); err != nil {
		return err
	}
	if len(valueIDs) > 0 {
		if err := setVariantAttributeValuesTx(ctx, tx, v.ID, valueIDs); err != nil {
			return err
		}
	}
	return nil
}

func insertVariantTx(ctx context.Context, tx pgx.Tx, v *variant.Variant) error {
	const q = `
		INSERT INTO product_variants (product_id, sku, price, weight, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`

	err := tx.QueryRow(ctx, q, v.ProductID, v.SKU, v.Price, v.Weight, v.IsActive).
		Scan(&v.ID, &v.CreatedAt)
	if err != nil {
		return mapVariantError(err)
	}
	return nil
}

func setVariantAttributeValuesTx(ctx context.Context, tx pgx.Tx, variantID int64, valueIDs []int64) error {
	if _, err := tx.Exec(ctx,
		`DELETE FROM variant_attribute_values WHERE variant_id = $1`, variantID,
	); err != nil {
		return fmt.Errorf("clear variant attribute values: %w", err)
	}
	for _, id := range valueIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO variant_attribute_values (variant_id, attribute_value_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			variantID, id,
		); err != nil {
			return fmt.Errorf("insert variant attribute value %d: %w", id, err)
		}
	}
	return nil
}

func scanVariantOne(ctx context.Context, db interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, query string, arg any) (*variant.Variant, error) {
	var v variant.Variant
	err := db.QueryRow(ctx, query, arg).Scan(
		&v.ID, &v.ProductID, &v.SKU, &v.Price, &v.Weight, &v.IsActive, &v.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, variant.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get variant: %w", err)
	}
	return &v, nil
}

func scanVariants(rows pgx.Rows) ([]variant.Variant, error) {
	var out []variant.Variant
	for rows.Next() {
		var v variant.Variant
		if err := rows.Scan(&v.ID, &v.ProductID, &v.SKU, &v.Price, &v.Weight, &v.IsActive, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan variant: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate variants: %w", err)
	}
	return out, nil
}

func mapVariantError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		if strings.Contains(pgErr.ConstraintName, "sku") ||
			strings.Contains(pgErr.ConstraintName, "product_variants") {
			return variant.ErrSKUConflict
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return variant.ErrNotFound
	}
	_ = sql.ErrNoRows
	return fmt.Errorf("variant repository: %w", err)
}
