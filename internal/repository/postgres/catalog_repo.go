// internal/repository/postgres/catalog_repo.go
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"diary-service/internal/domain/catalog"
	xerrors "diary-service/internal/pkg/errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// =============================================================================
// CategoryRepository
// =============================================================================

// CategoryRepository implements catalog.CategoryRepository.
type CategoryRepository struct {
	db *pgxpool.Pool
}

// NewCategoryRepository creates a new CategoryRepository.
func NewCategoryRepository(db *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) CreateCategory(ctx context.Context, c *catalog.Category) error {
	query := `
		INSERT INTO product_categories (name, slug, parent_id, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	err := r.db.QueryRow(ctx, query, c.Name, c.Slug, c.ParentID, c.IsActive).
		Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}
	return nil
}

func (r *CategoryRepository) GetCategoryByID(ctx context.Context, id int64) (*catalog.Category, error) {
	query := `
		SELECT id, name, slug, parent_id, is_active, created_at
		FROM product_categories WHERE id = $1
	`
	var c catalog.Category
	err := r.db.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.IsActive, &c.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get category: %w", err)
	}
	return &c, nil
}

func (r *CategoryRepository) GetCategoryBySlug(ctx context.Context, slug string) (*catalog.Category, error) {
	query := `
		SELECT id, name, slug, parent_id, is_active, created_at
		FROM product_categories WHERE slug = $1
	`
	var c catalog.Category
	err := r.db.QueryRow(ctx, query, slug).Scan(
		&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.IsActive, &c.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get category by slug: %w", err)
	}
	return &c, nil
}

func (r *CategoryRepository) UpdateCategory(ctx context.Context, c *catalog.Category) error {
	query := `
		UPDATE product_categories
		SET name = $1, slug = $2, parent_id = $3, is_active = $4
		WHERE id = $5
	`
	result, err := r.db.Exec(ctx, query, c.Name, c.Slug, c.ParentID, c.IsActive, c.ID)
	if err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}
	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}
	return nil
}

func (r *CategoryRepository) DeleteCategory(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM product_categories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}
	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}
	return nil
}

func (r *CategoryRepository) ListCategories(ctx context.Context) ([]catalog.Category, error) {
	query := `
		SELECT id, name, slug, parent_id, is_active, created_at
		FROM product_categories
		ORDER BY name
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	defer rows.Close()

	var categories []catalog.Category
	for rows.Next() {
		var c catalog.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (r *CategoryRepository) GetCategoryAncestors(ctx context.Context, id int64) ([]catalog.CategoryClosure, error) {
	query := `
		SELECT ancestor_id, descendant_id, depth
		FROM category_closure
		WHERE descendant_id = $1
		ORDER BY depth DESC
	`
	return r.queryClosure(ctx, query, id)
}

func (r *CategoryRepository) GetCategoryDescendants(ctx context.Context, id int64) ([]catalog.CategoryClosure, error) {
	query := `
		SELECT ancestor_id, descendant_id, depth
		FROM category_closure
		WHERE ancestor_id = $1
		ORDER BY depth
	`
	return r.queryClosure(ctx, query, id)
}

func (r *CategoryRepository) queryClosure(ctx context.Context, query string, id int64) ([]catalog.CategoryClosure, error) {
	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query closure: %w", err)
	}
	defer rows.Close()

	var result []catalog.CategoryClosure
	for rows.Next() {
		var cc catalog.CategoryClosure
		if err := rows.Scan(&cc.AncestorID, &cc.DescendantID, &cc.Depth); err != nil {
			return nil, fmt.Errorf("failed to scan closure: %w", err)
		}
		result = append(result, cc)
	}
	return result, nil
}

// =============================================================================
// BrandRepository
// =============================================================================

// BrandRepository implements catalog.BrandRepository.
type BrandRepository struct {
	db *pgxpool.Pool
}

// NewBrandRepository creates a new BrandRepository.
func NewBrandRepository(db *pgxpool.Pool) *BrandRepository {
	return &BrandRepository{db: db}
}

func (r *BrandRepository) CreateBrand(ctx context.Context, b *catalog.Brand) error {
	query := `
		INSERT INTO product_brands (name, slug, logo_url, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	err := r.db.QueryRow(ctx, query, b.Name, b.Slug, b.LogoURL, b.IsActive).
		Scan(&b.ID, &b.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create brand: %w", err)
	}
	return nil
}

func (r *BrandRepository) GetBrandByID(ctx context.Context, id int64) (*catalog.Brand, error) {
	query := `SELECT id, name, slug, logo_url, is_active, created_at FROM product_brands WHERE id = $1`
	return r.scanBrand(ctx, query, id)
}

func (r *BrandRepository) GetBrandBySlug(ctx context.Context, slug string) (*catalog.Brand, error) {
	query := `SELECT id, name, slug, logo_url, is_active, created_at FROM product_brands WHERE slug = $1`
	return r.scanBrand(ctx, query, slug)
}

func (r *BrandRepository) scanBrand(ctx context.Context, query string, arg interface{}) (*catalog.Brand, error) {
	var b catalog.Brand
	err := r.db.QueryRow(ctx, query, arg).Scan(&b.ID, &b.Name, &b.Slug, &b.LogoURL, &b.IsActive, &b.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get brand: %w", err)
	}
	return &b, nil
}

func (r *BrandRepository) UpdateBrand(ctx context.Context, b *catalog.Brand) error {
	query := `
		UPDATE product_brands
		SET name = $1, slug = $2, logo_url = $3, is_active = $4
		WHERE id = $5
	`
	result, err := r.db.Exec(ctx, query, b.Name, b.Slug, b.LogoURL, b.IsActive, b.ID)
	if err != nil {
		return fmt.Errorf("failed to update brand: %w", err)
	}
	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}
	return nil
}

func (r *BrandRepository) DeleteBrand(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM product_brands WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete brand: %w", err)
	}
	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}
	return nil
}

func (r *BrandRepository) ListBrands(ctx context.Context, activeOnly bool) ([]catalog.Brand, error) {
	query := `SELECT id, name, slug, logo_url, is_active, created_at FROM product_brands`
	if activeOnly {
		query += ` WHERE is_active = TRUE`
	}
	query += ` ORDER BY name`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list brands: %w", err)
	}
	defer rows.Close()

	var brands []catalog.Brand
	for rows.Next() {
		var b catalog.Brand
		if err := rows.Scan(&b.ID, &b.Name, &b.Slug, &b.LogoURL, &b.IsActive, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan brand: %w", err)
		}
		brands = append(brands, b)
	}
	return brands, nil
}

// =============================================================================
// TagRepository
// =============================================================================

// TagRepository implements catalog.TagRepository.
type TagRepository struct {
	db *pgxpool.Pool
}

// NewTagRepository creates a new TagRepository.
func NewTagRepository(db *pgxpool.Pool) *TagRepository {
	return &TagRepository{db: db}
}

// FindOrCreateByName looks up a tag by name (case-insensitive) and creates it
// if it does not exist.  The slug is derived by lower-casing and replacing
// spaces with hyphens.
func (r *TagRepository) FindOrCreateByName(ctx context.Context, name string) (*catalog.Tag, error) {
	slug := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "-"))

	// Try to find existing
	const selectQ = `SELECT id, name, slug FROM tags WHERE LOWER(name) = LOWER($1)`
	var t catalog.Tag
	err := r.db.QueryRow(ctx, selectQ, name).Scan(&t.ID, &t.Name, &t.Slug)
	if err == nil {
		return &t, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to look up tag: %w", err)
	}

	// Create new; ON CONFLICT handles the rare race-condition where another
	// request inserts the same tag between our SELECT and INSERT.
	const insertQ = `INSERT INTO tags (name, slug) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET slug = EXCLUDED.slug RETURNING id, name, slug`
	err = r.db.QueryRow(ctx, insertQ, name, slug).Scan(&t.ID, &t.Name, &t.Slug)
	if err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}
	return &t, nil
}

func (r *TagRepository) GetTagByID(ctx context.Context, id int64) (*catalog.Tag, error) {
	var t catalog.Tag
	err := r.db.QueryRow(ctx, `SELECT id, name, slug FROM tags WHERE id = $1`, id).
		Scan(&t.ID, &t.Name, &t.Slug)
	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}
	return &t, nil
}

func (r *TagRepository) ListTags(ctx context.Context) ([]catalog.Tag, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name, slug FROM tags ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	defer rows.Close()

	var tags []catalog.Tag
	for rows.Next() {
		var t catalog.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, nil
}

// =============================================================================
// ProductRepository
// =============================================================================

// ProductRepository implements catalog.ProductRepository.
type ProductRepository struct {
	db *pgxpool.Pool
}

// NewProductRepository creates a new ProductRepository.
func NewProductRepository(db *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) CreateProduct(ctx context.Context, p *catalog.Product) error {
	query := `
		INSERT INTO products
			(name, slug, description, short_description, brand_id, base_price,
			 status, is_featured, is_digital, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, rating, review_count, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		p.Name, p.Slug, p.Description, p.ShortDescription, p.BrandID, p.BasePrice,
		p.Status, p.IsFeatured, p.IsDigital, p.CreatedBy,
	).Scan(&p.ID, &p.Rating, &p.ReviewCount, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}
	return nil
}

func (r *ProductRepository) GetProductByID(ctx context.Context, id int64) (*catalog.Product, error) {
	query := `
		SELECT id, name, slug, description, short_description, brand_id, base_price,
		       status, is_featured, is_digital, rating, review_count, created_by,
		       created_at, updated_at
		FROM products WHERE id = $1
	`
	return r.scanProduct(ctx, query, id)
}

func (r *ProductRepository) GetProductBySlug(ctx context.Context, slug string) (*catalog.Product, error) {
	query := `
		SELECT id, name, slug, description, short_description, brand_id, base_price,
		       status, is_featured, is_digital, rating, review_count, created_by,
		       created_at, updated_at
		FROM products WHERE slug = $1
	`
	return r.scanProduct(ctx, query, slug)
}

func (r *ProductRepository) scanProduct(ctx context.Context, query string, arg interface{}) (*catalog.Product, error) {
	var p catalog.Product
	err := r.db.QueryRow(ctx, query, arg).Scan(
		&p.ID, &p.Name, &p.Slug, &p.Description, &p.ShortDescription, &p.BrandID,
		&p.BasePrice, &p.Status, &p.IsFeatured, &p.IsDigital, &p.Rating,
		&p.ReviewCount, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get product: %w", err)
	}
	return &p, nil
}

func (r *ProductRepository) UpdateProduct(ctx context.Context, p *catalog.Product) error {
	query := `
		UPDATE products
		SET name              = $1,
		    slug              = $2,
		    description       = $3,
		    short_description = $4,
		    brand_id          = $5,
		    base_price        = $6,
		    status            = $7,
		    is_featured       = $8,
		    is_digital        = $9,
		    updated_at        = NOW()
		WHERE id = $10
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query,
		p.Name, p.Slug, p.Description, p.ShortDescription, p.BrandID,
		p.BasePrice, p.Status, p.IsFeatured, p.IsDigital, p.ID,
	).Scan(&p.UpdatedAt)
	if err == sql.ErrNoRows {
		return xerrors.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}
	return nil
}

func (r *ProductRepository) DeleteProduct(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}
	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}
	return nil
}

func (r *ProductRepository) ListProducts(ctx context.Context, limit, offset int) ([]catalog.Product, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM products`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	query := `
		SELECT id, name, slug, description, short_description, brand_id, base_price,
		       status, is_featured, is_digital, rating, review_count, created_by,
		       created_at, updated_at
		FROM products
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list products: %w", err)
	}
	defer rows.Close()

	var products []catalog.Product
	for rows.Next() {
		var p catalog.Product
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Slug, &p.Description, &p.ShortDescription, &p.BrandID,
			&p.BasePrice, &p.Status, &p.IsFeatured, &p.IsDigital, &p.Rating,
			&p.ReviewCount, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, p)
	}
	return products, total, nil
}

// ---- Category mapping ----

func (r *ProductRepository) AddProductCategory(ctx context.Context, productID, categoryID int64) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO product_category_map (product_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		productID, categoryID,
	)
	if err != nil {
		return fmt.Errorf("failed to add product category: %w", err)
	}
	return nil
}

func (r *ProductRepository) RemoveProductCategory(ctx context.Context, productID, categoryID int64) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM product_category_map WHERE product_id = $1 AND category_id = $2`,
		productID, categoryID,
	)
	if err != nil {
		return fmt.Errorf("failed to remove product category: %w", err)
	}
	return nil
}

