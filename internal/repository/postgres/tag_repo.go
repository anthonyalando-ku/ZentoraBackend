package postgres

import (
	"context"
	"errors"
	"fmt"

	"zentora-service/internal/domain/tag"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TagRepository struct {
	db *pgxpool.Pool
}

func NewTagRepository(db *pgxpool.Pool) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) FindOrCreateByName(ctx context.Context, name string) (*tag.Tag, error) {
	t, err := FindOrCreateTagTx(ctx, r.db, name)
	if err != nil {
		return nil, fmt.Errorf("find or create tag %q: %w", name, err)
	}
	return t, nil
}

func (r *TagRepository) GetTagByID(ctx context.Context, id int64) (*tag.Tag, error) {
	var t tag.Tag
	err := r.db.QueryRow(ctx, `SELECT id, name, slug FROM tags WHERE id = $1`, id).
		Scan(&t.ID, &t.Name, &t.Slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, tag.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get tag: %w", err)
	}
	return &t, nil
}

func (r *TagRepository) ListTags(ctx context.Context) ([]tag.Tag, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name, slug FROM tags ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	var out []tag.Tag
	for rows.Next() {
		var t tag.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}
	return out, nil
}

// ---- product_tags ----

func (r *TagRepository) AddTagToProduct(ctx context.Context, productID, tagID int64) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO product_tags (product_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		productID, tagID,
	)
	if err != nil {
		return fmt.Errorf("add tag to product: %w", err)
	}
	return nil
}

func (r *TagRepository) RemoveTagFromProduct(ctx context.Context, productID, tagID int64) error {
	result, err := r.db.Exec(ctx,
		`DELETE FROM product_tags WHERE product_id = $1 AND tag_id = $2`,
		productID, tagID,
	)
	if err != nil {
		return fmt.Errorf("remove tag from product: %w", err)
	}
	if result.RowsAffected() == 0 {
		return tag.ErrNotFound
	}
	return nil
}

func (r *TagRepository) GetProductTags(ctx context.Context, productID int64) ([]tag.Tag, error) {
	const q = `
		SELECT t.id, t.name, t.slug
		FROM tags t
		JOIN product_tags pt ON pt.tag_id = t.id
		WHERE pt.product_id = $1
		ORDER BY t.name`

	rows, err := r.db.Query(ctx, q, productID)
	if err != nil {
		return nil, fmt.Errorf("get product tags: %w", err)
	}
	defer rows.Close()

	var out []tag.Tag
	for rows.Next() {
		var t tag.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product tags: %w", err)
	}
	return out, nil
}

// SetProductTags replaces all tags for a product atomically inside a transaction.
// Called by ProductRepository.CreateProductTx and the standalone SetProductTags service method.
func (r *TagRepository) SetProductTags(ctx context.Context, tx pgx.Tx, productID int64, names []string) error {
	if tx != nil {
		return setProductTagsTx(ctx, tx, productID, names)
	}

	localTx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = localTx.Rollback(ctx) }()

	if err := setProductTagsTx(ctx, localTx, productID, names); err != nil {
		return err
	}
	return localTx.Commit(ctx)
}

func setProductTagsTx(ctx context.Context, tx pgx.Tx, productID int64, names []string) error {
	if _, err := tx.Exec(ctx,
		`DELETE FROM product_tags WHERE product_id = $1`, productID,
	); err != nil {
		return fmt.Errorf("clear product tags: %w", err)
	}

	for _, name := range names {
		t, err := FindOrCreateTagTx(ctx, tx, name)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_tags (product_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			productID, t.ID,
		); err != nil {
			return fmt.Errorf("link tag %q: %w", name, err)
		}
	}
	return nil
}

// FindOrCreateTagTx is exported for use inside ProductRepository transactions.
func FindOrCreateTagTx(ctx context.Context, ex interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, name string) (*tag.Tag, error) {
	slug := GenerateSlug(name)

	const q = `
		INSERT INTO tags (name, slug)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET slug = EXCLUDED.slug
		RETURNING id, name, slug`

	var t tag.Tag
	if err := ex.QueryRow(ctx, q, name, slug).Scan(&t.ID, &t.Name, &t.Slug); err != nil {
		return nil, fmt.Errorf("find or create tag: %w", err)
	}
	return &t, nil
}
