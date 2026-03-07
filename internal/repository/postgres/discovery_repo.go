package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"zentora-service/internal/domain/discovery"
	"zentora-service/internal/domain/product"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DiscoveryRepository struct {
	db *pgxpool.Pool
}

const (
	defaultSearchTextConfig     = "simple"
	searchTextWeight            = 0.50
	searchPopularityWeight      = 0.20
	searchConversionWeight      = 0.15
	searchRatingWeight          = 0.10
	searchTrendingWeight        = 0.05
	suggestPrefixBoost          = 2.0
	recommendedAffinityWeight   = 0.55
	recommendedCoViewWeight     = 0.25
	recommendedConversionWeight = 0.10
	recommendedTrendingWeight   = 0.10
	recentInteractionLimit      = 20
	lowStockThreshold           = 5
)

func NewDiscoveryRepository(db *pgxpool.Pool) *DiscoveryRepository {
	return &DiscoveryRepository{db: db}
}

func (r *DiscoveryRepository) GetFeedCandidates(ctx context.Context, req *discovery.FeedRequest) ([]discovery.Candidate, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	switch req.FeedType {
	case discovery.FeedTrending:
		return r.getTrendingCandidates(ctx, req.Limit)
	case discovery.FeedBestSellers:
		return r.getBestSellerCandidates(ctx, req.Limit)
	case discovery.FeedRecommended:
		return r.getRecommendedCandidates(ctx, *req.UserID, req.Limit)
	case discovery.FeedCategory:
		return r.getCategoryCandidates(ctx, *req.CategoryID, req.Limit)
	case discovery.FeedDeals:
		return r.getDealCandidates(ctx, req.Limit)
	case discovery.FeedNewArrivals:
		return r.getNewArrivalCandidates(ctx, req.Limit)
	case discovery.FeedHighlyRated:
		return r.getHighlyRatedCandidates(ctx, req.Limit)
	case discovery.FeedMostWishlisted:
		return r.getMostWishlistedCandidates(ctx, req.Limit)
	case discovery.FeedAlsoViewed:
		return r.getAlsoViewedCandidates(ctx, req.UserID, req.SessionID, req.Limit)
	case discovery.FeedFeatured:
		return r.getFeaturedCandidates(ctx, req.Limit)
	case discovery.FeedSearch:
		return r.getSearchCandidates(ctx, *req.Query, req.Limit)
	default:
		return nil, discovery.ErrFeedNotImplemented
	}
}

