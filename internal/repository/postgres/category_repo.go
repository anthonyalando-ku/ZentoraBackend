package postgres

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"zentora-service/internal/domain/category"
	xerrors "zentora-service/internal/pkg/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const pgUniqueViolation = "23505"

type CategoryRepository struct {
	db *pgxpool.Pool
}

func NewCategoryRepository(db *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{db: db}
}

type dbExecutor interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (r *CategoryRepository) CreateCategory(ctx context.Context, tx pgx.Tx, c *category.Category) error {
	if tx != nil {
		return r.insertCategory(ctx, tx, c)
	}
	localTx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = localTx.Rollback(ctx) }()
	if err := r.insertCategory(ctx, localTx, c); err != nil {
		return err
	}
	return localTx.Commit(ctx)
}

func (r *CategoryRepository) insertCategory(ctx context.Context, ex dbExecutor, c *category.Category) error {
	c.Slug = GenerateSlug(c.Name)
	const q = `
		INSERT INTO product_categories (name, slug, parent_id, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	err := ex.QueryRow(ctx, q, c.Name, c.Slug, c.ParentID, c.IsActive).
		Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return mapCategoryError(err)
	}
	return nil
}

func (r *CategoryRepository) GetCategoryByID(ctx context.Context, id int64) (*category.Category, error) {
	const q = `
		SELECT id, name, slug, parent_id, is_active, created_at
		FROM product_categories WHERE id = $1`
	return r.scanOne(ctx, q, id)
}

func (r *CategoryRepository) GetCategoryBySlug(ctx context.Context, slug string) (*category.Category, error) {
	const q = `
		SELECT id, name, slug, parent_id, is_active, created_at
		FROM product_categories WHERE slug = $1`
	return r.scanOne(ctx, q, slug)
}

func (r *CategoryRepository) scanOne(ctx context.Context, query string, arg any) (*category.Category, error) {
	var c category.Category
	err := r.db.QueryRow(ctx, query, arg).Scan(
		&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.IsActive, &c.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, category.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get category: %w", err)
	}
	return &c, nil
}

func (r *CategoryRepository) UpdateCategory(ctx context.Context, c *category.Category) error {
	const q = `
		UPDATE product_categories
		SET name = $1, slug = $2, parent_id = $3, is_active = $4
		WHERE id = $5`
	result, err := r.db.Exec(ctx, q, c.Name, c.Slug, c.ParentID, c.IsActive, c.ID)
	if err != nil {
		return mapCategoryError(err)
	}
	if result.RowsAffected() == 0 {
		return category.ErrNotFound
	}
	return nil
}

func (r *CategoryRepository) DeleteCategory(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM product_categories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	if result.RowsAffected() == 0 {
		return category.ErrNotFound
	}
	return nil
}

func (r *CategoryRepository) ListCategories(ctx context.Context, f category.ListFilter) ([]category.Category, error) {
	q := `
		SELECT id, name, slug, parent_id, is_active, created_at
		FROM product_categories
		WHERE 1=1`

	args := make([]any, 0, 2)
	idx := 1

	if f.ActiveOnly {
		q += fmt.Sprintf(" AND is_active = $%d", idx)
		args = append(args, true)
		idx++
	}

	switch {
	case f.ParentID == nil:
	case *f.ParentID == 0:
		q += " AND parent_id IS NULL"
	default:
		q += fmt.Sprintf(" AND parent_id = $%d", idx)
		args = append(args, *f.ParentID)
		idx++
	}

	_ = idx
	q += " ORDER BY name"

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()
	return scanCategories(rows)
}

func (r *CategoryRepository) GetCategoryAncestors(ctx context.Context, id int64) ([]category.CategoryClosure, error) {
	const q = `
		SELECT ancestor_id, descendant_id, depth
		FROM category_closure
		WHERE descendant_id = $1 AND ancestor_id <> $1
		ORDER BY depth DESC`
	return queryCategoryClosure(ctx, r.db, q, id)
}

func (r *CategoryRepository) GetCategoryDescendants(ctx context.Context, id int64) ([]category.CategoryClosure, error) {
	const q = `
		SELECT ancestor_id, descendant_id, depth
		FROM category_closure
		WHERE ancestor_id = $1 AND descendant_id <> $1
		ORDER BY depth`
	return queryCategoryClosure(ctx, r.db, q, id)
}

func (r *CategoryRepository) IsAncestor(ctx context.Context, id, potentialAncestor int64) (bool, error) {
	if id == potentialAncestor {
		return true, nil
	}
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM category_closure
			WHERE ancestor_id = $1 AND descendant_id = $2 AND depth > 0
		)`, potentialAncestor, id,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check ancestor: %w", err)
	}
	return exists, nil
}

