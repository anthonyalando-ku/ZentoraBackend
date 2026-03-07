package postgres

import (
	"context"
	"fmt"

	"zentora-service/internal/domain/discovery"
	"zentora-service/internal/domain/product"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DiscoveryRepository struct {
	db *pgxpool.Pool
}

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
	case discovery.FeedFeatured:
		return r.getFeaturedCandidates(ctx, req.Limit)
	default:
		return nil, discovery.ErrFeedNotImplemented
	}
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