func (r *ProductRepository) GetProductCategories(ctx context.Context, productID int64) ([]catalog.Category, error) {
	query := `
		SELECT pc.id, pc.name, pc.slug, pc.parent_id, pc.is_active, pc.created_at
		FROM product_categories pc
		JOIN product_category_map pcm ON pcm.category_id = pc.id
		WHERE pcm.product_id = $1
		ORDER BY pc.name
	`
	rows, err := r.db.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product categories: %w", err)
	}
	defer rows.Close()

	var categories []catalog.Category
	for rows.Next() {
		var c catalog.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, c)
	}
	return categories, nil
}

// ---- Tag linking (transactional find-or-create) ----

// SetProductTags replaces the complete set of tags for a product.  For each
// tag name that does not yet exist it is created inside the same transaction.
func (r *ProductRepository) SetProductTags(ctx context.Context, productID int64, tagNames []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Remove existing links
	if _, err := tx.Exec(ctx, `DELETE FROM product_tags WHERE product_id = $1`, productID); err != nil {
		return fmt.Errorf("failed to clear product tags: %w", err)
	}

	for _, name := range tagNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

		// Find or create tag within the transaction
		var tagID int64
		err := tx.QueryRow(ctx,
			`SELECT id FROM tags WHERE LOWER(name) = LOWER($1)`, name,
		).Scan(&tagID)
		if err == sql.ErrNoRows {
			err = tx.QueryRow(ctx,
				`INSERT INTO tags (name, slug) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET slug = EXCLUDED.slug RETURNING id`,
				name, slug,
			).Scan(&tagID)
		}
		if err != nil {
			return fmt.Errorf("failed to find or create tag %q: %w", name, err)
		}

		// Link to product
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_tags (product_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			productID, tagID,
		); err != nil {
			return fmt.Errorf("failed to link tag to product: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *ProductRepository) GetProductTags(ctx context.Context, productID int64) ([]catalog.Tag, error) {
	query := `
		SELECT t.id, t.name, t.slug
		FROM tags t
		JOIN product_tags pt ON pt.tag_id = t.id
		WHERE pt.product_id = $1
		ORDER BY t.name
	`
	rows, err := r.db.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product tags: %w", err)
	}
	defer rows.Close()

	var tags []catalog.Tag
	for rows.Next() {
		var t catalog.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, nil
}

// ---- Images ----

func (r *ProductRepository) AddProductImage(ctx context.Context, img *catalog.ProductImage) error {
	query := `
		INSERT INTO product_images (product_id, image_url, is_primary, sort_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	err := r.db.QueryRow(ctx, query, img.ProductID, img.ImageURL, img.IsPrimary, img.SortOrder).
		Scan(&img.ID, &img.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to add product image: %w", err)
	}
	return nil
}

func (r *ProductRepository) GetProductImages(ctx context.Context, productID int64) ([]catalog.ProductImage, error) {
	query := `
		SELECT id, product_id, image_url, is_primary, sort_order, created_at
		FROM product_images
		WHERE product_id = $1
		ORDER BY is_primary DESC, sort_order
	`
	rows, err := r.db.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product images: %w", err)
	}
	defer rows.Close()

	var images []catalog.ProductImage
	for rows.Next() {
		var img catalog.ProductImage
		if err := rows.Scan(&img.ID, &img.ProductID, &img.ImageURL, &img.IsPrimary, &img.SortOrder, &img.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan product image: %w", err)
		}
		images = append(images, img)
	}
	return images, nil
}

func (r *ProductRepository) DeleteProductImage(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM product_images WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete product image: %w", err)
	}
	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}
	return nil
}

// SetPrimaryImage marks one image as primary and clears the flag on all others
// for the same product.
func (r *ProductRepository) SetPrimaryImage(ctx context.Context, productID, imageID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`UPDATE product_images SET is_primary = FALSE WHERE product_id = $1`, productID,
	); err != nil {
		return fmt.Errorf("failed to clear primary images: %w", err)
	}

	result, err := tx.Exec(ctx,
		`UPDATE product_images SET is_primary = TRUE WHERE id = $1 AND product_id = $2`,
		imageID, productID,
	)
	if err != nil {
		return fmt.Errorf("failed to set primary image: %w", err)
	}
	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return tx.Commit(ctx)
}

// ---- Product-level attribute values ----

// SetProductAttributeValues replaces the attribute values linked to a product.
func (r *ProductRepository) SetProductAttributeValues(ctx context.Context, productID int64, attributeValueIDs []int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`DELETE FROM product_attribute_values WHERE product_id = $1`, productID,
	); err != nil {
		return fmt.Errorf("failed to clear product attribute values: %w", err)
	}

	for _, avID := range attributeValueIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_attribute_values (product_id, attribute_value_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			productID, avID,
		); err != nil {
			return fmt.Errorf("failed to insert product attribute value: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *ProductRepository) GetProductAttributeValues(ctx context.Context, productID int64) ([]catalog.AttributeValue, error) {
	query := `
		SELECT av.id, av.attribute_id, av.value
		FROM attribute_values av
		JOIN product_attribute_values pav ON pav.attribute_value_id = av.id
		WHERE pav.product_id = $1
		ORDER BY av.attribute_id, av.value
	`
	return r.scanAttributeValues(ctx, query, productID)
}

func (r *ProductRepository) scanAttributeValues(ctx context.Context, query string, arg interface{}) ([]catalog.AttributeValue, error) {
	rows, err := r.db.Query(ctx, query, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to query attribute values: %w", err)
	}
	defer rows.Close()

	var values []catalog.AttributeValue
	for rows.Next() {
		var av catalog.AttributeValue
		if err := rows.Scan(&av.ID, &av.AttributeID, &av.Value); err != nil {
			return nil, fmt.Errorf("failed to scan attribute value: %w", err)
		}
		values = append(values, av)
	}
	return values, nil
}

// =============================================================================
// AttributeRepository
// =============================================================================

// AttributeRepository implements catalog.AttributeRepository.
type AttributeRepository struct {
	db *pgxpool.Pool
}

// NewAttributeRepository creates a new AttributeRepository.
func NewAttributeRepository(db *pgxpool.Pool) *AttributeRepository {
	return &AttributeRepository{db: db}
}

func (r *AttributeRepository) CreateAttribute(ctx context.Context, a *catalog.Attribute) error {
	query := `
		INSERT INTO attributes (name, slug, is_variant_dimension)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	err := r.db.QueryRow(ctx, query, a.Name, a.Slug, a.IsVariantDimension).Scan(&a.ID)
	if err != nil {
		return fmt.Errorf("failed to create attribute: %w", err)
	}
	return nil
}

func (r *AttributeRepository) GetAttributeByID(ctx context.Context, id int64) (*catalog.Attribute, error) {
	query := `SELECT id, name, slug, is_variant_dimension FROM attributes WHERE id = $1`
	return r.scanAttribute(ctx, query, id)
}

func (r *AttributeRepository) GetAttributeBySlug(ctx context.Context, slug string) (*catalog.Attribute, error) {
	query := `SELECT id, name, slug, is_variant_dimension FROM attributes WHERE slug = $1`
	return r.scanAttribute(ctx, query, slug)
}

func (r *AttributeRepository) scanAttribute(ctx context.Context, query string, arg interface{}) (*catalog.Attribute, error) {
	var a catalog.Attribute
	err := r.db.QueryRow(ctx, query, arg).Scan(&a.ID, &a.Name, &a.Slug, &a.IsVariantDimension)
	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get attribute: %w", err)
	}
	return &a, nil
}

func (r *AttributeRepository) UpdateAttribute(ctx context.Context, a *catalog.Attribute) error {
	query := `
		UPDATE attributes
		SET name = $1, slug = $2, is_variant_dimension = $3
		WHERE id = $4
	`
	result, err := r.db.Exec(ctx, query, a.Name, a.Slug, a.IsVariantDimension, a.ID)
	if err != nil {
		return fmt.Errorf("failed to update attribute: %w", err)
	}
	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}
	return nil
}

