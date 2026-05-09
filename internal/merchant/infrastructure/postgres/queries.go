package postgres

// ---------------------------------------------------------------------------
// queryLoadFeedBatch
//
// The primary paginated feed query.  Fetches one variant per row with all
// merchant-relevant data resolved in a single round-trip.
//
// Parameters:
//   $1  BIGINT   -- keyset cursor (last seen variant_id, 0 = start)
//   $2  INT      -- batch size / LIMIT
//   $3  BOOLEAN  -- active_only (true = products.status = 'active')
//   $4  BOOLEAN  -- in_stock_only
//   $5  BOOLEAN  -- exclude_digital
//   $6  BIGINT[] -- category_ids filter (empty = no restriction)
//   $7  BIGINT[] -- brand_ids filter (empty = no restriction)
//   $8  TIMESTAMP -- updated_after for incremental mode (NULL = full)
//
// Design notes:
//   * eligible_variants CTE gates all eligibility rules once.
//   * best_discounts CTE reuses the three-way discount resolution from the
//     discovery engine (product / brand / category targets).
//   * inventory_by_variant aggregates across all locations.
//   * variant_attrs pivots the EAV attribute system into named columns.
//   * images CTE ranks primary + up to 10 additional images per product.
//   * category_path builds the full breadcrumb path for product_type.
//   * Results ordered by pv.id ASC for stable keyset pagination.
// ---------------------------------------------------------------------------

