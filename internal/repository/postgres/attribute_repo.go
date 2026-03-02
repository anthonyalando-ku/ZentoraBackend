package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"zentora-service/internal/domain/attribute"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttributeRepository struct {
	db *pgxpool.Pool
}

func NewAttributeRepository(db *pgxpool.Pool) *AttributeRepository {
	return &AttributeRepository{db: db}
}

func (r *AttributeRepository) CreateAttribute(ctx context.Context, a *attribute.Attribute) error {
	const q = `
		INSERT INTO attributes (name, slug, is_variant_dimension)
		VALUES ($1, $2, $3)
		RETURNING id`

	err := r.db.QueryRow(ctx, q, a.Name, a.Slug, a.IsVariantDimension).Scan(&a.ID)
	if err != nil {
		return mapAttributeError(err)
	}
	return nil
}

func (r *AttributeRepository) GetAttributeByID(ctx context.Context, id int64) (*attribute.Attribute, error) {
	const q = `SELECT id, name, slug, is_variant_dimension FROM attributes WHERE id = $1`
	return r.scanOne(ctx, q, id)
}

func (r *AttributeRepository) GetAttributeBySlug(ctx context.Context, slug string) (*attribute.Attribute, error) {
	const q = `SELECT id, name, slug, is_variant_dimension FROM attributes WHERE slug = $1`
	return r.scanOne(ctx, q, slug)
}

func (r *AttributeRepository) scanOne(ctx context.Context, query string, arg any) (*attribute.Attribute, error) {
	var a attribute.Attribute
	err := r.db.QueryRow(ctx, query, arg).Scan(&a.ID, &a.Name, &a.Slug, &a.IsVariantDimension)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, attribute.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get attribute: %w", err)
	}
	return &a, nil
}

func (r *AttributeRepository) UpdateAttribute(ctx context.Context, a *attribute.Attribute) error {
	const q = `
		UPDATE attributes
		SET name = $1, slug = $2, is_variant_dimension = $3
		WHERE id = $4`

	result, err := r.db.Exec(ctx, q, a.Name, a.Slug, a.IsVariantDimension, a.ID)
	if err != nil {
		return mapAttributeError(err)
	}
	if result.RowsAffected() == 0 {
		return attribute.ErrNotFound
	}
	return nil
}

func (r *AttributeRepository) DeleteAttribute(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM attributes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete attribute: %w", err)
	}
	if result.RowsAffected() == 0 {
		return attribute.ErrNotFound
	}
	return nil
}

func (r *AttributeRepository) ListAttributes(ctx context.Context) ([]attribute.Attribute, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name, slug, is_variant_dimension FROM attributes ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list attributes: %w", err)
	}
	defer rows.Close()

	var out []attribute.Attribute
	for rows.Next() {
		var a attribute.Attribute
		if err := rows.Scan(&a.ID, &a.Name, &a.Slug, &a.IsVariantDimension); err != nil {
			return nil, fmt.Errorf("scan attribute: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attributes: %w", err)
	}
	return out, nil
}