func (r *DiscoveryRepository) HydrateProductCards(ctx context.Context, productIDs []int64) ([]discovery.ProductCard, error) {
	if len(productIDs) == 0 {
		return []discovery.ProductCard{}, nil
	}

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
				CASE
					WHEN ad.discount_type = 'percentage' THEN ad.value::DOUBLE PRECISION
					ELSE ((ad.value / NULLIF(p.base_price, 0)) * 100)::DOUBLE PRECISION
				END AS discount_percent
			FROM requested_products rp
			JOIN products p ON p.id = rp.product_id
			JOIN discount_targets dt
			  ON dt.target_type = 'product'
			 AND dt.target_id = p.id
			JOIN active_discounts ad ON ad.id = dt.discount_id

			UNION ALL

			SELECT p.id AS product_id,
				CASE
					WHEN ad.discount_type = 'percentage' THEN ad.value::DOUBLE PRECISION
					ELSE ((ad.value / NULLIF(p.base_price, 0)) * 100)::DOUBLE PRECISION
				END AS discount_percent
			FROM requested_products rp
			JOIN products p ON p.id = rp.product_id
			JOIN discount_targets dt
			  ON dt.target_type = 'brand'
			 AND dt.target_id = p.brand_id
			JOIN active_discounts ad ON ad.id = dt.discount_id

			UNION ALL

			SELECT p.id AS product_id,
				CASE
					WHEN ad.discount_type = 'percentage' THEN ad.value::DOUBLE PRECISION
					ELSE ((ad.value / NULLIF(p.base_price, 0)) * 100)::DOUBLE PRECISION
				END AS discount_percent
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

	rows, err := r.db.Query(
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

func (r *DiscoveryRepository) getTrendingCandidates(ctx context.Context, limit int) ([]discovery.Candidate, error) {
	const q = `
		SELECT pm.product_id,
		       pm.trending_score,
		       pm.conversion_rate
		FROM product_metrics pm
		JOIN products p ON p.id = pm.product_id
		WHERE p.status = $1
		  AND pm.trending_score > 0
		ORDER BY pm.trending_score DESC, pm.conversion_rate DESC, pm.product_id DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, q, product.StatusActive, limit)
	if err != nil {
		return nil, fmt.Errorf("get trending candidates: %w", err)
	}
	defer rows.Close()

	return scanCandidatesWithSignals(rows, "trending_score", "conversion_rate")
}

func (r *DiscoveryRepository) getBestSellerCandidates(ctx context.Context, limit int) ([]discovery.Candidate, error) {
	const q = `
		SELECT pm.product_id,
		       pm.weekly_purchases::DOUBLE PRECISION AS weekly_purchases,
		       pm.conversion_rate
		FROM product_metrics pm
		JOIN products p ON p.id = pm.product_id
		WHERE p.status = $1
		  AND pm.weekly_purchases > 0
		ORDER BY pm.weekly_purchases DESC, pm.conversion_rate DESC, pm.product_id DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, q, product.StatusActive, limit)
	if err != nil {
		return nil, fmt.Errorf("get best seller candidates: %w", err)
	}
	defer rows.Close()

	return scanCandidatesWithSignals(rows, "weekly_purchases", "conversion_rate")
}

func (r *DiscoveryRepository) getRecommendedCandidates(ctx context.Context, userID int64, limit int) ([]discovery.Candidate, error) {
	const q = `
		WITH recent_user_products AS (
			SELECT product_id
			FROM (
				SELECT product_id, MAX(interacted_at) AS interacted_at
				FROM (
					SELECT pe.product_id, MAX(pe.created_at) AS interacted_at
					FROM product_events pe
					WHERE pe.user_id = $2
					GROUP BY pe.product_id

					UNION ALL

					SELECT oi.product_id, MAX(o.created_at) AS interacted_at
					FROM orders o
					JOIN order_items oi ON oi.order_id = o.id
					WHERE o.user_id = $2
					GROUP BY oi.product_id

					UNION ALL

					SELECT wi.product_id, MAX(wi.added_at) AS interacted_at
					FROM wishlists w
					JOIN wishlist_items wi ON wi.wishlist_id = w.id
					WHERE w.user_id = $2
					GROUP BY wi.product_id
				) interactions
				GROUP BY product_id
			) ranked_recent
			ORDER BY interacted_at DESC, product_id DESC
			LIMIT $3
		),
		affinity_products AS (
			SELECT pcm.product_id,
			       MAX(uca.score)::DOUBLE PRECISION AS personalization_score
			FROM user_category_affinity uca
			JOIN product_category_map pcm ON pcm.category_id = uca.category_id
			JOIN products p ON p.id = pcm.product_id
			WHERE uca.user_id = $2
			  AND p.status = $1
			GROUP BY pcm.product_id
		),
		co_view_products AS (
			SELECT pcv.related_product_id AS product_id,
			       MAX(pcv.score)::DOUBLE PRECISION AS co_view_score
			FROM product_co_views pcv
			JOIN recent_user_products rup ON rup.product_id = pcv.product_id
			JOIN products p ON p.id = pcv.related_product_id
			WHERE p.status = $1
			GROUP BY pcv.related_product_id
		)
		SELECT p.id,
		       COALESCE(ap.personalization_score, 0)::DOUBLE PRECISION AS personalization_score,
		       COALESCE(cvp.co_view_score, 0)::DOUBLE PRECISION AS co_view_score,
		       COALESCE(pm.conversion_rate, 0)::DOUBLE PRECISION AS conversion_rate,
		       COALESCE(pm.trending_score, 0)::DOUBLE PRECISION AS trending_score
		FROM products p
		LEFT JOIN affinity_products ap ON ap.product_id = p.id
		LEFT JOIN co_view_products cvp ON cvp.product_id = p.id
		LEFT JOIN product_metrics pm ON pm.product_id = p.id
		WHERE p.status = $1
		  AND (ap.product_id IS NOT NULL OR cvp.product_id IS NOT NULL)
		  AND NOT EXISTS (
		      SELECT 1
		      FROM recent_user_products rup
		      WHERE rup.product_id = p.id
		  )
		ORDER BY
			($4 * COALESCE(ap.personalization_score, 0)::DOUBLE PRECISION)
			+ ($5 * COALESCE(cvp.co_view_score, 0)::DOUBLE PRECISION)
			+ ($6 * COALESCE(pm.conversion_rate, 0)::DOUBLE PRECISION)
			+ ($7 * COALESCE(pm.trending_score, 0)::DOUBLE PRECISION) DESC,
			p.id DESC
		LIMIT $8`

	rows, err := r.db.Query(
		ctx,
		q,
		product.StatusActive,
		userID,
		recentInteractionLimit,
		recommendedAffinityWeight,
		recommendedCoViewWeight,
		recommendedConversionWeight,
		recommendedTrendingWeight,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get recommended candidates: %w", err)
	}
	defer rows.Close()

	return scanCandidatesWithSignals(rows, "personalization_score", "co_view_score", "conversion_rate", "trending_score")
}

func (r *DiscoveryRepository) getCategoryCandidates(ctx context.Context, categoryID int64, limit int) ([]discovery.Candidate, error) {
	const q = `
		SELECT p.id,
		       MAX(1.0 / (cc.depth + 1))::DOUBLE PRECISION AS category_score,
		       MIN(cc.depth)::DOUBLE PRECISION AS category_depth
		FROM products p
		JOIN product_category_map pcm ON pcm.product_id = p.id
		JOIN category_closure cc ON cc.descendant_id = pcm.category_id
		WHERE p.status = $1
		  AND cc.ancestor_id = $2
		GROUP BY p.id
		ORDER BY MIN(cc.depth), p.id DESC
		LIMIT $3`

	rows, err := r.db.Query(ctx, q, product.StatusActive, categoryID, limit)
	if err != nil {
		return nil, fmt.Errorf("get category candidates: %w", err)
	}
	defer rows.Close()

	return scanCandidatesWithSignals(rows, "category_score", "category_depth")
}

func (r *DiscoveryRepository) getDealCandidates(ctx context.Context, limit int) ([]discovery.Candidate, error) {
	const q = `
		WITH active_discounts AS (
			SELECT id, discount_type, value
			FROM discounts
			WHERE is_active = TRUE
			  AND (starts_at IS NULL OR starts_at <= NOW())
			  AND (ends_at IS NULL OR ends_at >= NOW())
		),
		target_products AS (
			SELECT p.id AS product_id,
			       ad.discount_type,
			       ad.value,
			       p.base_price
			FROM active_discounts ad
			JOIN discount_targets dt
			  ON dt.discount_id = ad.id
			 AND dt.target_type = 'product'
			JOIN products p ON p.id = dt.target_id
			WHERE p.status = $1

			UNION ALL

			SELECT p.id AS product_id,
			       ad.discount_type,
			       ad.value,
			       p.base_price
			FROM active_discounts ad
			JOIN discount_targets dt
			  ON dt.discount_id = ad.id
			 AND dt.target_type = 'category'
			JOIN product_category_map pcm ON pcm.category_id = dt.target_id
			JOIN products p ON p.id = pcm.product_id
			WHERE p.status = $1

			UNION ALL

			SELECT p.id AS product_id,
			       ad.discount_type,
			       ad.value,
			       p.base_price
			FROM active_discounts ad
			JOIN discount_targets dt
			  ON dt.discount_id = ad.id
			 AND dt.target_type = 'brand'
			JOIN products p
			  ON p.brand_id = dt.target_id
			 AND p.status = $1
		)
		SELECT product_id,
		       COALESCE(MAX(
		           CASE
		               WHEN discount_type = 'percentage' THEN value::DOUBLE PRECISION
		               ELSE ((value / NULLIF(base_price, 0)) * 100)::DOUBLE PRECISION
		           END
		       ), 0.0) AS discount_score
		FROM target_products
		GROUP BY product_id
		ORDER BY discount_score DESC, product_id DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, q, product.StatusActive, limit)
	if err != nil {
		return nil, fmt.Errorf("get deal candidates: %w", err)
	}
	defer rows.Close()

	return scanCandidatesWithSignals(rows, "discount_score")
}

func (r *DiscoveryRepository) getNewArrivalCandidates(ctx context.Context, limit int) ([]discovery.Candidate, error) {
	const q = `
		SELECT p.id,
		       EXTRACT(EPOCH FROM p.created_at)::DOUBLE PRECISION AS freshness_score
		FROM products p
		WHERE p.status = $1
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, q, product.StatusActive, limit)
	if err != nil {
		return nil, fmt.Errorf("get new arrival candidates: %w", err)
	}
	defer rows.Close()

	return scanCandidatesWithSignals(rows, "freshness_score")
}