// ---- product_category_map ----

func (r *CategoryRepository) AddProductCategory(ctx context.Context, productID, categoryID int64) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO product_category_map (product_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		productID, categoryID,
	)
	if err != nil {
		return fmt.Errorf("add product category: %w", err)
	}
	return nil
}

func (r *CategoryRepository) RemoveProductCategory(ctx context.Context, productID, categoryID int64) error {
	result, err := r.db.Exec(ctx,
		`DELETE FROM product_category_map WHERE product_id = $1 AND category_id = $2`,
		productID, categoryID,
	)
	if err != nil {
		return fmt.Errorf("remove product category: %w", err)
	}
	if result.RowsAffected() == 0 {
		return category.ErrNotFound
	}
	return nil
}

func (r *CategoryRepository) GetProductCategories(ctx context.Context, productID int64) ([]category.Category, error) {
	const q = `
		SELECT pc.id, pc.name, pc.slug, pc.parent_id, pc.is_active, pc.created_at
		FROM product_categories pc
		JOIN product_category_map pcm ON pcm.category_id = pc.id
		WHERE pcm.product_id = $1
		ORDER BY pc.name`

	rows, err := r.db.Query(ctx, q, productID)
	if err != nil {
		return nil, fmt.Errorf("get product categories: %w", err)
	}
	defer rows.Close()
	return scanCategories(rows)
}

// SetProductCategories replaces all category links for a product atomically.
// Accepts an optional tx; starts its own if nil.
func (r *CategoryRepository) SetProductCategories(ctx context.Context, tx pgx.Tx, productID int64, categoryIDs []int64) error {
	if tx != nil {
		return setProductCategoriesTx(ctx, tx, productID, categoryIDs)
	}
	localTx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = localTx.Rollback(ctx) }()
	if err := setProductCategoriesTx(ctx, localTx, productID, categoryIDs); err != nil {
		return err
	}
	return localTx.Commit(ctx)
}

func setProductCategoriesTx(ctx context.Context, tx pgx.Tx, productID int64, categoryIDs []int64) error {
	if _, err := tx.Exec(ctx,
		`DELETE FROM product_category_map WHERE product_id = $1`, productID,
	); err != nil {
		return fmt.Errorf("clear product categories: %w", err)
	}
	for _, catID := range categoryIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_category_map (product_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			productID, catID,
		); err != nil {
			return fmt.Errorf("link category %d: %w", catID, err)
		}
	}
	return nil
}

// ---- shared scanners ----

func scanCategories(rows pgx.Rows) ([]category.Category, error) {
	var out []category.Category
	for rows.Next() {
		var c category.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}
	return out, nil
}

type closureQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func queryCategoryClosure(ctx context.Context, q closureQuerier, query string, id int64) ([]category.CategoryClosure, error) {
	rows, err := q.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("query closure: %w", err)
	}
	defer rows.Close()

	var out []category.CategoryClosure
	for rows.Next() {
		var cc category.CategoryClosure
		if err := rows.Scan(&cc.AncestorID, &cc.DescendantID, &cc.Depth); err != nil {
			return nil, fmt.Errorf("scan closure: %w", err)
		}
		out = append(out, cc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate closure: %w", err)
	}
	return out, nil
}

var (
	reNonAlphanumeric = regexp.MustCompile(`[^\p{L}\p{N}]+`)
	reMultipleDash    = regexp.MustCompile(`-{2,}`)
)

func GenerateSlug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = reNonAlphanumeric.ReplaceAllString(s, "-")
	s = reMultipleDash.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	return s
}

func mapCategoryError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		if strings.Contains(pgErr.ConstraintName, "slug") {
			return category.ErrSlugConflict
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return category.ErrNotFound
	}
	_ = xerrors.ErrNotFound
	return fmt.Errorf("category repository: %w", err)
}