const queryLoadFeedBatch = `
WITH

-- =========================================================================
-- 1. Eligible variants gate
--    Apply all merchant eligibility rules ONCE here.
-- =========================================================================
eligible_variants AS (
    SELECT
        pv.id          AS variant_id,
        pv.product_id,
        pv.sku,
        pv.price       AS variant_price,
        pv.weight      AS weight_kg,
        p.name         AS product_name,
        p.slug         AS product_slug,
        p.description,
        p.short_description,
        p.base_price,
        p.is_digital,
        p.updated_at,
        COALESCE(pb.name, '')  AS brand_name,
        COALESCE(pb.slug, '')  AS brand_slug,
        o.currency
    FROM product_variants pv
    JOIN products p
      ON p.id = pv.product_id
    LEFT JOIN product_brands pb
      ON pb.id = p.brand_id
    CROSS JOIN (SELECT 'KES'::TEXT AS currency) o
    WHERE pv.id > $1                           -- keyset cursor
      AND pv.is_active = TRUE
      AND ($3 = FALSE OR p.status = 'active')  -- active_only
      AND p.status != 'archived'
      AND ($5 = FALSE OR p.is_digital = FALSE) -- exclude_digital
      -- brand filter
      AND (
          COALESCE(array_length($7::BIGINT[], 1), 0) = 0
          OR p.brand_id = ANY($7::BIGINT[])
      )
      -- category filter (via closure so parent-category filter includes children)
      AND (
          COALESCE(array_length($6::BIGINT[], 1), 0) = 0
          OR EXISTS (
              SELECT 1
              FROM product_category_map pcm
              JOIN category_closure cc ON cc.descendant_id = pcm.category_id
              WHERE pcm.product_id = p.id
                AND cc.ancestor_id = ANY($6::BIGINT[])
          )
      )
      -- incremental mode
      AND ($8::TIMESTAMP IS NULL OR p.updated_at > $8::TIMESTAMP OR pv.created_at > $8::TIMESTAMP)
    ORDER BY pv.id ASC
    LIMIT $2
),

-- =========================================================================
-- 2. Active discounts (reused from discovery engine pattern)
-- =========================================================================
active_discounts AS (
    SELECT
        d.id,
        d.discount_type,
        d.value,
        d.starts_at,
        d.ends_at
    FROM discounts d
    WHERE d.is_active = TRUE
      AND (d.starts_at IS NULL OR d.starts_at <= NOW())
      AND (d.ends_at   IS NULL OR d.ends_at   >= NOW())
),

-- Three-way discount resolution: product / brand / category
-- We keep starts_at / ends_at for sale_price_effective_date.
-- All three arms use identical column names so UNION ALL produces a
-- consistent result set (PostgreSQL names columns from the first arm).
discount_candidates AS (
    -- Product-level discounts
    SELECT
        ev.variant_id,
        ev.variant_price,
        ad.discount_type,
        ad.value        AS discount_value,
        ad.starts_at    AS discount_starts_at,
        ad.ends_at      AS discount_ends_at
    FROM eligible_variants ev
    JOIN discount_targets dt
      ON dt.target_type = 'product'
     AND dt.target_id   = ev.product_id
    JOIN active_discounts ad ON ad.id = dt.discount_id

    UNION ALL

    -- Brand-level discounts
    SELECT
        ev.variant_id,
        ev.variant_price,
        ad.discount_type,
        ad.value        AS discount_value,
        ad.starts_at    AS discount_starts_at,
        ad.ends_at      AS discount_ends_at
    FROM eligible_variants ev
    JOIN products p ON p.id = ev.product_id
    JOIN discount_targets dt
      ON dt.target_type = 'brand'
     AND dt.target_id   = p.brand_id
    JOIN active_discounts ad ON ad.id = dt.discount_id

    UNION ALL

    -- Category-level discounts
    SELECT
        ev.variant_id,
        ev.variant_price,
        ad.discount_type,
        ad.value        AS discount_value,
        ad.starts_at    AS discount_starts_at,
        ad.ends_at      AS discount_ends_at
    FROM eligible_variants ev
    JOIN product_category_map pcm ON pcm.product_id = ev.product_id
    JOIN discount_targets dt
      ON dt.target_type = 'category'
     AND dt.target_id   = pcm.category_id
    JOIN active_discounts ad ON ad.id = dt.discount_id
),

-- Pick the best (highest %) discount per variant and retain its dates.
-- ORDER BY inside DISTINCT ON must reference the aliased column names
-- that exist in discount_candidates, not the original table columns.
best_discounts AS (
    SELECT DISTINCT ON (variant_id)
        variant_id,
        discount_type,
        CASE
            WHEN discount_type = 'percentage'
                THEN discount_value
            ELSE
                ROUND(
                    ((discount_value / NULLIF(variant_price, 0)) * 100)::NUMERIC,
                    4
                )::DOUBLE PRECISION
        END                  AS discount_percent,
        discount_value       AS discount_raw_value,
        discount_starts_at,
        discount_ends_at
    FROM discount_candidates
    ORDER BY
        variant_id,
        CASE
            WHEN discount_type = 'percentage'
                THEN discount_value
            ELSE
                ROUND(
                    ((discount_value / NULLIF(variant_price, 0)) * 100)::NUMERIC,
                    4
                )::DOUBLE PRECISION
        END DESC
),

-- =========================================================================
-- 3. Inventory aggregation across all warehouse locations
-- =========================================================================
inventory_by_variant AS (
    SELECT
        ii.variant_id,
        COALESCE(SUM(ii.available_qty), 0) AS available_qty,
        COALESCE(SUM(ii.reserved_qty),  0) AS reserved_qty,
        COALESCE(SUM(ii.incoming_qty),  0) AS incoming_qty
    FROM inventory_items ii
    JOIN eligible_variants ev ON ev.variant_id = ii.variant_id
    GROUP BY ii.variant_id
),

-- =========================================================================
-- 4. In-stock filter (applied after aggregation)
-- =========================================================================
stock_filtered AS (
    SELECT ev.*
    FROM eligible_variants ev
    LEFT JOIN inventory_by_variant ibv ON ibv.variant_id = ev.variant_id
    WHERE $4 = FALSE
       OR COALESCE(ibv.available_qty, 0) - COALESCE(ibv.reserved_qty, 0) > 0
),

-- =========================================================================
-- 5. Variant attribute pivot
--    Extracts the six merchant-relevant dimensions from the EAV model.
--    Uses conditional aggregation — no sub-selects or joins per attribute.
-- =========================================================================
variant_attrs AS (
    SELECT
        vav.variant_id,
        MAX(CASE WHEN LOWER(a.slug) IN ('color', 'colour') THEN av.value END) AS color,
        MAX(CASE WHEN LOWER(a.slug) = 'size'               THEN av.value END) AS size,
        MAX(CASE WHEN LOWER(a.slug) = 'gender'             THEN av.value END) AS gender,
        MAX(CASE WHEN LOWER(a.slug) IN ('age_group','age-group') THEN av.value END) AS age_group,
        MAX(CASE WHEN LOWER(a.slug) = 'material'           THEN av.value END) AS material,
        MAX(CASE WHEN LOWER(a.slug) = 'pattern'            THEN av.value END) AS pattern,
        MAX(CASE WHEN LOWER(a.slug) IN ('gtin','barcode','ean') THEN av.value END) AS gtin,
        MAX(CASE WHEN LOWER(a.slug) IN ('mpn','manufacturer_part_number') THEN av.value END) AS mpn
    FROM variant_attribute_values vav
    JOIN attribute_values av ON av.id = vav.attribute_value_id
    JOIN attributes a        ON a.id  = av.attribute_id
    JOIN stock_filtered sf   ON sf.variant_id = vav.variant_id
    GROUP BY vav.variant_id
),

-- =========================================================================
-- 6. Image hydration
--    Primary image + up to 10 additional, ordered by priority.
-- =========================================================================
ranked_images AS (
    SELECT
        pi.product_id,
        pi.image_url,
        pi.is_primary,
        ROW_NUMBER() OVER (
            PARTITION BY pi.product_id
            ORDER BY pi.is_primary DESC, pi.sort_order ASC, pi.id ASC
        ) AS rn
    FROM product_images pi
    JOIN stock_filtered sf ON sf.product_id = pi.product_id
),
primary_image AS (
    SELECT product_id, image_url
    FROM ranked_images
    WHERE rn = 1
),
additional_images AS (
    SELECT
        product_id,
        ARRAY_AGG(image_url ORDER BY rn ASC) FILTER (WHERE rn BETWEEN 2 AND 11) AS image_urls
    FROM ranked_images
    GROUP BY product_id
),

-- =========================================================================
-- 7. Category breadcrumb path (for product_type field)
--    Walks the closure table to build "Electronics > Phones > Smartphones"
-- =========================================================================
category_primary AS (
    SELECT DISTINCT ON (pcm.product_id)
        pcm.product_id,
        pcm.category_id,
        pc.name AS category_name
    FROM product_category_map pcm
    JOIN product_categories pc ON pc.id = pcm.category_id
    JOIN stock_filtered sf ON sf.product_id = pcm.product_id
    WHERE pc.is_active = TRUE
    ORDER BY pcm.product_id, pc.id ASC
),
category_path_cte AS (
    SELECT
        cp.product_id,
        cp.category_id,
        STRING_AGG(
            anc.name,
            ' > '
            ORDER BY cc.depth DESC
        ) AS path
    FROM category_primary cp
    JOIN category_closure cc ON cc.descendant_id = cp.category_id
    JOIN product_categories anc ON anc.id = cc.ancestor_id
    WHERE anc.is_active = TRUE
    GROUP BY cp.product_id, cp.category_id
)

-- =========================================================================
-- 8. Final SELECT — one row per variant
-- =========================================================================
SELECT
    sf.variant_id,
    sf.product_id,
    sf.product_slug,
    sf.sku,
    COALESCE(va.gtin, '')      AS gtin,
    COALESCE(va.mpn,  '')      AS mpn,

    -- Title: append variant dimension for clarity (GMC best practice)
    TRIM(
        sf.product_name
        || CASE
               WHEN COALESCE(va.color, '') != '' OR COALESCE(va.size, '') != ''
               THEN ' - '
                    || COALESCE(NULLIF(va.color, ''), '')
                    || CASE WHEN va.color != '' AND va.size != '' THEN ' / ' ELSE '' END
                    || COALESCE(NULLIF(va.size,  ''), '')
               ELSE ''
           END
    )                          AS title,

    COALESCE(sf.description, sf.short_description, sf.product_name) AS description,
    sf.brand_name,

    -- Condition defaults to 'new'; can be overridden via product attributes
    'new'                      AS condition,
    sf.is_digital,

    COALESCE(cp.path, COALESCE(cat.category_name, ''))  AS category_path,
    ''                         AS google_product_category,   -- populated by HydrationService from map

    sf.base_price,
    sf.variant_price,
    sf.currency,

    COALESCE(bd.discount_percent,   0)::DOUBLE PRECISION AS discount_percent,
    COALESCE(bd.discount_type,      '')                  AS discount_type,
    bd.discount_starts_at,
    bd.discount_ends_at,

    COALESCE(ibv.available_qty, 0)  AS available_qty,
    COALESCE(ibv.reserved_qty,  0)  AS reserved_qty,
    COALESCE(ibv.incoming_qty,  0)  AS incoming_qty,

    COALESCE(img.image_url, '')     AS primary_image_url,
    COALESCE(ai.image_urls, '{}')   AS additional_image_urls,

    COALESCE(sf.weight_kg, 0)    AS weight_kg,

    COALESCE(va.color,     '')  AS attr_color,
    COALESCE(va.size,      '')  AS attr_size,
    COALESCE(va.gender,    '')  AS attr_gender,
    COALESCE(va.age_group, '')  AS attr_age_group,
    COALESCE(va.material,  '')  AS attr_material,
    COALESCE(va.pattern,   '')  AS attr_pattern,

    sf.updated_at

FROM stock_filtered sf
LEFT JOIN best_discounts     bd  ON bd.variant_id  = sf.variant_id
LEFT JOIN inventory_by_variant ibv ON ibv.variant_id = sf.variant_id
LEFT JOIN variant_attrs       va  ON va.variant_id  = sf.variant_id
LEFT JOIN primary_image       img ON img.product_id = sf.product_id
LEFT JOIN additional_images   ai  ON ai.product_id  = sf.product_id
LEFT JOIN category_path_cte   cp  ON cp.product_id  = sf.product_id
LEFT JOIN category_primary    cat ON cat.product_id = sf.product_id

ORDER BY sf.variant_id ASC
`