func (r *DiscoveryRepository) getHighlyRatedCandidates(ctx context.Context, limit int) ([]discovery.Candidate, error) {
	const q = `
		SELECT p.id,
		       p.rating::DOUBLE PRECISION AS rating_score,
		       p.review_count::DOUBLE PRECISION AS review_count
		FROM products p
		WHERE p.status = $1
		  AND p.review_count > 0
		ORDER BY p.rating DESC, p.review_count DESC, p.id DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, q, product.StatusActive, limit)
	if err != nil {
		return nil, fmt.Errorf("get highly rated candidates: %w", err)
	}
	defer rows.Close()

	return scanCandidatesWithSignals(rows, "rating_score", "review_count")
}

func (r *DiscoveryRepository) getMostWishlistedCandidates(ctx context.Context, limit int) ([]discovery.Candidate, error) {
	const q = `
		SELECT p.id,
		       COUNT(*)::DOUBLE PRECISION AS wishlist_count
		FROM wishlist_items wi
		JOIN products p ON p.id = wi.product_id
		WHERE p.status = $1
		GROUP BY p.id
		ORDER BY COUNT(*) DESC, p.id DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, q, product.StatusActive, limit)
	if err != nil {
		return nil, fmt.Errorf("get most wishlisted candidates: %w", err)
	}
	defer rows.Close()

	return scanCandidatesWithSignals(rows, "wishlist_count")
}

