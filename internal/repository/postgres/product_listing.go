package postgres

import (
	"context"
	"fmt"
	"strings"

	"zentora-service/internal/domain/discovery"
	"zentora-service/internal/domain/product"

	"github.com/jackc/pgx/v5"
)

type CatalogSort string

const (
	CatalogSortNewArrivals CatalogSort = "new_arrivals"
	CatalogSortTrending    CatalogSort = "trending"
	CatalogSortBestSellers CatalogSort = "best_sellers"
	CatalogSortRating      CatalogSort = "rating"
	CatalogSortPriceAsc    CatalogSort = "price_asc"
	CatalogSortPriceDesc   CatalogSort = "price_desc"

	defaultCatalogPageSize = 20
	maxCatalogPageSize     = 100
)

func (r *ProductRepository) ListProductsForCatalog(
	ctx context.Context,
	req *product.ListRequest,
	sort string,
) ([]discovery.ProductCard, int64, error) {
	if req == nil {
		return nil, 0, fmt.Errorf("list products for catalog: nil request")
	}

	if req.Page < 1 {
		return nil, 0, fmt.Errorf("list products for catalog: page must be >= 1")
	}
	if req.PageSize <= 0 {
		req.PageSize = defaultCatalogPageSize
	}
	if req.PageSize > maxCatalogPageSize {
		req.PageSize = maxCatalogPageSize
	}

	where, args, joins := buildCatalogQueryParts(req.Filter)

	orderBy, needMetrics, err := catalogOrderBy(sort)
	if err != nil {
		return nil, 0, fmt.Errorf("list products for catalog: %w", err)
	}
	if needMetrics {
		joins = append(joins, "LEFT JOIN product_metrics pm ON pm.product_id = p.id")
	}

	joinSQL := ""
	if len(joins) > 0 {
		joinSQL = "\n" + strings.Join(joins, "\n")
	}

	countQ := `
		SELECT COUNT(DISTINCT p.id)
		FROM products p` + joinSQL + where

	var total int64
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count catalog products: %w", err)
	}

	limit := req.PageSize
	offset := (req.Page - 1) * req.PageSize
	limitPos := len(args) + 1
	offsetPos := len(args) + 2

	// idsQ := fmt.Sprintf(`
	// 	SELECT DISTINCT p.id
	// 	FROM products p%s%s
	// 	ORDER BY %s
	// 	LIMIT $%d OFFSET $%d`,
	// 	joinSQL,
	// 	where,
	// 	orderBy,
	// 	limitPos,
	// 	offsetPos,
	// )
	idsQ := fmt.Sprintf(`
		SELECT p.id
		FROM products p%s%s
		GROUP BY p.id
		ORDER BY %s
		LIMIT $%d OFFSET $%d`,
		joinSQL,
		where,
		orderBy,
		limitPos,
		offsetPos,
	)

	idArgs := append(append([]any{}, args...), limit, offset)

	rows, err := r.db.Query(ctx, idsQ, idArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list catalog product ids: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, 0, fmt.Errorf("scan catalog product id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate catalog product ids: %w", err)
	}

	cards, err := hydrateDiscoveryProductCards(ctx, r.db, ids)
	if err != nil {
		return nil, 0, fmt.Errorf("hydrate catalog product cards: %w", err)
	}

	return cards, total, nil
}

func catalogOrderBy(sort string) (string, bool, error) {
	switch CatalogSort(strings.TrimSpace(strings.ToLower(sort))) {
	case "", CatalogSortNewArrivals:
		return "p.created_at DESC, p.id DESC", false, nil
	case CatalogSortTrending:
		return "COALESCE(pm.trending_score, 0) DESC, p.id DESC", true, nil
	case CatalogSortBestSellers:
		return "COALESCE(pm.weekly_purchases, 0) DESC, p.id DESC", true, nil
	case CatalogSortRating:
		return "p.rating DESC, p.review_count DESC, p.id DESC", false, nil
	case CatalogSortPriceAsc:
		return "p.base_price ASC, p.id DESC", false, nil
	case CatalogSortPriceDesc:
		return "p.base_price DESC, p.id DESC", false, nil
	default:
		return "", false, fmt.Errorf("invalid sort: %q", sort)
	}
}

