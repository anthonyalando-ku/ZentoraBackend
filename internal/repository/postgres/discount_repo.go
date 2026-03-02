package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	//"time"

	"zentora-service/internal/domain/discount"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DiscountRepository struct {
	db *pgxpool.Pool
}

func NewDiscountRepository(db *pgxpool.Pool) *DiscountRepository {
	return &DiscountRepository{db: db}
}

func (r *DiscountRepository) CreateDiscount(ctx context.Context, tx pgx.Tx, d *discount.Discount, targets []discount.DiscountTarget) error {
	if tx != nil {
		return r.insertDiscountGraph(ctx, tx, d, targets)
	}
	localTx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = localTx.Rollback(ctx) }()
	if err := r.insertDiscountGraph(ctx, localTx, d, targets); err != nil {
		return err
	}
	return localTx.Commit(ctx)
}

func (r *DiscountRepository) insertDiscountGraph(ctx context.Context, tx pgx.Tx, d *discount.Discount, targets []discount.DiscountTarget) error {
	const q = `
		INSERT INTO discounts
			(name, code, discount_type, value, min_order_amount, max_redemptions, starts_at, ends_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`

	err := tx.QueryRow(ctx, q,
		d.Name, d.Code, d.DiscountType, d.Value,
		d.MinOrderAmount, d.MaxRedemptions,
		d.StartsAt, d.EndsAt, d.IsActive,
	).Scan(&d.ID, &d.CreatedAt)
	if err != nil {
		return mapDiscountError(err)
	}

	for i := range targets {
		targets[i].DiscountID = d.ID
		if err := insertTargetTx(ctx, tx, &targets[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *DiscountRepository) GetDiscountByID(ctx context.Context, id int64) (*discount.Discount, error) {
	const q = `
		SELECT id, name, code, discount_type, value, min_order_amount,
		       max_redemptions, starts_at, ends_at, is_active, created_at
		FROM discounts WHERE id = $1`
	return r.scanOne(ctx, q, id)
}

func (r *DiscountRepository) GetDiscountByCode(ctx context.Context, code string) (*discount.Discount, error) {
	const q = `
		SELECT id, name, code, discount_type, value, min_order_amount,
		       max_redemptions, starts_at, ends_at, is_active, created_at
		FROM discounts WHERE code = $1`
	return r.scanOne(ctx, q, code)
}

func (r *DiscountRepository) GetDiscountWithTargets(ctx context.Context, id int64) (*discount.DiscountWithTargets, error) {
	d, err := r.GetDiscountByID(ctx, id)
	if err != nil {
		return nil, err
	}
	targets, err := r.GetTargets(ctx, id)
	if err != nil {
		return nil, err
	}
	return &discount.DiscountWithTargets{Discount: *d, Targets: targets}, nil
}

func (r *DiscountRepository) UpdateDiscount(ctx context.Context, d *discount.Discount) error {
	const q = `
		UPDATE discounts
		SET name = $1, code = $2, discount_type = $3, value = $4,
		    min_order_amount = $5, max_redemptions = $6,
		    starts_at = $7, ends_at = $8, is_active = $9
		WHERE id = $10`

	result, err := r.db.Exec(ctx, q,
		d.Name, d.Code, d.DiscountType, d.Value,
		d.MinOrderAmount, d.MaxRedemptions,
		d.StartsAt, d.EndsAt, d.IsActive, d.ID,
	)
	if err != nil {
		return mapDiscountError(err)
	}
	if result.RowsAffected() == 0 {
		return discount.ErrNotFound
	}
	return nil
}

func (r *DiscountRepository) DeleteDiscount(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM discounts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete discount: %w", err)
	}
	if result.RowsAffected() == 0 {
		return discount.ErrNotFound
	}
	return nil
}

func (r *DiscountRepository) ListDiscounts(ctx context.Context, f discount.ListFilter) ([]discount.Discount, error) {
	q := `
		SELECT id, name, code, discount_type, value, min_order_amount,
		       max_redemptions, starts_at, ends_at, is_active, created_at
		FROM discounts WHERE 1=1`

	args := make([]any, 0, 2)
	idx := 1

	if f.ActiveOnly {
		q += fmt.Sprintf(" AND is_active = $%d", idx)
		args = append(args, true)
		idx++
	}
	if f.Code != nil {
		q += fmt.Sprintf(" AND code = $%d", idx)
		args = append(args, *f.Code)
		idx++
	}
	_ = idx
	q += " ORDER BY created_at DESC"

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list discounts: %w", err)
	}
	defer rows.Close()

	var out []discount.Discount
	for rows.Next() {
		var d discount.Discount
		if err := scanDiscount(rows, &d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate discounts: %w", err)
	}
	return out, nil
}

func (r *DiscountRepository) CountRedemptions(ctx context.Context, discountID int64) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM discount_redemptions WHERE discount_id = $1`, discountID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count redemptions: %w", err)
	}
	return count, nil
}

func (r *DiscountRepository) RecordRedemption(ctx context.Context, tx pgx.Tx, red *discount.DiscountRedemption) error {
	if tx != nil {
		return insertRedemptionTx(ctx, tx, red)
	}
	localTx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = localTx.Rollback(ctx) }()
	if err := insertRedemptionTx(ctx, localTx, red); err != nil {
		return err
	}
	return localTx.Commit(ctx)
}

func insertRedemptionTx(ctx context.Context, tx pgx.Tx, red *discount.DiscountRedemption) error {
	const q = `
		INSERT INTO discount_redemptions (discount_id, order_id, user_id)
		VALUES ($1, $2, $3)
		RETURNING id, redeemed_at`

	err := tx.QueryRow(ctx, q, red.DiscountID, red.OrderID, red.UserID).
		Scan(&red.ID, &red.RedeemedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return discount.ErrRedemptionConflict
		}
		return fmt.Errorf("record redemption: %w", err)
	}
	return nil
}

func (r *DiscountRepository) GetTargets(ctx context.Context, discountID int64) ([]discount.DiscountTarget, error) {
	rows, err := r.db.Query(ctx,
		`SELECT discount_id, target_type, target_id FROM discount_targets WHERE discount_id = $1`,
		discountID,
	)
	if err != nil {
		return nil, fmt.Errorf("get discount targets: %w", err)
	}
	defer rows.Close()

	var out []discount.DiscountTarget
	for rows.Next() {
		var t discount.DiscountTarget
		if err := rows.Scan(&t.DiscountID, &t.TargetType, &t.TargetID); err != nil {
			return nil, fmt.Errorf("scan discount target: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate discount targets: %w", err)
	}
	return out, nil
}

func (r *DiscountRepository) SetTargets(ctx context.Context, tx pgx.Tx, discountID int64, targets []discount.DiscountTarget) error {
	if tx != nil {
		return setTargetsTx(ctx, tx, discountID, targets)
	}
	localTx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = localTx.Rollback(ctx) }()
	if err := setTargetsTx(ctx, localTx, discountID, targets); err != nil {
		return err
	}
	return localTx.Commit(ctx)
}

func setTargetsTx(ctx context.Context, tx pgx.Tx, discountID int64, targets []discount.DiscountTarget) error {
	if _, err := tx.Exec(ctx,
		`DELETE FROM discount_targets WHERE discount_id = $1`, discountID,
	); err != nil {
		return fmt.Errorf("clear discount targets: %w", err)
	}
	for i := range targets {
		targets[i].DiscountID = discountID
		if err := insertTargetTx(ctx, tx, &targets[i]); err != nil {
			return err
		}
	}
	return nil
}

func insertTargetTx(ctx context.Context, tx pgx.Tx, t *discount.DiscountTarget) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO discount_targets (discount_id, target_type, target_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		t.DiscountID, t.TargetType, t.TargetID,
	)
	if err != nil {
		return fmt.Errorf("insert discount target: %w", err)
	}
	return nil
}

func (r *DiscountRepository) scanOne(ctx context.Context, query string, arg any) (*discount.Discount, error) {
	row := r.db.QueryRow(ctx, query, arg)
	var d discount.Discount
	if err := scanDiscountRow(row, &d); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, discount.ErrNotFound
		}
		return nil, fmt.Errorf("get discount: %w", err)
	}
	return &d, nil
}

func scanDiscountRow(row pgx.Row, d *discount.Discount) error {
	return row.Scan(
		&d.ID, &d.Name, &d.Code, &d.DiscountType, &d.Value,
		&d.MinOrderAmount, &d.MaxRedemptions,
		&d.StartsAt, &d.EndsAt, &d.IsActive, &d.CreatedAt,
	)
}

func scanDiscount(rows pgx.Rows, d *discount.Discount) error {
	return rows.Scan(
		&d.ID, &d.Name, &d.Code, &d.DiscountType, &d.Value,
		&d.MinOrderAmount, &d.MaxRedemptions,
		&d.StartsAt, &d.EndsAt, &d.IsActive, &d.CreatedAt,
	)
}

func mapDiscountError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		if strings.Contains(pgErr.ConstraintName, "code") {
			return discount.ErrCodeConflict
		}
	}
	_ = sql.ErrNoRows
	return fmt.Errorf("discount repository: %w", err)
}