func (r *AttributeRepository) ListAttributesWithValues(ctx context.Context) ([]attribute.AttributeWithValues, error) {
	const q = `
		SELECT a.id, a.name, a.slug, a.is_variant_dimension,
		       av.id, av.attribute_id, av.value
		FROM attributes a
		LEFT JOIN attribute_values av ON av.attribute_id = a.id
		ORDER BY a.name, av.value`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list attributes with values: %w", err)
	}
	defer rows.Close()

	index := make(map[int64]*attribute.AttributeWithValues)
	var order []int64

	for rows.Next() {
		var (
			a        attribute.Attribute
			avID     *int64
			avAttrID *int64
			avValue  *string
		)
		if err := rows.Scan(&a.ID, &a.Name, &a.Slug, &a.IsVariantDimension, &avID, &avAttrID, &avValue); err != nil {
			return nil, fmt.Errorf("scan attribute with values: %w", err)
		}
		entry, ok := index[a.ID]
		if !ok {
			entry = &attribute.AttributeWithValues{Attribute: a, Values: []attribute.AttributeValue{}}
			index[a.ID] = entry
			order = append(order, a.ID)
		}
		if avID != nil {
			entry.Values = append(entry.Values, attribute.AttributeValue{
				ID:          *avID,
				AttributeID: *avAttrID,
				Value:       *avValue,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attributes with values: %w", err)
	}

	out := make([]attribute.AttributeWithValues, 0, len(order))
	for _, id := range order {
		out = append(out, *index[id])
	}
	return out, nil
}

func (r *AttributeRepository) CreateAttributeValue(ctx context.Context, v *attribute.AttributeValue) error {
	const q = `
		INSERT INTO attribute_values (attribute_id, value)
		VALUES ($1, $2)
		RETURNING id`

	err := r.db.QueryRow(ctx, q, v.AttributeID, v.Value).Scan(&v.ID)
	if err != nil {
		return mapAttributeError(err)
	}
	return nil
}

func (r *AttributeRepository) GetAttributeValueByID(ctx context.Context, id int64) (*attribute.AttributeValue, error) {
	var v attribute.AttributeValue
	err := r.db.QueryRow(ctx,
		`SELECT id, attribute_id, value FROM attribute_values WHERE id = $1`, id,
	).Scan(&v.ID, &v.AttributeID, &v.Value)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, attribute.ErrValueNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get attribute value: %w", err)
	}
	return &v, nil
}

func (r *AttributeRepository) ListAttributeValues(ctx context.Context, attributeID int64) ([]attribute.AttributeValue, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, attribute_id, value FROM attribute_values WHERE attribute_id = $1 ORDER BY value`,
		attributeID,
	)
	if err != nil {
		return nil, fmt.Errorf("list attribute values: %w", err)
	}
	defer rows.Close()

	var out []attribute.AttributeValue
	for rows.Next() {
		var v attribute.AttributeValue
		if err := rows.Scan(&v.ID, &v.AttributeID, &v.Value); err != nil {
			return nil, fmt.Errorf("scan attribute value: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attribute values: %w", err)
	}
	return out, nil
}

func (r *AttributeRepository) DeleteAttributeValue(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM attribute_values WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete attribute value: %w", err)
	}
	if result.RowsAffected() == 0 {
		return attribute.ErrValueNotFound
	}
	return nil
}

func (r *AttributeRepository) SetProductAttributeValues(ctx context.Context, tx pgx.Tx, productID int64, valueIDs []int64) error {
	if tx != nil {
		return setProductAttributeValuesTx(ctx, tx, productID, valueIDs)
	}

	localTx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = localTx.Rollback(ctx) }()

	if err := setProductAttributeValuesTx(ctx, localTx, productID, valueIDs); err != nil {
		return err
	}
	return localTx.Commit(ctx)
}

func (r *AttributeRepository) GetProductAttributeValues(ctx context.Context, productID int64) ([]attribute.AttributeValue, error) {
	const q = `
		SELECT av.id, av.attribute_id, av.value
		FROM attribute_values av
		JOIN product_attribute_values pav ON pav.attribute_value_id = av.id
		WHERE pav.product_id = $1
		ORDER BY av.attribute_id, av.value`

	rows, err := r.db.Query(ctx, q, productID)
	if err != nil {
		return nil, fmt.Errorf("get product attribute values: %w", err)
	}
	defer rows.Close()

	var out []attribute.AttributeValue
	for rows.Next() {
		var v attribute.AttributeValue
		if err := rows.Scan(&v.ID, &v.AttributeID, &v.Value); err != nil {
			return nil, fmt.Errorf("scan attribute value: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product attribute values: %w", err)
	}
	return out, nil
}

func setProductAttributeValuesTx(ctx context.Context, tx pgx.Tx, productID int64, valueIDs []int64) error {
	if _, err := tx.Exec(ctx,
		`DELETE FROM product_attribute_values WHERE product_id = $1`, productID,
	); err != nil {
		return fmt.Errorf("clear product attribute values: %w", err)
	}
	for _, id := range valueIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_attribute_values (product_id, attribute_value_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			productID, id,
		); err != nil {
			return fmt.Errorf("insert product attribute value %d: %w", id, err)
		}
	}
	return nil
}

func mapAttributeError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		if strings.Contains(pgErr.ConstraintName, "slug") {
			return attribute.ErrSlugConflict
		}
		if strings.Contains(pgErr.ConstraintName, "attribute_values") {
			return attribute.ErrValueConflict
		}
	}
	return fmt.Errorf("attribute repository: %w", err)
}
