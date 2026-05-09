\c zentora;

-- =============================================================================
-- 007_merchant_feed_infrastructure.sql
-- Google Merchant Center Feed – Supporting schema
-- =============================================================================
-- Adds indexes and tables that make the merchant feed queries performant
-- without touching or duplicating any existing tables.
-- =============================================================================

BEGIN;

-- ---------------------------------------------------------------------------
-- Feed generation audit / metadata table
-- Persists one row per generation run for monitoring and caching decisions.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS merchant_feed_runs (
    id               BIGSERIAL PRIMARY KEY,
    feed_id          VARCHAR(100) NOT NULL,
    export_mode      VARCHAR(20)  NOT NULL DEFAULT 'full', -- full | incremental
    total_items      BIGINT       NOT NULL DEFAULT 0,
    valid_items      BIGINT       NOT NULL DEFAULT 0,
    invalid_items    BIGINT       NOT NULL DEFAULT 0,
    error_count      INT          NOT NULL DEFAULT 0,
    duration_ms      INT,
    schema_version   VARCHAR(20)  NOT NULL DEFAULT '1.0',
    started_at       TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at     TIMESTAMP,
    status           VARCHAR(20)  NOT NULL DEFAULT 'running', -- running | completed | failed
    error_message    TEXT
);

CREATE INDEX IF NOT EXISTS idx_merchant_feed_runs_feed_started
    ON merchant_feed_runs (feed_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_merchant_feed_runs_status
    ON merchant_feed_runs (status, started_at DESC);

-- ---------------------------------------------------------------------------
-- Merchant item overrides
-- Allows per-variant manual overrides of GMC fields (GTIN, MPN, condition,
-- custom labels, google_product_category) without altering the core schema.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS merchant_item_overrides (
    variant_id             BIGINT       PRIMARY KEY,
    gtin                   VARCHAR(14),
    mpn                    VARCHAR(100),
    condition              VARCHAR(20),
    google_product_category VARCHAR(500),
    custom_label_0         VARCHAR(100),
    custom_label_1         VARCHAR(100),
    custom_label_2         VARCHAR(100),
    custom_label_3         VARCHAR(100),
    custom_label_4         VARCHAR(100),
    energy_efficiency_class VARCHAR(10),
    multipack              INT,
    adult                  BOOLEAN DEFAULT FALSE,
    expiration_date        DATE,
    unit_pricing_measure   VARCHAR(50),
    unit_pricing_base_measure VARCHAR(50),
    ads_redirect           VARCHAR(500),
    pickup_method          VARCHAR(30),
    pickup_sla             VARCHAR(30),
    cost_of_goods_sold     DECIMAL(12,2),
    updated_at             TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (variant_id) REFERENCES product_variants(id) ON DELETE CASCADE
);

-- ---------------------------------------------------------------------------
-- Google Product Category mapping
-- Maps internal category IDs to GMC taxonomy paths.
-- e.g. category_id=42 → "Electronics > Communications > Telephony > Mobile Phones"
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS merchant_category_map (
    category_id              BIGINT       PRIMARY KEY,
    google_product_category  VARCHAR(500) NOT NULL,
    gmc_category_id          INT,                  -- numeric GMC taxonomy ID
    updated_at               TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (category_id) REFERENCES product_categories(id) ON DELETE CASCADE
);

-- ---------------------------------------------------------------------------
-- Shipping configuration for merchant feed (per-country overrides)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS merchant_shipping_config (
    id          BIGSERIAL  PRIMARY KEY,
    country     VARCHAR(2) NOT NULL,  -- ISO 3166-1 alpha-2, e.g. "KE"
    service     VARCHAR(100) NOT NULL,
    price       DECIMAL(12,2) NOT NULL DEFAULT 0,
    currency    VARCHAR(10)   NOT NULL DEFAULT 'KES',
    is_active   BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_merchant_shipping_country_service
    ON merchant_shipping_config (country, service)
    WHERE is_active = TRUE;

-- ---------------------------------------------------------------------------
-- Indexes to accelerate merchant feed queries
--
-- All existing tables; CONCURRENTLY avoids locking in production.
-- ---------------------------------------------------------------------------

-- Feed batch scan: active variants ordered by id (keyset pagination)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pv_merchant_active_id
    ON product_variants (id ASC)
    WHERE is_active = TRUE;

-- Filter by product status + updated_at (incremental mode)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_products_status_updated
    ON products (status, updated_at DESC)
    WHERE status = 'active';

-- Inventory aggregation (critical join in the feed query)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_inventory_variant_qty
    ON inventory_items (variant_id, available_qty, reserved_qty, incoming_qty);

-- Variant attribute lookup (EAV pivot; covers attribute_id + variant_id)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_vav_variant_attr
    ON variant_attribute_values (variant_id, attribute_value_id);

-- Image hydration by product, ordered for primary selection
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_product_images_hydration
    ON product_images (product_id, is_primary DESC, sort_order ASC, id ASC);

-- Discount targets scan (product / brand / category)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_discount_targets_type_id
    ON discount_targets (target_type, target_id, discount_id);

-- Category closure ancestor filter (used in category eligibility check)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_category_closure_ancestor
    ON category_closure (ancestor_id, descendant_id);

COMMIT;