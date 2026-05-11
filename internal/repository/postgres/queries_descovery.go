package postgres

const suggestQuery = `
WITH input AS (
    SELECT
        LOWER($1)                      AS prefix,
        LOWER($1) || '%'               AS like_prefix
),
product_matches AS (
    SELECT
        p.name                                                         AS text,
        'product'                                                      AS type,
        p.id                                                           AS reference_id,
        (
            CASE WHEN LOWER(p.name) LIKE i.like_prefix THEN $2 ELSE 0.0 END
            + GREATEST(
                similarity(LOWER(p.name), i.prefix),
                word_similarity(i.prefix, LOWER(p.name))
              )
            + COALESCE(pm.weekly_views, 0)::DOUBLE PRECISION / 1000.0
        )                                                              AS score
    FROM products p
    CROSS JOIN input i
    LEFT JOIN product_metrics pm ON pm.product_id = p.id
    WHERE p.status = $3
      AND (
          LOWER(p.name) LIKE i.like_prefix
          OR word_similarity(i.prefix, LOWER(p.name)) > 0.2
      )
    ORDER BY score DESC, p.id DESC
    LIMIT 20
),
category_matches AS (
    SELECT
        c.name                                                         AS text,
        'category'                                                     AS type,
        c.id                                                           AS reference_id,
        (
            CASE WHEN LOWER(c.name) LIKE i.like_prefix THEN $2 ELSE 0.0 END
            + GREATEST(
                similarity(LOWER(c.name), i.prefix),
                word_similarity(i.prefix, LOWER(c.name))
              )
        )                                                              AS score
    FROM product_categories c
    CROSS JOIN input i
    WHERE c.is_active = TRUE
      AND (
          LOWER(c.name) LIKE i.like_prefix
          OR word_similarity(i.prefix, LOWER(c.name)) > 0.2
      )
      -- only include categories that actually have active products
      AND EXISTS (
          SELECT 1
          FROM product_category_map pcm
          JOIN products p ON p.id = pcm.product_id
          WHERE pcm.category_id = c.id
            AND p.status = $3
      )
    ORDER BY score DESC, c.id DESC
    LIMIT 10
),
brand_matches AS (
    SELECT
        b.name                                                         AS text,
        'brand'                                                        AS type,
        b.id                                                           AS reference_id,
        (
            CASE WHEN LOWER(b.name) LIKE i.like_prefix THEN $2 ELSE 0.0 END
            + GREATEST(
                similarity(LOWER(b.name), i.prefix),
                word_similarity(i.prefix, LOWER(b.name))
              )
        )                                                              AS score
    FROM product_brands b
    CROSS JOIN input i
    WHERE b.is_active = TRUE
      AND (
          LOWER(b.name) LIKE i.like_prefix
          OR word_similarity(i.prefix, LOWER(b.name)) > 0.2
      )
      -- only include brands with active products
      AND EXISTS (
          SELECT 1
          FROM products p
          WHERE p.brand_id = b.id
            AND p.status = $3
      )
    ORDER BY score DESC, b.id DESC
    LIMIT 10
),
query_matches AS (
    SELECT
        se.normalized_query                                            AS text,
        'query'                                                        AS type,
        NULL::BIGINT                                                   AS reference_id,
        (
            COUNT(*)::DOUBLE PRECISION
            + CASE WHEN se.normalized_query LIKE i.like_prefix THEN $2 ELSE 0.0 END
            + GREATEST(
                similarity(se.normalized_query, i.prefix),
                word_similarity(i.prefix, se.normalized_query)
              )
        )                                                              AS score
    FROM (
        -- cap the scan; search_events can be huge
        SELECT normalized_query
        FROM search_events
        ORDER BY id DESC
        LIMIT 50000
    ) se
    CROSS JOIN input i
    WHERE (
        se.normalized_query LIKE i.like_prefix
        OR word_similarity(i.prefix, se.normalized_query) > 0.2
    )
    GROUP BY se.normalized_query, i.prefix, i.like_prefix
    ORDER BY score DESC
    LIMIT 10
),
all_suggestions AS (
    SELECT text, type, reference_id, score FROM product_matches
    UNION ALL
    SELECT text, type, reference_id, score FROM category_matches
    UNION ALL
    SELECT text, type, reference_id, score FROM brand_matches
    UNION ALL
    SELECT text, type, reference_id, score FROM query_matches
)
SELECT text, type, reference_id, score
FROM all_suggestions
ORDER BY score DESC, text ASC
LIMIT $4`


const searchCandidateQuery = `
WITH search_input AS (
    SELECT
        LOWER($1)                                                       AS raw,
        LOWER($1) || '%'                                                AS like_prefix,
        -- plainto_tsquery is safe for arbitrary user text
        plainto_tsquery($3::regconfig, LOWER($1))                       AS ts_query
),
fts_candidates AS (
    SELECT
        psd.product_id,
        ts_rank_cd(psd.search_vector, si.ts_query, 32)::DOUBLE PRECISION AS text_relevance
    FROM product_search_documents psd
    CROSS JOIN search_input si
    JOIN products p ON p.id = psd.product_id
    WHERE p.status = $2
      -- guard: empty tsquery (e.g. stop words only) skips FTS entirely
      AND si.ts_query <> ''::tsquery
      AND psd.search_vector @@ si.ts_query
),
trigram_candidates AS (
    SELECT
        psd.product_id,
        -- scale to 0–1 range comparable with ts_rank_cd
        LEAST(
            GREATEST(
                word_similarity(si.raw, LOWER(p.name)),
                similarity(LOWER(psd.search_document), si.raw)
            ) * 1.2,   -- slight boost so partial matches stay competitive
            1.0
        )::DOUBLE PRECISION AS text_relevance
    FROM product_search_documents psd
    CROSS JOIN search_input si
    JOIN products p ON p.id = psd.product_id
    WHERE p.status = $2
      AND (
          -- name prefix match (fast, uses btree index on lower(name))
          LOWER(p.name) LIKE si.like_prefix
          -- word-level trigram on the name (good for mid-string matches)
          OR word_similarity(si.raw, LOWER(p.name)) > 0.2
          -- full-document trigram fallback
          OR similarity(LOWER(psd.search_document), si.raw) > 0.15
      )
),
combined AS (
    SELECT product_id, MAX(text_relevance) AS text_relevance
    FROM (
        SELECT * FROM fts_candidates
        UNION ALL
        SELECT * FROM trigram_candidates
    ) c
    GROUP BY product_id
)
SELECT
    c.product_id,
    c.text_relevance,
    COALESCE(pm.weekly_purchases, 0)::DOUBLE PRECISION  AS popularity_score,
    COALESCE(pm.conversion_rate,  0)::DOUBLE PRECISION  AS conversion_rate,
    COALESCE(p.rating,            0)::DOUBLE PRECISION  AS rating_score,
    COALESCE(pm.trending_score,   0)::DOUBLE PRECISION  AS trending_score
FROM combined c
JOIN products p ON p.id = c.product_id
LEFT JOIN product_metrics pm ON pm.product_id = c.product_id`

const searchCandidateOrderBy = `
    (0.50 * ranked.text_relevance
   + 0.20 * ranked.popularity_score
   + 0.15 * ranked.conversion_rate
   + 0.10 * ranked.rating_score
   + 0.05 * ranked.trending_score) DESC,
    ranked.product_id DESC`