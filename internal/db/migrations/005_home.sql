\c zentora;
BEGIN;
-- Covering index for the hot read path: active sections ordered by sort_order.
-- With only the WHERE and ORDER BY columns in the index the planner can
-- satisfy ListActiveSections with an index-only scan.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_homepage_sections_active_order
    ON homepage_sections (is_active, sort_order ASC, id ASC)
    WHERE is_active = TRUE;

-- Partial index for admin type-filtered queries.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_homepage_sections_type
    ON homepage_sections (type, sort_order ASC)
    WHERE is_active = TRUE;

COMMIT;