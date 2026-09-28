-- Run once with the same search_path as the API, before deploying the new API.
-- Extends existing shipping tables; does not rewrite existing rates or orders.
BEGIN;
SET LOCAL lock_timeout = '5s';

ALTER TABLE shipping_methods
    ADD COLUMN code TEXT UNIQUE,
    ADD COLUMN pricing_mode TEXT NOT NULL DEFAULT 'fixed'
        CHECK (pricing_mode IN ('fixed', 'informational')),
    ADD COLUMN policy_version INTEGER NOT NULL DEFAULT 1 CHECK (policy_version > 0),
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN updated_by BIGINT;

ALTER TABLE shipping_zones
    ADD COLUMN code TEXT UNIQUE,
    ADD COLUMN region_code TEXT;

ALTER TABLE shipping_method_rates
    ALTER COLUMN base_fee DROP NOT NULL,
    ADD COLUMN currency VARCHAR(3) NOT NULL DEFAULT 'KES',
    ADD COLUMN rate_type TEXT NOT NULL DEFAULT 'fixed'
        CHECK (rate_type IN ('fixed', 'indicative', 'quote_required'));

-- NOT VALID preserves legacy rows; the constraint applies to new/updated rows.
ALTER TABLE shipping_method_rates ADD CONSTRAINT delivery_rate_amount_check CHECK (
    (rate_type = 'quote_required' AND base_fee IS NULL) OR
    (rate_type IN ('fixed', 'indicative') AND base_fee IS NOT NULL AND base_fee >= 0)
) NOT VALID;

ALTER TABLE orders ADD COLUMN delivery_information JSONB;

INSERT INTO shipping_methods (code, name, description, pricing_mode)
VALUES ('standard-delivery', 'Standard delivery', '', 'informational');

INSERT INTO shipping_zones (code, name, region_code) VALUES
    ('ke-nationwide', 'Kenya nationwide (standard delivery)', NULL),
    ('ke-nairobi', 'Nairobi (standard delivery)', 'KE-30');

INSERT INTO shipping_zone_countries (zone_id, country)
SELECT id, 'KE' FROM shipping_zones WHERE code IN ('ke-nationwide', 'ke-nairobi');

-- base_fee uses the existing major-unit convention: 300 means KES 300.
INSERT INTO shipping_method_rates (shipping_method_id, zone_id, base_fee, currency, rate_type)
SELECT m.id, z.id, CASE WHEN z.code = 'ke-nairobi' THEN 300 ELSE NULL END,
       'KES', CASE WHEN z.code = 'ke-nairobi' THEN 'indicative' ELSE 'quote_required' END
FROM shipping_methods m CROSS JOIN shipping_zones z
WHERE m.code = 'standard-delivery' AND z.code IN ('ke-nationwide', 'ke-nairobi');
COMMIT;
