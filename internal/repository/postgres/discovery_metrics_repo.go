package postgres

import (
	"context"
	"fmt"
)

const (
	productMetricsDailyViewWeight      = 0.4
	productMetricsWeeklyViewWeight     = 0.2
	productMetricsWeeklyPurchaseWeight = 1.2
	productMetricsConversionScale      = 100.0

	categoryAffinityViewWeight      = 1.0
	categoryAffinityAddToCartWeight = 3.0
	categoryAffinityPurchaseWeight  = 5.0
	categoryAffinityWishlistWeight  = 2.0
)

func (r *DiscoveryRepository) RefreshProductMetrics(ctx context.Context) error {
	q := fmt.Sprintf(`
		WITH view_metrics AS (
			SELECT pe.product_id,
			       COUNT(*) FILTER (WHERE pe.event_type = 'view' AND pe.created_at >= NOW() - INTERVAL '1 day') AS daily_views,
			       COUNT(*) FILTER (WHERE pe.event_type = 'view' AND pe.created_at >= NOW() - INTERVAL '7 days') AS weekly_views
			FROM product_events pe
			WHERE pe.created_at >= NOW() - INTERVAL '7 days'
			GROUP BY pe.product_id
		),
		purchase_metrics AS (
			SELECT oi.product_id,
			       COALESCE(SUM(oi.quantity), 0) AS weekly_purchases
			FROM order_items oi
			JOIN orders o ON o.id = oi.order_id
			WHERE o.created_at >= NOW() - INTERVAL '7 days'
			GROUP BY oi.product_id
		),
		combined AS (
			SELECT p.id AS product_id,
			       COALESCE(vm.daily_views, 0) AS daily_views,
			       COALESCE(vm.weekly_views, 0) AS weekly_views,
			       COALESCE(pm.weekly_purchases, 0) AS weekly_purchases
			FROM products p
			LEFT JOIN view_metrics vm ON vm.product_id = p.id
			LEFT JOIN purchase_metrics pm ON pm.product_id = p.id
		)
		INSERT INTO product_metrics (
			product_id,
			trending_score,
			daily_views,
			weekly_views,
			weekly_purchases,
			conversion_rate,
			updated_at
		)
		SELECT c.product_id,
		       (c.daily_views * %f)
		       + (c.weekly_views * %f)
		       + (c.weekly_purchases * %f)
		       + (
		           CASE
		               WHEN c.weekly_views > 0 THEN (c.weekly_purchases::DOUBLE PRECISION / c.weekly_views)
		               ELSE 0
		           END * %f
		       ) AS trending_score,
		       c.daily_views,
		       c.weekly_views,
		       c.weekly_purchases,
		       CASE
		           WHEN c.weekly_views > 0 THEN c.weekly_purchases::DOUBLE PRECISION / c.weekly_views
		           ELSE 0
		       END AS conversion_rate,
		       NOW()
		FROM combined c
		ON CONFLICT (product_id) DO UPDATE
		SET trending_score = EXCLUDED.trending_score,
		    daily_views = EXCLUDED.daily_views,
		    weekly_views = EXCLUDED.weekly_views,
		    weekly_purchases = EXCLUDED.weekly_purchases,
		    conversion_rate = EXCLUDED.conversion_rate,
		    updated_at = EXCLUDED.updated_at`,
		productMetricsDailyViewWeight,
		productMetricsWeeklyViewWeight,
		productMetricsWeeklyPurchaseWeight,
		productMetricsConversionScale,
	)

	if _, err := r.db.Exec(ctx, q); err != nil {
		return fmt.Errorf("refresh product metrics: %w", err)
	}
	return nil
}