// ---------------------------------------------------------------------------
// queryLoadSingleProduct
//   Same as the batch query but scoped to one product_id.
//   $1 = product_id
// ---------------------------------------------------------------------------

const queryLoadSingleProduct = `
WITH base AS (
    SELECT pv.id AS variant_id
    FROM product_variants pv
    WHERE pv.product_id = $1
      AND pv.is_active  = TRUE
    ORDER BY pv.id ASC
)
` + feedBatchFromVariantIDs

// ---------------------------------------------------------------------------
// queryLoadSingleVariant
//   $1 = variant_id
// ---------------------------------------------------------------------------

const queryLoadSingleVariant = `
WITH base AS (
    SELECT $1::BIGINT AS variant_id
)
` + feedBatchFromVariantIDs

// feedBatchFromVariantIDs is a SQL fragment reused by both single-product
// and single-variant queries.  Assumes a CTE named "base (variant_id)".
const feedBatchFromVariantIDs = `
,
eligible_variants AS (
    SELECT
        pv.id          AS variant_id,
        pv.product_id,
        pv.sku,
        pv.price       AS variant_price,
        pv.weight      AS weight_kg,
        p.name         AS product_name,
        p.slug         AS product_slug,
        p.description,
        p.short_description,
        p.base_price,
        p.is_digital,
        p.updated_at,
        COALESCE(pb.name, '') AS brand_name,
        COALESCE(pb.slug, '') AS brand_slug,
        'KES'::TEXT           AS currency
    FROM base b
    JOIN product_variants pv ON pv.id = b.variant_id
    JOIN products p          ON p.id  = pv.product_id
    LEFT JOIN product_brands pb ON pb.id = p.brand_id
    WHERE pv.is_active = TRUE
      AND p.status = 'active'
),
active_discounts AS (
    SELECT id, discount_type, value, starts_at, ends_at
    FROM discounts
    WHERE is_active = TRUE
      AND (starts_at IS NULL OR starts_at <= NOW())
      AND (ends_at   IS NULL OR ends_at   >= NOW())
),
discount_candidates AS (
    SELECT ev.variant_id, ev.variant_price, ad.discount_type,
           ad.value        AS discount_value,
           ad.starts_at    AS discount_starts_at,
           ad.ends_at      AS discount_ends_at
    FROM eligible_variants ev
    JOIN discount_targets dt ON dt.target_type = 'product' AND dt.target_id = ev.product_id
    JOIN active_discounts ad ON ad.id = dt.discount_id
    UNION ALL
    SELECT ev.variant_id, ev.variant_price, ad.discount_type,
           ad.value        AS discount_value,
           ad.starts_at    AS discount_starts_at,
           ad.ends_at      AS discount_ends_at
    FROM eligible_variants ev
    JOIN products p ON p.id = ev.product_id
    JOIN discount_targets dt ON dt.target_type = 'brand' AND dt.target_id = p.brand_id
    JOIN active_discounts ad ON ad.id = dt.discount_id
    UNION ALL
    SELECT ev.variant_id, ev.variant_price, ad.discount_type,
           ad.value        AS discount_value,
           ad.starts_at    AS discount_starts_at,
           ad.ends_at      AS discount_ends_at
    FROM eligible_variants ev
    JOIN product_category_map pcm ON pcm.product_id = ev.product_id
    JOIN discount_targets dt ON dt.target_type = 'category' AND dt.target_id = pcm.category_id
    JOIN active_discounts ad ON ad.id = dt.discount_id
),
best_discounts AS (
    SELECT DISTINCT ON (variant_id)
        variant_id,
        discount_type,
        CASE WHEN discount_type = 'percentage'
             THEN discount_value
             ELSE ROUND(((discount_value / NULLIF(variant_price,0))*100)::NUMERIC,4)::DOUBLE PRECISION
        END                  AS discount_percent,
        discount_value       AS discount_raw_value,
        discount_starts_at,
        discount_ends_at
    FROM discount_candidates
    ORDER BY variant_id,
        CASE WHEN discount_type = 'percentage'
             THEN discount_value
             ELSE ROUND(((discount_value / NULLIF(variant_price,0))*100)::NUMERIC,4)::DOUBLE PRECISION
        END DESC
),
inventory_by_variant AS (
    SELECT variant_id,
        COALESCE(SUM(available_qty),0) AS available_qty,
        COALESCE(SUM(reserved_qty), 0) AS reserved_qty,
        COALESCE(SUM(incoming_qty), 0) AS incoming_qty
    FROM inventory_items
    WHERE variant_id IN (SELECT variant_id FROM eligible_variants)
    GROUP BY variant_id
),
variant_attrs AS (
    SELECT vav.variant_id,
        MAX(CASE WHEN LOWER(a.slug) IN ('color','colour') THEN av.value END) AS color,
        MAX(CASE WHEN LOWER(a.slug) = 'size'              THEN av.value END) AS size,
        MAX(CASE WHEN LOWER(a.slug) = 'gender'            THEN av.value END) AS gender,
        MAX(CASE WHEN LOWER(a.slug) IN ('age_group','age-group') THEN av.value END) AS age_group,
        MAX(CASE WHEN LOWER(a.slug) = 'material'          THEN av.value END) AS material,
        MAX(CASE WHEN LOWER(a.slug) = 'pattern'           THEN av.value END) AS pattern,
        MAX(CASE WHEN LOWER(a.slug) IN ('gtin','barcode','ean') THEN av.value END) AS gtin,
        MAX(CASE WHEN LOWER(a.slug) IN ('mpn','manufacturer_part_number') THEN av.value END) AS mpn
    FROM variant_attribute_values vav
    JOIN attribute_values av ON av.id = vav.attribute_value_id
    JOIN attributes a        ON a.id  = av.attribute_id
    WHERE vav.variant_id IN (SELECT variant_id FROM eligible_variants)
    GROUP BY vav.variant_id
),
ranked_images AS (
    SELECT pi.product_id, pi.image_url, pi.is_primary,
        ROW_NUMBER() OVER (PARTITION BY pi.product_id ORDER BY pi.is_primary DESC, pi.sort_order ASC, pi.id ASC) AS rn
    FROM product_images pi
    WHERE pi.product_id IN (SELECT product_id FROM eligible_variants)
),
primary_image AS (SELECT product_id, image_url FROM ranked_images WHERE rn = 1),
additional_images AS (
    SELECT product_id, ARRAY_AGG(image_url ORDER BY rn) FILTER (WHERE rn BETWEEN 2 AND 11) AS image_urls
    FROM ranked_images GROUP BY product_id
),
category_primary AS (
    SELECT DISTINCT ON (pcm.product_id) pcm.product_id, pcm.category_id, pc.name AS category_name
    FROM product_category_map pcm
    JOIN product_categories pc ON pc.id = pcm.category_id
    WHERE pcm.product_id IN (SELECT product_id FROM eligible_variants) AND pc.is_active = TRUE
    ORDER BY pcm.product_id, pc.id ASC
),
category_path_cte AS (
    SELECT cp.product_id, STRING_AGG(anc.name, ' > ' ORDER BY cc.depth DESC) AS path
    FROM category_primary cp
    JOIN category_closure cc ON cc.descendant_id = cp.category_id
    JOIN product_categories anc ON anc.id = cc.ancestor_id
    WHERE anc.is_active = TRUE
    GROUP BY cp.product_id
)
SELECT ev.variant_id, ev.product_id, ev.product_slug, ev.sku,
    COALESCE(va.gtin,'') AS gtin, COALESCE(va.mpn,'') AS mpn,
    TRIM(ev.product_name
        || CASE WHEN COALESCE(va.color,'')<>'' OR COALESCE(va.size,'')<>''
               THEN ' - '
                    || COALESCE(NULLIF(va.color,''),'')
                    || CASE WHEN va.color<>'' AND va.size<>'' THEN ' / ' ELSE '' END
                    || COALESCE(NULLIF(va.size,''),'')
               ELSE '' END) AS title,
    COALESCE(ev.description, ev.short_description, ev.product_name) AS description,
    ev.brand_name, 'new' AS condition, ev.is_digital,
    COALESCE(cp.path, COALESCE(cat.category_name,'')) AS category_path,
    '' AS google_product_category,
    ev.base_price, ev.variant_price, ev.currency,
    COALESCE(bd.discount_percent,0)::DOUBLE PRECISION, COALESCE(bd.discount_type,''),
    bd.discount_starts_at, bd.discount_ends_at,
    COALESCE(ibv.available_qty,0), COALESCE(ibv.reserved_qty,0), COALESCE(ibv.incoming_qty,0),
    COALESCE(img.image_url,''), COALESCE(ai.image_urls,'{}'),
    COALESCE(ev.weight_kg,0),
    COALESCE(va.color,''), COALESCE(va.size,''), COALESCE(va.gender,''),
    COALESCE(va.age_group,''), COALESCE(va.material,''), COALESCE(va.pattern,''),
    ev.updated_at
FROM eligible_variants ev
LEFT JOIN best_discounts      bd  ON bd.variant_id  = ev.variant_id
LEFT JOIN inventory_by_variant ibv ON ibv.variant_id = ev.variant_id
LEFT JOIN variant_attrs        va  ON va.variant_id  = ev.variant_id
LEFT JOIN primary_image        img ON img.product_id = ev.product_id
LEFT JOIN additional_images    ai  ON ai.product_id  = ev.product_id
LEFT JOIN category_path_cte    cp  ON cp.product_id  = ev.product_id
LEFT JOIN category_primary     cat ON cat.product_id = ev.product_id
ORDER BY ev.variant_id ASC
`