func buildCatalogQueryParts(f product.ListFilter) (string, []any, []string) {
	var (
		clauses []string
		args    []any
		joins   []string
		idx     = 1
	)

	joins = append(joins, `
LEFT JOIN (
	WITH active_discounts AS (
		SELECT d.id, d.discount_type, d.value
		FROM discounts d
		WHERE d.is_active = TRUE
		  AND (d.starts_at IS NULL OR d.starts_at <= NOW())
		  AND (d.ends_at IS NULL OR d.ends_at >= NOW())
	),
	discount_candidates AS (
		SELECT p.id AS product_id,
			COALESCE(CASE
				WHEN ad.discount_type = 'percentage' THEN ad.value::DOUBLE PRECISION
				ELSE ((ad.value / NULLIF(p.base_price, 0)) * 100)::DOUBLE PRECISION
			END, 0)::DOUBLE PRECISION AS discount_percent
		FROM products p
		JOIN discount_targets dt
		  ON dt.target_type = 'product'
		 AND dt.target_id = p.id
		JOIN active_discounts ad ON ad.id = dt.discount_id

		UNION ALL

		SELECT p.id AS product_id,
			COALESCE(CASE
				WHEN ad.discount_type = 'percentage' THEN ad.value::DOUBLE PRECISION
				ELSE ((ad.value / NULLIF(p.base_price, 0)) * 100)::DOUBLE PRECISION
			END, 0)::DOUBLE PRECISION AS discount_percent
		FROM products p
		JOIN discount_targets dt
		  ON dt.target_type = 'brand'
		 AND dt.target_id = p.brand_id
		JOIN active_discounts ad ON ad.id = dt.discount_id

		UNION ALL

		SELECT p.id AS product_id,
			COALESCE(CASE
				WHEN ad.discount_type = 'percentage' THEN ad.value::DOUBLE PRECISION
				ELSE ((ad.value / NULLIF(p.base_price, 0)) * 100)::DOUBLE PRECISION
			END, 0)::DOUBLE PRECISION AS discount_percent
		FROM products p
		JOIN product_category_map pcmx ON pcmx.product_id = p.id
		JOIN discount_targets dt
		  ON dt.target_type = 'category'
		 AND dt.target_id = pcmx.category_id
		JOIN active_discounts ad ON ad.id = dt.discount_id
	),
	best_discounts AS (
		SELECT dc.product_id, COALESCE(MAX(dc.discount_percent), 0)::DOUBLE PRECISION AS discount_percent
		FROM discount_candidates dc
		GROUP BY dc.product_id
	)
	SELECT * FROM best_discounts
) bd ON bd.product_id = p.id
`)

	effectivePriceExpr := "ROUND((p.base_price * (1 - (COALESCE(bd.discount_percent, 0) / 100.0)))::NUMERIC, 2)::DOUBLE PRECISION"

	if f.InStockOnly {
		joins = append(joins, `
LEFT JOIN (
	SELECT pv.product_id,
	       COALESCE(SUM(ii.available_qty - ii.reserved_qty), 0) AS available_inventory
	FROM product_variants pv
	LEFT JOIN inventory_items ii ON ii.variant_id = pv.id
	WHERE pv.is_active = TRUE
	GROUP BY pv.product_id
) inv ON inv.product_id = p.id
`)
		clauses = append(clauses, "COALESCE(inv.available_inventory, 0) > 0")
	}

	if f.Status != nil {
		clauses = append(clauses, fmt.Sprintf("p.status = $%d", idx))
		args = append(args, *f.Status)
		idx++
	}

	if f.BrandID != nil {
		clauses = append(clauses, fmt.Sprintf("p.brand_id = $%d", idx))
		args = append(args, *f.BrandID)
		idx++
	}

	if len(f.BrandIDs) > 0 {
		clauses = append(clauses, fmt.Sprintf("p.brand_id = ANY($%d::BIGINT[])", idx))
		args = append(args, f.BrandIDs)
		idx++
	}

	if f.IsFeatured != nil {
		clauses = append(clauses, fmt.Sprintf("p.is_featured = $%d", idx))
		args = append(args, *f.IsFeatured)
		idx++
	}

	if f.Search != nil && strings.TrimSpace(*f.Search) != "" {
		clauses = append(clauses, fmt.Sprintf("(p.name ILIKE $%d OR p.short_description ILIKE $%d)", idx, idx))
		args = append(args, "%"+*f.Search+"%")
		idx++
	}

	if f.CategoryID != nil {
		joins = append(joins, "JOIN product_category_map pcm ON pcm.product_id = p.id")
		clauses = append(clauses, fmt.Sprintf("pcm.category_id = $%d", idx))
		args = append(args, *f.CategoryID)
		idx++
	}

	if len(f.TagIDs) > 0 {
		joins = append(joins, "JOIN product_tags pt ON pt.product_id = p.id")
		clauses = append(clauses, fmt.Sprintf("pt.tag_id = ANY($%d::BIGINT[])", idx))
		args = append(args, f.TagIDs)
		idx++
	}

	if f.PriceMin != nil {
		clauses = append(clauses, fmt.Sprintf("%s >= $%d::DOUBLE PRECISION", effectivePriceExpr, idx))
		args = append(args, *f.PriceMin)
		idx++
	}
	if f.PriceMax != nil {
		clauses = append(clauses, fmt.Sprintf("%s <= $%d::DOUBLE PRECISION", effectivePriceExpr, idx))
		args = append(args, *f.PriceMax)
		idx++
	}

	if f.MinRating != nil {
		clauses = append(clauses, fmt.Sprintf("p.rating::DOUBLE PRECISION >= $%d::DOUBLE PRECISION", idx))
		args = append(args, *f.MinRating)
		idx++
	}

	if f.DiscountOnly {
		clauses = append(clauses, "COALESCE(bd.discount_percent, 0) > 0")
	}

	if len(clauses) == 0 {
		return "", args, joins
	}
	return " WHERE " + strings.Join(clauses, " AND "), args, joins
}