func (r *DiscoveryRepository) getAlsoViewedCandidates(ctx context.Context, userID *int64, sessionID *string, limit int) ([]discovery.Candidate, error) {
	const qTemplate = `
		WITH recent_products AS (
			SELECT product_id
			FROM (
				SELECT pe.product_id, MAX(pe.created_at) AS interacted_at
				FROM product_events pe
				WHERE pe.event_type = 'view'
				  AND %s
				GROUP BY pe.product_id
			) ranked_recent
			ORDER BY interacted_at DESC, product_id DESC
			LIMIT $3
		),
		co_view_products AS (
			SELECT pcv.related_product_id AS product_id,
			       MAX(pcv.score)::DOUBLE PRECISION AS co_view_score
			FROM product_co_views pcv
			JOIN recent_products rp ON rp.product_id = pcv.product_id
			JOIN products p ON p.id = pcv.related_product_id
			WHERE p.status = $1
			GROUP BY pcv.related_product_id
		)
		SELECT p.id,
		       cvp.co_view_score,
		       COALESCE(pm.weekly_views, 0)::DOUBLE PRECISION AS popularity_score,
		       COALESCE(pm.conversion_rate, 0)::DOUBLE PRECISION AS conversion_rate
		FROM co_view_products cvp
		JOIN products p ON p.id = cvp.product_id
		LEFT JOIN product_metrics pm ON pm.product_id = cvp.product_id
		WHERE p.status = $1
		  AND NOT EXISTS (
		      SELECT 1
		      FROM recent_products rp
		      WHERE rp.product_id = p.id
		  )
			ORDER BY
			cvp.co_view_score DESC,
			COALESCE(pm.weekly_views, 0)::DOUBLE PRECISION DESC,
			COALESCE(pm.conversion_rate, 0)::DOUBLE PRECISION DESC,
			p.id DESC
		LIMIT $4`

	filterCondition := "pe.user_id = $2"
	filterValue := any(nil)
	if userID != nil {
		filterValue = *userID
	} else {
		filterCondition = "pe.session_id = $2"
		filterValue = *sessionID
	}

	rows, err := r.db.Query(
		ctx,
		fmt.Sprintf(qTemplate, filterCondition),
		product.StatusActive,
		filterValue,
		recentInteractionLimit,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get also viewed candidates: %w", err)
	}
	defer rows.Close()

	return scanCandidatesWithSignals(rows, "co_view_score", "popularity_score", "conversion_rate")
}