// ---------------------------------------------------------------------------
// queryCountEligibleVariants
//   Parameters match the eligibility filter in queryLoadFeedBatch.
//   $1 active_only, $2 in_stock_only, $3 exclude_digital,
//   $4 category_ids[], $5 brand_ids[], $6 updated_after
// ---------------------------------------------------------------------------

const queryCountEligibleVariants = `
SELECT COUNT(pv.id)
FROM product_variants pv
JOIN products p ON p.id = pv.product_id
WHERE pv.is_active = TRUE
  AND ($1 = FALSE OR p.status = 'active')
  AND p.status != 'archived'
  AND ($3 = FALSE OR p.is_digital = FALSE)
  AND (COALESCE(array_length($5::BIGINT[], 1), 0) = 0 OR p.brand_id = ANY($5::BIGINT[]))
  AND (COALESCE(array_length($4::BIGINT[], 1), 0) = 0 OR EXISTS (
      SELECT 1 FROM product_category_map pcm
      JOIN category_closure cc ON cc.descendant_id = pcm.category_id
      WHERE pcm.product_id = p.id AND cc.ancestor_id = ANY($4::BIGINT[])
  ))
  AND ($6::TIMESTAMP IS NULL OR p.updated_at > $6 OR pv.created_at > $6)
  AND ($2 = FALSE OR EXISTS (
      SELECT 1 FROM inventory_items ii
      WHERE ii.variant_id = pv.id
      GROUP BY ii.variant_id
      HAVING COALESCE(SUM(ii.available_qty),0) - COALESCE(SUM(ii.reserved_qty),0) > 0
  ))
`

// ---------------------------------------------------------------------------
// queryLoadUpdatedVariantIDs
//   Returns variant IDs whose product or variant was updated after $1.
//   $1 = since TIMESTAMP
//   $2 = LIMIT
// ---------------------------------------------------------------------------

const queryLoadUpdatedVariantIDs = `
SELECT pv.id
FROM product_variants pv
JOIN products p ON p.id = pv.product_id
WHERE pv.is_active = TRUE
  AND p.status = 'active'
  AND (p.updated_at > $1 OR pv.created_at > $1)
ORDER BY GREATEST(p.updated_at, pv.created_at) DESC, pv.id DESC
LIMIT $2
`