package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"zentora-service/internal/domain/inventory"
	"zentora-service/internal/domain/product"
	"zentora-service/internal/domain/variant"
	xerrors "zentora-service/internal/pkg/errors"
	productsearchsvc "zentora-service/internal/service/productsearch"
	productsearchrepo "zentora-service/internal/repository/productsearch"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductRepository struct {
	db            *pgxpool.Pool
	attributeRepo *AttributeRepository
	brandRepo     *BrandRepository
	categoryRepo  *CategoryRepository
	discountRepo  *DiscountRepository
	inventoryRepo *InventoryRepository
	tagRepo       *TagRepository
	variantRepo   *VariantRepository
	searchRepo productsearchrepo.Repository
}

func NewProductRepository(
	db *pgxpool.Pool,
	attributeRepo *AttributeRepository,
	brandRepo *BrandRepository,
	categoryRepo *CategoryRepository,
	discountRepo *DiscountRepository,
	inventoryRepo *InventoryRepository,
	tagRepo *TagRepository,
	variantRepo *VariantRepository,
	searchRepo productsearchrepo.Repository,
) *ProductRepository {
	return &ProductRepository{
		db:            db,
		attributeRepo: attributeRepo,
		brandRepo:     brandRepo,
		categoryRepo:  categoryRepo,
		discountRepo:  discountRepo,
		inventoryRepo: inventoryRepo,
		tagRepo:       tagRepo,
		variantRepo:   variantRepo,
		searchRepo: searchRepo,
	}
}

type CreateProductTxInput struct {
	Product           *product.Product
	CategoryIDs       []int64
	TagNames          []string
	Images            []product.Image
	AttributeValueIDs []int64
	Variants          []CreateVariantTxInput
	DiscountID        *int64
}

type CreateVariantTxInput struct {
	SKU               string
	Price             float64
	Weight            *float64
	IsActive          bool
	AttributeValueIDs []int64
	Quantity          int
	LocationID        int64
}

func (r *ProductRepository) CreateProductTx(ctx context.Context, in *CreateProductTxInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := r.buildProductGraph(ctx, tx, in); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *ProductRepository) buildProductGraph(ctx context.Context, tx pgx.Tx, in *CreateProductTxInput) error {
	if err := insertProductTx(ctx, tx, in.Product); err != nil {
		return err
	}
	productID := in.Product.ID

	// NEW: create product search doc in same tx
	if r.searchRepo != nil {
		doc := productsearchsvc.BuildSearchDocument(in.Product)
		if err := r.searchRepo.UpsertForProductTx(ctx, tx, productID, doc); err != nil {
			return fmt.Errorf("create product search document: %w", err)
		}
	}

	if err := setProductCategoriesTx(ctx, tx, productID, in.CategoryIDs); err != nil {
		return fmt.Errorf("set product categories: %w", err)
	}

	if err := setProductTagsTx(ctx, tx, productID, in.TagNames); err != nil {
		return fmt.Errorf("set product tags: %w", err)
	}

	for i := range in.Images {
		in.Images[i].ProductID = productID
		if err := insertProductImageTx(ctx, tx, &in.Images[i]); err != nil {
			return fmt.Errorf("insert image: %w", err)
		}
	}

	if len(in.AttributeValueIDs) > 0 {
		if err := setProductAttributeValuesTx(ctx, tx, productID, in.AttributeValueIDs); err != nil {
			return fmt.Errorf("set product attribute values: %w", err)
		}
	}

	for i := range in.Variants {
		vi := &in.Variants[i]
		v := buildVariantEntity(productID, vi)

		if err := CreateVariantWithAttributesTx(ctx, tx, v, vi.AttributeValueIDs); err != nil {
			return fmt.Errorf("create variant %q: %w", vi.SKU, err)
		}

		if vi.Quantity > 0 && vi.LocationID > 0 {
			inventoryItem := &inventory.Item{
				VariantID:    v.ID,
				LocationID:   vi.LocationID,
				AvailableQty: vi.Quantity,
			}
			if err := r.inventoryRepo.UpsertItem(ctx, tx, inventoryItem); err != nil {
				return fmt.Errorf("upsert inventory for variant %q: %w", vi.SKU, err)
			}
		}
	}

	if in.DiscountID != nil {
		if _, err := tx.Exec(ctx,
			`INSERT INTO discount_targets (discount_id, target_type, target_id)
			 VALUES ($1, 'product', $2) ON CONFLICT DO NOTHING`,
			*in.DiscountID, productID,
		); err != nil {
			return fmt.Errorf("link discount: %w", err)
		}
	}

	return nil
}