type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func hydrateDiscoveryProductCards(
	ctx context.Context,
	db queryer,
	productIDs []int64,
) ([]discovery.ProductCard, error) {
	if len(productIDs) == 0 {
		return []discovery.ProductCard{}, nil
	}

	const lowStockThreshold = 5

	const q = `
		WITH requested_products AS (
			SELECT product_id, ord
			FROM unnest($1::BIGINT[]) WITH ORDINALITY AS rp(product_id, ord)
		),
		primary_images AS (
			SELECT DISTINCT ON (pi.product_id)
				pi.product_id,
				pi.image_url
			FROM product_images pi
			JOIN requested_products rp ON rp.product_id = pi.product_id
			ORDER BY pi.product_id, pi.is_primary DESC, pi.sort_order ASC, pi.id ASC
		),
		primary_categories AS (
			SELECT DISTINCT ON (pcm.product_id)
				pcm.product_id,
				pc.name
			FROM product_category_map pcm
			JOIN product_categories pc ON pc.id = pcm.category_id
			JOIN requested_products rp ON rp.product_id = pcm.product_id
			ORDER BY pcm.product_id, pc.name ASC, pc.id ASC
		),
		inventory_summary AS (
			SELECT pv.product_id,
				COALESCE(SUM(ii.available_qty - ii.reserved_qty), 0) AS available_inventory
			FROM product_variants pv
			LEFT JOIN inventory_items ii ON ii.variant_id = pv.id
			JOIN requested_products rp ON rp.product_id = pv.product_id
			WHERE pv.is_active = TRUE
			GROUP BY pv.product_id
		),
		active_discounts AS (
			SELECT d.id, d.discount_type, d.value
			FROM discounts d
			WHERE d.is_active = TRUE
			  AND (d.starts_at IS NULL OR d.starts_at <= NOW())
			  AND (d.ends_at IS NULL OR d.ends_at >= NOW())
		),
		discount_candidates AS (
			SELECT p.id AS product_id,
				COALESCE(CASE
					WHEN ad.discount_type = 'percentage' THEN ad.value::DOUBLE PRECISION
					ELSE ((ad.value / NULLIF(p.base_price, 0)) * 100)::DOUBLE PRECISION
				END, 0)::DOUBLE PRECISION AS discount_percent
			FROM requested_products rp
			JOIN products p ON p.id = rp.product_id
			JOIN discount_targets dt
			  ON dt.target_type = 'product'
			 AND dt.target_id = p.id
			JOIN active_discounts ad ON ad.id = dt.discount_id

			UNION ALL

			SELECT p.id AS product_id,
				COALESCE(CASE
					WHEN ad.discount_type = 'percentage' THEN ad.value::DOUBLE PRECISION
					ELSE ((ad.value / NULLIF(p.base_price, 0)) * 100)::DOUBLE PRECISION
				END, 0)::DOUBLE PRECISION AS discount_percent
			FROM requested_products rp
			JOIN products p ON p.id = rp.product_id
			JOIN discount_targets dt
			  ON dt.target_type = 'brand'
			 AND dt.target_id = p.brand_id
			JOIN active_discounts ad ON ad.id = dt.discount_id

			UNION ALL

			SELECT p.id AS product_id,
				COALESCE(CASE
					WHEN ad.discount_type = 'percentage' THEN ad.value::DOUBLE PRECISION
					ELSE ((ad.value / NULLIF(p.base_price, 0)) * 100)::DOUBLE PRECISION
				END, 0)::DOUBLE PRECISION AS discount_percent
			FROM requested_products rp
			JOIN products p ON p.id = rp.product_id
			JOIN product_category_map pcm ON pcm.product_id = p.id
			JOIN discount_targets dt
			  ON dt.target_type = 'category'
			 AND dt.target_id = pcm.category_id
			JOIN active_discounts ad ON ad.id = dt.discount_id
		),
		best_discounts AS (
			SELECT dc.product_id,
			       COALESCE(MAX(dc.discount_percent), 0)::DOUBLE PRECISION AS discount_percent
			FROM discount_candidates dc
			GROUP BY dc.product_id
		)
		SELECT
			p.id AS product_id,
			p.name,
			p.slug,
			COALESCE(pi.image_url, '') AS primary_image,
			ROUND((p.base_price * (1 - (COALESCE(bd.discount_percent, 0) / 100.0)))::NUMERIC, 2)::DOUBLE PRECISION AS price,
			COALESCE(bd.discount_percent, 0)::DOUBLE PRECISION AS discount,
			p.rating::DOUBLE PRECISION AS rating,
			p.review_count,
			CASE
				WHEN COALESCE(inv.available_inventory, 0) <= 0 THEN $2
				WHEN COALESCE(inv.available_inventory, 0) <= $3 THEN $4
				ELSE $5
			END AS inventory_status,
			COALESCE(pb.name, '') AS brand,
			COALESCE(pc.name, '') AS category
		FROM requested_products rp
		JOIN products p ON p.id = rp.product_id
		LEFT JOIN primary_images pi ON pi.product_id = p.id
		LEFT JOIN primary_categories pc ON pc.product_id = p.id
		LEFT JOIN product_brands pb ON pb.id = p.brand_id
		LEFT JOIN inventory_summary inv ON inv.product_id = p.id
		LEFT JOIN best_discounts bd ON bd.product_id = p.id
		ORDER BY rp.ord`

	rows, err := db.Query(
		ctx,
		q,
		productIDs,
		discovery.InventoryStatusOutOfStock,
		lowStockThreshold,
		discovery.InventoryStatusLowStock,
		discovery.InventoryStatusInStock,
	)
	if err != nil {
		return nil, fmt.Errorf("hydrate discovery product cards: %w", err)
	}
	defer rows.Close()

	var cards []discovery.ProductCard
	for rows.Next() {
		var card discovery.ProductCard
		if err := rows.Scan(
			&card.ProductID,
			&card.Name,
			&card.Slug,
			&card.PrimaryImage,
			&card.Price,
			&card.Discount,
			&card.Rating,
			&card.ReviewCount,
			&card.InventoryStatus,
			&card.Brand,
			&card.Category,
		); err != nil {
			return nil, fmt.Errorf("scan discovery product card: %w", err)
		}
		cards = append(cards, card)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate discovery product cards: %w", err)
	}
	return cards, nil
}