func (r *AttributeRepository) DeleteAttribute(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM attributes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete attribute: %w", err)
	}
	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}
	return nil
}

func (r *AttributeRepository) ListAttributes(ctx context.Context) ([]catalog.Attribute, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name, slug, is_variant_dimension FROM attributes ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to list attributes: %w", err)
	}
	defer rows.Close()

	var attributes []catalog.Attribute
	for rows.Next() {
		var a catalog.Attribute
		if err := rows.Scan(&a.ID, &a.Name, &a.Slug, &a.IsVariantDimension); err != nil {
			return nil, fmt.Errorf("failed to scan attribute: %w", err)
		}
		attributes = append(attributes, a)
	}
	return attributes, nil
}

func (r *AttributeRepository) CreateAttributeValue(ctx context.Context, v *catalog.AttributeValue) error {
	query := `
		INSERT INTO attribute_values (attribute_id, value)
		VALUES ($1, $2)
		RETURNING id
	`
	err := r.db.QueryRow(ctx, query, v.AttributeID, v.Value).Scan(&v.ID)
	if err != nil {
		return fmt.Errorf("failed to create attribute value: %w", err)
	}
	return nil
}

func (r *AttributeRepository) GetAttributeValueByID(ctx context.Context, id int64) (*catalog.AttributeValue, error) {
	var v catalog.AttributeValue
	err := r.db.QueryRow(ctx,
		`SELECT id, attribute_id, value FROM attribute_values WHERE id = $1`, id,
	).Scan(&v.ID, &v.AttributeID, &v.Value)
	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get attribute value: %w", err)
	}
	return &v, nil
}