func (r *DiscoveryRepository) getFeaturedCandidates(ctx context.Context, limit int) ([]discovery.Candidate, error) {
	const q = `
		SELECT p.id,
		       1.0::DOUBLE PRECISION AS merchandising_score,
		       EXTRACT(EPOCH FROM p.created_at)::DOUBLE PRECISION AS freshness_score
		FROM products p
		WHERE p.status = $1
		  AND p.is_featured = TRUE
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, q, product.StatusActive, limit)
	if err != nil {
		return nil, fmt.Errorf("get featured candidates: %w", err)
	}
	defer rows.Close()

	return scanCandidatesWithSignals(rows, "merchandising_score", "freshness_score")
}

func (r *DiscoveryRepository) getSearchCandidates(ctx context.Context, query string, limit int) ([]discovery.Candidate, error) {
	const q = `
		WITH search_input AS (
			SELECT LOWER($1) AS normalized_query,
			       websearch_to_tsquery($3::regconfig, LOWER($1)) AS ts_query
		),
		fts_candidates AS (
			SELECT psd.product_id,
			       ts_rank_cd(psd.search_vector, si.ts_query)::DOUBLE PRECISION AS text_relevance
			FROM product_search_documents psd
			CROSS JOIN search_input si
			JOIN products p ON p.id = psd.product_id
			WHERE p.status = $2
			  AND psd.search_vector @@ si.ts_query
		),
		trigram_candidates AS (
			SELECT psd.product_id,
			       GREATEST(
			           similarity(LOWER(psd.search_document), si.normalized_query),
			           similarity(LOWER(p.name), si.normalized_query)
			       )::DOUBLE PRECISION AS text_relevance
			FROM product_search_documents psd
			CROSS JOIN search_input si
			JOIN products p ON p.id = psd.product_id
			WHERE p.status = $2
			  AND (
			      LOWER(p.name) LIKE si.normalized_query || '%'
			      OR LOWER(psd.search_document) LIKE si.normalized_query || '%'
			      OR LOWER(psd.search_document) % si.normalized_query
			  )
		),
		ranked AS (
			SELECT c.product_id,
			       MAX(c.text_relevance) AS text_relevance
			FROM (
				SELECT * FROM fts_candidates
				UNION ALL
				SELECT * FROM trigram_candidates
			) c
			GROUP BY c.product_id
		)
		SELECT r.product_id,
		       r.text_relevance,
		       COALESCE(pm.weekly_purchases, 0)::DOUBLE PRECISION AS popularity_score,
		       COALESCE(pm.conversion_rate, 0)::DOUBLE PRECISION AS conversion_rate,
		       p.rating::DOUBLE PRECISION AS rating_score,
		       COALESCE(pm.trending_score, 0)::DOUBLE PRECISION AS trending_score
		FROM ranked r
		JOIN products p ON p.id = r.product_id
		LEFT JOIN product_metrics pm ON pm.product_id = r.product_id
		ORDER BY
			($4 * r.text_relevance)
			+ ($5 * COALESCE(pm.weekly_purchases, 0)::DOUBLE PRECISION)
			+ ($6 * COALESCE(pm.conversion_rate, 0)::DOUBLE PRECISION)
			+ ($7 * p.rating::DOUBLE PRECISION)
			+ ($8 * COALESCE(pm.trending_score, 0)::DOUBLE PRECISION) DESC,
			r.product_id DESC
		LIMIT $9`

	rows, err := r.db.Query(
		ctx,
		q,
		query,
		product.StatusActive,
		defaultSearchTextConfig,
		searchTextWeight,
		searchPopularityWeight,
		searchConversionWeight,
		searchRatingWeight,
		searchTrendingWeight,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get search candidates: %w", err)
	}
	defer rows.Close()

	return scanCandidatesWithSignals(rows, "text_relevance", "popularity_score", "conversion_rate", "rating_score", "trending_score")
}

func (r *DiscoveryRepository) Suggest(ctx context.Context, req *discovery.SuggestRequest) ([]discovery.Suggestion, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	const q = `
		WITH input AS (
			SELECT LOWER($1) AS prefix
		),
		product_matches AS (
			SELECT p.name AS suggestion_text,
			       'product'::TEXT AS suggestion_type,
			       p.id AS reference_id,
			       (
			           CASE WHEN LOWER(p.name) LIKE i.prefix || '%' THEN $2 ELSE 0.0 END
			           + similarity(LOWER(p.name), i.prefix)
			           + (COALESCE(pm.weekly_views, 0)::DOUBLE PRECISION / 1000.0)
			       ) AS popularity_score
			FROM products p
			CROSS JOIN input i
			LEFT JOIN product_metrics pm ON pm.product_id = p.id
			WHERE p.status = $3
			  AND (
			      LOWER(p.name) LIKE i.prefix || '%'
			      OR LOWER(p.name) % i.prefix
			  )
			ORDER BY popularity_score DESC, p.id DESC
			LIMIT $4
		),
		category_matches AS (
			SELECT c.name AS suggestion_text,
			       'category'::TEXT AS suggestion_type,
			       c.id AS reference_id,
			       (
			           CASE WHEN LOWER(c.name) LIKE i.prefix || '%' THEN $2 ELSE 0.0 END
			           + similarity(LOWER(c.name), i.prefix)
			       ) AS popularity_score
			FROM product_categories c
			CROSS JOIN input i
			WHERE c.is_active = TRUE
			  AND (
			      LOWER(c.name) LIKE i.prefix || '%'
			      OR LOWER(c.name) % i.prefix
			  )
			ORDER BY popularity_score DESC, c.id DESC
			LIMIT $4
		),
		brand_matches AS (
			SELECT b.name AS suggestion_text,
			       'brand'::TEXT AS suggestion_type,
			       b.id AS reference_id,
			       (
			           CASE WHEN LOWER(b.name) LIKE i.prefix || '%' THEN $2 ELSE 0.0 END
			           + similarity(LOWER(b.name), i.prefix)
			       ) AS popularity_score
			FROM product_brands b
			CROSS JOIN input i
			WHERE b.is_active = TRUE
			  AND (
			      LOWER(b.name) LIKE i.prefix || '%'
			      OR LOWER(b.name) % i.prefix
			  )
			ORDER BY popularity_score DESC, b.id DESC
			LIMIT $4
		),
		query_matches AS (
			SELECT se.normalized_query AS suggestion_text,
			       'query'::TEXT AS suggestion_type,
			       NULL::BIGINT AS reference_id,
			       (
			           COUNT(*)::DOUBLE PRECISION
			           + CASE WHEN se.normalized_query LIKE i.prefix || '%' THEN $2 ELSE 0.0 END
			       ) AS popularity_score
			FROM search_events se
			CROSS JOIN input i
			WHERE se.normalized_query LIKE i.prefix || '%'
			   OR se.normalized_query % i.prefix
			GROUP BY se.normalized_query, i.prefix
			ORDER BY popularity_score DESC, se.normalized_query ASC
			LIMIT $4
		)
		SELECT suggestion_text, suggestion_type, reference_id, popularity_score
		FROM (
			SELECT * FROM product_matches
			UNION ALL
			SELECT * FROM category_matches
			UNION ALL
			SELECT * FROM brand_matches
			UNION ALL
			SELECT * FROM query_matches
		) suggestions
		ORDER BY popularity_score DESC, suggestion_text ASC
		LIMIT $4`

	rows, err := r.db.Query(ctx, q, req.Prefix, suggestPrefixBoost, product.StatusActive, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("suggest discovery terms: %w", err)
	}
	defer rows.Close()

	var suggestions []discovery.Suggestion
	for rows.Next() {
		var suggestion discovery.Suggestion
		var referenceID sql.NullInt64
		if err := rows.Scan(&suggestion.Text, &suggestion.Type, &referenceID, &suggestion.PopularityScore); err != nil {
			return nil, fmt.Errorf("scan discovery suggestion: %w", err)
		}
		if referenceID.Valid {
			id := referenceID.Int64
			suggestion.ReferenceID = &id
		}
		suggestions = append(suggestions, suggestion)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate discovery suggestions: %w", err)
	}
	return suggestions, nil
}

func (r *DiscoveryRepository) TrackSearch(ctx context.Context, event *discovery.SearchEvent) (int64, error) {
	if err := event.Validate(); err != nil {
		return 0, err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin search tracking transaction: %w", err)
	}

	var eventID int64
	if err := withTx(ctx, tx, func() error {
		const insertEvent = `
			INSERT INTO search_events (query, normalized_query, user_id, session_id, result_count)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id`

		if err := tx.QueryRow(
			ctx,
			insertEvent,
			event.Query,
			event.NormalizedQuery,
			event.UserID,
			event.SessionID,
			event.ResultCount,
		).Scan(&eventID); err != nil {
			return fmt.Errorf("insert search event: %w", err)
		}

		if len(event.Results) == 0 {
			return nil
		}

		const insertPosition = `
			INSERT INTO search_result_positions (search_event_id, product_id, position, score)
			VALUES ($1, $2, $3, $4)`

		var batch pgx.Batch
		for _, result := range event.Results {
			batch.Queue(insertPosition, eventID, result.ProductID, result.Position, result.Score)
		}

		results := tx.SendBatch(ctx, &batch)
		defer results.Close()

		for range event.Results {
			if _, err := results.Exec(); err != nil {
				return fmt.Errorf("insert search result position: %w", err)
			}
		}

		return nil
	}); err != nil {
		return 0, err
	}

	return eventID, nil
}

func (r *DiscoveryRepository) TrackSearchClick(ctx context.Context, event *discovery.SearchClickEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}

	const q = `
		INSERT INTO search_clicks (search_event_id, product_id, position, user_id, session_id)
		VALUES ($1, $2, $3, $4, $5)`

	if _, err := r.db.Exec(ctx, q, event.SearchEventID, event.ProductID, event.Position, event.UserID, event.SessionID); err != nil {
		return fmt.Errorf("insert search click: %w", err)
	}
	return nil
}

func scanCandidatesWithSignals(rows pgx.Rows, signalNames ...string) ([]discovery.Candidate, error) {
	var out []discovery.Candidate
	for rows.Next() {
		candidate := discovery.Candidate{
			Signals: make(map[string]float64, len(signalNames)),
		}

		values := make([]float64, len(signalNames))
		scanArgs := make([]any, 1+len(signalNames))
		scanArgs[0] = &candidate.ProductID
		for i := range values {
			scanArgs[i+1] = &values[i]
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("scan discovery candidate: %w", err)
		}
		for i, signalName := range signalNames {
			candidate.Signals[signalName] = values[i]
		}

		out = append(out, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate discovery candidates: %w", err)
	}
	return out, nil
}