func (r *ProductRepository) GetProductByID(ctx context.Context, id int64) (*product.Product, error) {
	const q = `
		SELECT id, name, slug, description, short_description, brand_id, base_price,
		       status, is_featured, is_digital, rating, review_count, created_by,
		       created_at, updated_at
		FROM products WHERE id = $1`
	return scanProductOne(ctx, r.db, q, id)
}

func (r *ProductRepository) GetProductBySlug(ctx context.Context, slug string) (*product.Product, error) {
	const q = `
		SELECT id, name, slug, description, short_description, brand_id, base_price,
		       status, is_featured, is_digital, rating, review_count, created_by,
		       created_at, updated_at
		FROM products WHERE slug = $1`
	return scanProductOne(ctx, r.db, q, slug)
}

func (r *ProductRepository) UpdateProduct(ctx context.Context, p *product.Product) error {
	const q = `
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
		RETURNING updated_at`

	err := r.db.QueryRow(ctx, q,
		p.Name, p.Slug, p.Description, p.ShortDescription,
		p.BrandID, p.BasePrice, p.Status, p.IsFeatured, p.IsDigital, p.ID,
	).Scan(&p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return product.ErrNotFound
	}
	if err != nil {
		return mapProductError(err)
	}
	return nil
}

func (r *ProductRepository) DeleteProduct(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if result.RowsAffected() == 0 {
		return product.ErrNotFound
	}
	return nil
}