func (r *AttributeRepository) ListAttributeValues(ctx context.Context, attributeID int64) ([]catalog.AttributeValue, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, attribute_id, value FROM attribute_values WHERE attribute_id = $1 ORDER BY value`,
		attributeID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list attribute values: %w", err)
	}
	defer rows.Close()

	var values []catalog.AttributeValue
	for rows.Next() {
		var v catalog.AttributeValue
		if err := rows.Scan(&v.ID, &v.AttributeID, &v.Value); err != nil {
			return nil, fmt.Errorf("failed to scan attribute value: %w", err)
		}
		values = append(values, v)
	}
	return values, nil
}

func (r *AttributeRepository) DeleteAttributeValue(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM attribute_values WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete attribute value: %w", err)
	}
	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}
	return nil
}

// =============================================================================
// VariantRepository
// =============================================================================

// VariantRepository implements catalog.VariantRepository.
type VariantRepository struct {
	db *pgxpool.Pool
}

// NewVariantRepository creates a new VariantRepository.
func NewVariantRepository(db *pgxpool.Pool) *VariantRepository {
	return &VariantRepository{db: db}
}

func (r *VariantRepository) CreateVariant(ctx context.Context, v *catalog.ProductVariant) error {
	query := `
		INSERT INTO product_variants (product_id, sku, price, weight, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	err := r.db.QueryRow(ctx, query, v.ProductID, v.SKU, v.Price, v.Weight, v.IsActive).
		Scan(&v.ID, &v.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create variant: %w", err)
	}
	return nil
}

func (r *VariantRepository) GetVariantByID(ctx context.Context, id int64) (*catalog.ProductVariant, error) {
	query := `SELECT id, product_id, sku, price, weight, is_active, created_at FROM product_variants WHERE id = $1`
	return r.scanVariant(ctx, query, id)
}

func (r *VariantRepository) GetVariantBySKU(ctx context.Context, sku string) (*catalog.ProductVariant, error) {
	query := `SELECT id, product_id, sku, price, weight, is_active, created_at FROM product_variants WHERE sku = $1`
	return r.scanVariant(ctx, query, sku)
}

func (r *VariantRepository) scanVariant(ctx context.Context, query string, arg interface{}) (*catalog.ProductVariant, error) {
	var v catalog.ProductVariant
	err := r.db.QueryRow(ctx, query, arg).Scan(
		&v.ID, &v.ProductID, &v.SKU, &v.Price, &v.Weight, &v.IsActive, &v.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get variant: %w", err)
	}
	return &v, nil
}

func (r *VariantRepository) UpdateVariant(ctx context.Context, v *catalog.ProductVariant) error {
	query := `
		UPDATE product_variants
		SET sku = $1, price = $2, weight = $3, is_active = $4
		WHERE id = $5
	`
	result, err := r.db.Exec(ctx, query, v.SKU, v.Price, v.Weight, v.IsActive, v.ID)
	if err != nil {
		return fmt.Errorf("failed to update variant: %w", err)
	}
	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}
	return nil
}

