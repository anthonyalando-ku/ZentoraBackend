package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"zentora-service/internal/domain/brand"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BrandRepository struct {
	db *pgxpool.Pool
}

func NewBrandRepository(db *pgxpool.Pool) *BrandRepository {
	return &BrandRepository{db: db}
}

func (r *BrandRepository) CreateBrand(ctx context.Context, b *brand.Brand) error {
	const q = `
		INSERT INTO product_brands (name, slug, logo_url, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	err := r.db.QueryRow(ctx, q, b.Name, b.Slug, b.LogoURL, b.IsActive).
		Scan(&b.ID, &b.CreatedAt)
	if err != nil {
		return mapBrandError(err)
	}
	return nil
}

func (r *BrandRepository) GetBrandByID(ctx context.Context, id int64) (*brand.Brand, error) {
	const q = `SELECT id, name, slug, logo_url, is_active, created_at FROM product_brands WHERE id = $1`
	return r.scanOne(ctx, q, id)
}

func (r *BrandRepository) GetBrandBySlug(ctx context.Context, slug string) (*brand.Brand, error) {
	const q = `SELECT id, name, slug, logo_url, is_active, created_at FROM product_brands WHERE slug = $1`
	return r.scanOne(ctx, q, slug)
}

func (r *BrandRepository) UpdateBrand(ctx context.Context, b *brand.Brand) error {
	const q = `
		UPDATE product_brands
		SET name = $1, slug = $2, logo_url = $3, is_active = $4
		WHERE id = $5`

	result, err := r.db.Exec(ctx, q, b.Name, b.Slug, b.LogoURL, b.IsActive, b.ID)
	if err != nil {
		return mapBrandError(err)
	}
	if result.RowsAffected() == 0 {
		return brand.ErrNotFound
	}
	return nil
}

func (r *BrandRepository) DeleteBrand(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM product_brands WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete brand: %w", err)
	}
	if result.RowsAffected() == 0 {
		return brand.ErrNotFound
	}
	return nil
}

func (r *BrandRepository) ListBrands(ctx context.Context, f brand.ListFilter) ([]brand.Brand, error) {
	q := `SELECT id, name, slug, logo_url, is_active, created_at FROM product_brands`
	if f.ActiveOnly {
		q += ` WHERE is_active = TRUE`
	}
	q += ` ORDER BY name`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list brands: %w", err)
	}
	defer rows.Close()

	var out []brand.Brand
	for rows.Next() {
		var b brand.Brand
		if err := rows.Scan(&b.ID, &b.Name, &b.Slug, &b.LogoURL, &b.IsActive, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan brand: %w", err)
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate brands: %w", err)
	}
	return out, nil
}

func (r *BrandRepository) scanOne(ctx context.Context, query string, arg any) (*brand.Brand, error) {
	var b brand.Brand
	err := r.db.QueryRow(ctx, query, arg).Scan(
		&b.ID, &b.Name, &b.Slug, &b.LogoURL, &b.IsActive, &b.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, brand.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get brand: %w", err)
	}
	return &b, nil
}

func mapBrandError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		if strings.Contains(pgErr.ConstraintName, "slug") {
			return brand.ErrSlugConflict
		}
		if strings.Contains(pgErr.ConstraintName, "name") {
			return brand.ErrNameConflict
		}
	}
	return fmt.Errorf("brand repository: %w", err)
}