func (r *DiscoveryRepository) RefreshUserCategoryAffinity(ctx context.Context) error {
	q := fmt.Sprintf(`
		WITH interaction_scores AS (
			SELECT pe.user_id,
			       pcm.category_id,
			       SUM(
			           CASE pe.event_type
			               WHEN 'view' THEN %f
			               WHEN 'add_to_cart' THEN %f
			               WHEN 'purchase' THEN %f * GREATEST(pe.quantity, 1)
			               ELSE 0.0
			           END
			       ) AS score
			FROM product_events pe
			JOIN product_category_map pcm ON pcm.product_id = pe.product_id
			WHERE pe.user_id IS NOT NULL
			  AND pe.created_at >= NOW() - INTERVAL '30 days'
			GROUP BY pe.user_id, pcm.category_id

			UNION ALL

			SELECT o.user_id,
			       pcm.category_id,
			       SUM(%f * GREATEST(oi.quantity, 1)) AS score
			FROM orders o
			JOIN order_items oi ON oi.order_id = o.id
			JOIN product_category_map pcm ON pcm.product_id = oi.product_id
			WHERE o.user_id IS NOT NULL
			  AND o.created_at >= NOW() - INTERVAL '30 days'
			GROUP BY o.user_id, pcm.category_id

			UNION ALL

			SELECT w.user_id,
			       pcm.category_id,
			       COUNT(*)::DOUBLE PRECISION * %f AS score
			FROM wishlists w
			JOIN wishlist_items wi ON wi.wishlist_id = w.id
			JOIN product_category_map pcm ON pcm.product_id = wi.product_id
			WHERE wi.added_at >= NOW() - INTERVAL '30 days'
			GROUP BY w.user_id, pcm.category_id
		),
		aggregated AS (
			SELECT user_id, category_id, SUM(score) AS score
			FROM interaction_scores
			GROUP BY user_id, category_id
		)
		INSERT INTO user_category_affinity (user_id, category_id, score)
		SELECT user_id, category_id, score
		FROM aggregated
		ON CONFLICT (user_id, category_id) DO UPDATE
		SET score = EXCLUDED.score`,
		categoryAffinityViewWeight,
		categoryAffinityAddToCartWeight,
		categoryAffinityPurchaseWeight,
		categoryAffinityPurchaseWeight,
		categoryAffinityWishlistWeight,
	)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin user category affinity refresh: %w", err)
	}

	return withTx(ctx, tx, func() error {
		if _, err := tx.Exec(ctx, "TRUNCATE TABLE user_category_affinity"); err != nil {
			return fmt.Errorf("truncate user category affinity: %w", err)
		}
		if _, err := tx.Exec(ctx, q); err != nil {
			return fmt.Errorf("refresh user category affinity: %w", err)
		}
		return nil
	})
}

func (r *DiscoveryRepository) RefreshProductCoViews(ctx context.Context) error {
	const q = `
		WITH recent_views AS (
			SELECT DISTINCT
			       COALESCE('u:' || pe.user_id::TEXT, 's:' || pe.session_id) AS actor_id,
			       pe.product_id
			FROM product_events pe
			WHERE pe.event_type = 'view'
			  AND pe.created_at >= NOW() - INTERVAL '30 days'
			  AND (pe.user_id IS NOT NULL OR pe.session_id IS NOT NULL)
		),
		pairs AS (
			SELECT rv1.product_id,
			       rv2.product_id AS related_product_id,
			       COUNT(*)::DOUBLE PRECISION AS score
			FROM recent_views rv1
			JOIN recent_views rv2
			  ON rv1.actor_id = rv2.actor_id
			 AND rv1.product_id <> rv2.product_id
			GROUP BY rv1.product_id, rv2.product_id
		)
		INSERT INTO product_co_views (product_id, related_product_id, score)
		SELECT product_id, related_product_id, score
		FROM pairs
		ON CONFLICT (product_id, related_product_id) DO UPDATE
		SET score = EXCLUDED.score`

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin product co-views refresh: %w", err)
	}

	return withTx(ctx, tx, func() error {
		if _, err := tx.Exec(ctx, "TRUNCATE TABLE product_co_views"); err != nil {
			return fmt.Errorf("truncate product co-views: %w", err)
		}
		if _, err := tx.Exec(ctx, q); err != nil {
			return fmt.Errorf("refresh product co-views: %w", err)
		}
		return nil
	})
}