func (r *VariantRepository) DeleteVariant(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM product_variants WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete variant: %w", err)
	}
	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}
	return nil
}

func (r *VariantRepository) ListVariantsByProduct(ctx context.Context, productID int64) ([]catalog.ProductVariant, error) {
	query := `
		SELECT id, product_id, sku, price, weight, is_active, created_at
		FROM product_variants
		WHERE product_id = $1
		ORDER BY id
	`
	rows, err := r.db.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to list variants: %w", err)
	}
	defer rows.Close()

	var variants []catalog.ProductVariant
	for rows.Next() {
		var v catalog.ProductVariant
		if err := rows.Scan(&v.ID, &v.ProductID, &v.SKU, &v.Price, &v.Weight, &v.IsActive, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan variant: %w", err)
		}
		variants = append(variants, v)
	}
	return variants, nil
}

// SetVariantAttributeValues replaces the attribute values linked to a variant.
func (r *VariantRepository) SetVariantAttributeValues(ctx context.Context, variantID int64, attributeValueIDs []int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`DELETE FROM variant_attribute_values WHERE variant_id = $1`, variantID,
	); err != nil {
		return fmt.Errorf("failed to clear variant attribute values: %w", err)
	}

	for _, avID := range attributeValueIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO variant_attribute_values (variant_id, attribute_value_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			variantID, avID,
		); err != nil {
			return fmt.Errorf("failed to insert variant attribute value: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *VariantRepository) GetVariantAttributeValues(ctx context.Context, variantID int64) ([]catalog.AttributeValue, error) {
	query := `
		SELECT av.id, av.attribute_id, av.value
		FROM attribute_values av
		JOIN variant_attribute_values vav ON vav.attribute_value_id = av.id
		WHERE vav.variant_id = $1
		ORDER BY av.attribute_id, av.value
	`
	rows, err := r.db.Query(ctx, query, variantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get variant attribute values: %w", err)
	}
	defer rows.Close()

	var values []catalog.AttributeValue
	for rows.Next() {
		var av catalog.AttributeValue
		if err := rows.Scan(&av.ID, &av.AttributeID, &av.Value); err != nil {
			return nil, fmt.Errorf("failed to scan attribute value: %w", err)
		}
		values = append(values, av)
	}
	return values, nil
}