func (r *ProductRepository) ListProducts(ctx context.Context, req *product.ListRequest) ([]product.Product, int64, error) {
	where, args := buildProductWhere(req.Filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM products`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	limit := req.PageSize
	offset := (req.Page - 1) * req.PageSize
	args = append(args, limit, offset)

	q := `SELECT id, name, slug, description, short_description, brand_id, base_price,
		       status, is_featured, is_digital, rating, review_count, created_by,
		       created_at, updated_at
		FROM products` + where +
		fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	products, err := scanProducts(rows)
	return products, total, err
}


func buildProductWhere(f product.ListFilter) (string, []any) {
	var clauses []string
	var args []any
	idx := 1

	if f.Status != nil {
		clauses = append(clauses, fmt.Sprintf("status = $%d", idx))
		args = append(args, *f.Status)
		idx++
	}
	if f.BrandID != nil {
		clauses = append(clauses, fmt.Sprintf("brand_id = $%d", idx))
		args = append(args, *f.BrandID)
		idx++
	}
	if f.IsFeatured != nil {
		clauses = append(clauses, fmt.Sprintf("is_featured = $%d", idx))
		args = append(args, *f.IsFeatured)
		idx++
	}
	if f.Search != nil {
		clauses = append(clauses, fmt.Sprintf("(name ILIKE $%d OR short_description ILIKE $%d)", idx, idx))
		args = append(args, "%"+*f.Search+"%")
		idx++
	}
	if f.CategoryID != nil {
		clauses = append(clauses, fmt.Sprintf(
			"id IN (SELECT product_id FROM product_category_map WHERE category_id = $%d)", idx,
		))
		args = append(args, *f.CategoryID)
		idx++
	}
	_ = idx
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}


func (r *ProductRepository) AddProductImage(ctx context.Context, img *product.Image) error {
	return insertProductImageTx(ctx, r.db, img)
}

func (r *ProductRepository) GetProductImages(ctx context.Context, productID int64) ([]product.Image, error) {
	const q = `
		SELECT id, product_id, image_url, is_primary, sort_order, created_at
		FROM product_images
		WHERE product_id = $1
		ORDER BY is_primary DESC, sort_order`

	rows, err := r.db.Query(ctx, q, productID)
	if err != nil {
		return nil, fmt.Errorf("get product images: %w", err)
	}
	defer rows.Close()
	return scanProductImages(rows)
}

func (r *ProductRepository) DeleteProductImage(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM product_images WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete product image: %w", err)
	}
	if result.RowsAffected() == 0 {
		return product.ErrImageNotFound
	}
	return nil
}

func (r *ProductRepository) SetPrimaryImage(ctx context.Context, productID, imageID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`UPDATE product_images SET is_primary = FALSE WHERE product_id = $1`, productID,
	); err != nil {
		return fmt.Errorf("clear primary: %w", err)
	}
	result, err := tx.Exec(ctx,
		`UPDATE product_images SET is_primary = TRUE WHERE id = $1 AND product_id = $2`,
		imageID, productID,
	)
	if err != nil {
		return fmt.Errorf("set primary: %w", err)
	}
	if result.RowsAffected() == 0 {
		return product.ErrImageNotFound
	}
	return tx.Commit(ctx)
}

type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func insertProductTx(ctx context.Context, tx pgx.Tx, p *product.Product) error {
	const q = `
		INSERT INTO products
			(name, slug, description, short_description, brand_id, base_price,
			 status, is_featured, is_digital, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, rating, review_count, created_at, updated_at`

	err := tx.QueryRow(ctx, q,
		p.Name, p.Slug, p.Description, p.ShortDescription,
		p.BrandID, p.BasePrice, p.Status, p.IsFeatured, p.IsDigital, p.CreatedBy,
	).Scan(&p.ID, &p.Rating, &p.ReviewCount, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return mapProductError(err)
	}
	return nil
}

func insertProductImageTx(ctx context.Context, ex rowQuerier, img *product.Image) error {
	const q = `
		INSERT INTO product_images (product_id, image_url, is_primary, sort_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	return ex.QueryRow(ctx, q, img.ProductID, img.ImageURL, img.IsPrimary, img.SortOrder).
		Scan(&img.ID, &img.CreatedAt)
}

func buildVariantEntity(productID int64, vi *CreateVariantTxInput) *variant.Variant {
	v := &variant.Variant{
		ProductID: productID,
		SKU:       vi.SKU,
		Price:     vi.Price,
		IsActive:  vi.IsActive,
	}
	if vi.Weight != nil {
		v.Weight = sql.NullFloat64{Float64: *vi.Weight, Valid: true}
	}
	return v
}

func scanProductOne(ctx context.Context, db interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, query string, arg any) (*product.Product, error) {
	var p product.Product
	err := db.QueryRow(ctx, query, arg).Scan(
		&p.ID, &p.Name, &p.Slug, &p.Description, &p.ShortDescription,
		&p.BrandID, &p.BasePrice, &p.Status, &p.IsFeatured, &p.IsDigital,
		&p.Rating, &p.ReviewCount, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, product.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	return &p, nil
}

func scanProducts(rows pgx.Rows) ([]product.Product, error) {
	var out []product.Product
	for rows.Next() {
		var p product.Product
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Slug, &p.Description, &p.ShortDescription,
			&p.BrandID, &p.BasePrice, &p.Status, &p.IsFeatured, &p.IsDigital,
			&p.Rating, &p.ReviewCount, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}
	return out, nil
}

func scanProductImages(rows pgx.Rows) ([]product.Image, error) {
	var out []product.Image
	for rows.Next() {
		var img product.Image
		if err := rows.Scan(
			&img.ID, &img.ProductID, &img.ImageURL, &img.IsPrimary, &img.SortOrder, &img.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan product image: %w", err)
		}
		out = append(out, img)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product images: %w", err)
	}
	return out, nil
}

func mapProductError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		if strings.Contains(pgErr.ConstraintName, "slug") {
			return product.ErrSlugConflict
		}
	}
	_ = xerrors.ErrNotFound
	return fmt.Errorf("product repository: %w", err)
}
