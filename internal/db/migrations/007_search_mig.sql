-- Migration: improve search and suggest performance
-- Requires PostgreSQL 12+ with pg_trgm extension.
-- Run once; idempotent.
-- ALTER SYSTEM SET pg_trgm.word_similarity_threshold = 0.2;

-- 1. Ensure pg_trgm is available (needed for word_similarity, similarity, %)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 2. Lower the word-similarity threshold so short prefixes match mid-word tokens.
--    Default is 0.6 — far too high for 2-3 char autocomplete prefixes.
--    0.2 matches what the queries use explicitly via > 0.2 comparisons.
--    This only affects the implicit % operator; our queries use explicit
--    threshold comparisons so this is belt-and-suspenders.
-- SELECT pg_reload_conf();

-- 3. Trigram GIN index on product name (case-insensitive) for suggest speed
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_products_name_trgm
    ON products USING GIN (LOWER(name) gin_trgm_ops);

-- 4. Trigram GIN index on the search document for full-text fallback
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_product_search_doc_trgm
    ON product_search_documents USING GIN (search_document gin_trgm_ops);

-- 5. Trigram GIN indexes on category and brand names for suggest
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_product_categories_name_trgm
    ON product_categories USING GIN (LOWER(name) gin_trgm_ops);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_product_brands_name_trgm
    ON product_brands USING GIN (LOWER(name) gin_trgm_ops);

-- 6. BTree index on lower(name) for fast LIKE 'prefix%' scans
--    (GIN covers similarity; BTree covers prefix LIKE)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_products_name_lower_btree
    ON products (LOWER(name) text_pattern_ops);

-- 7. Index for search_events prefix scan (query_matches CTE)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_search_events_normalized_query
    ON search_events (normalized_query text_pattern_ops);

