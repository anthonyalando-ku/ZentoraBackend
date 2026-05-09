\c zentora;

-- =============================================================================
-- 008_merchant_category_seed.sql
-- Seeds merchant_category_map with Google Merchant Center taxonomy paths
-- for every category currently in the Zentora product_categories table.
--
-- GMC Taxonomy reference: https://www.google.com/basepages/producttype/taxonomy-with-ids.en-US.txt
-- =============================================================================

BEGIN;

INSERT INTO merchant_category_map (category_id, google_product_category, gmc_category_id)
VALUES
    -- =========================================================================
    -- Root / top-level categories
    -- =========================================================================

    -- id=1  Electronics
    (1,  'Electronics', 222),

    -- id=12 Kitchen  →  Home & Garden > Kitchen & Dining
    (12, 'Home & Garden > Kitchen & Dining', 638),

    -- id=13 Home & Living  →  Home & Garden
    (13, 'Home & Garden', 536),

    -- id=14 Beauty & Personal Care  →  Health & Beauty > Personal Care
    (14, 'Health & Beauty > Personal Care', 2915),

    -- id=15 Sports & Outdoors  →  Sporting Goods
    (15, 'Sporting Goods', 499),

    -- id=16 Fashion & Apparel  →  Apparel & Accessories
    (16, 'Apparel & Accessories', 166),

    -- id=17 Baby & Kids  →  Baby & Toddler
    (17, 'Baby & Toddler', 537),

    -- id=18 Health & Wellness  →  Health & Beauty
    (18, 'Health & Beauty', 491),

    -- id=19 Automotive  →  Vehicles & Parts
    (19, 'Vehicles & Parts', 888),

    -- id=20 Books & Stationery  →  Media > Books
    (20, 'Media > Books', 784),

    -- id=23 Car Accessories  →  Vehicles & Parts > Vehicle Parts & Accessories
    (23, 'Vehicles & Parts > Vehicle Parts & Accessories', 5613),

    -- id=24 Emergency Tools  →  Hardware > Tools > Hand Tools
    (24, 'Hardware > Tools > Hand Tools', 1567),

    -- id=25 Power Banks  →  Electronics > Electronics Accessories > Power > Portable Power Stations
    (25, 'Electronics > Electronics Accessories > Power > Portable Power Stations', 7516),

    -- id=26 Air Compressors  →  Tools & Hardware > Power Tools > Air Compressors
    (26, 'Hardware > Tools > Power Tools > Air Compressors', 3650),

    -- id=27 Android Tablets  →  Electronics > Computers > Tablet Computers
    (27, 'Electronics > Computers > Tablet Computers', 4745),

    -- id=28 Mobile Devices  →  Electronics > Communications > Telephony > Mobile Phones
    (28, 'Electronics > Communications > Telephony > Mobile Phones', 267),

    -- id=29 Smart Devices  →  Electronics > Networking > Smart Home Devices
    (29, 'Electronics > Networking > Smart Home Devices', 500044),

    -- id=30 Tablets  →  Electronics > Computers > Tablet Computers
    (30, 'Electronics > Computers > Tablet Computers', 4745),

    -- id=31 Agriculture  →  Business & Industrial > Agriculture
    (31, 'Business & Industrial > Agriculture', 111),

    -- id=32 Tools  →  Hardware > Tools
    (32, 'Hardware > Tools', 1305)

ON CONFLICT (category_id) DO UPDATE
    SET google_product_category = EXCLUDED.google_product_category,
        gmc_category_id         = EXCLUDED.gmc_category_id,
        updated_at              = CURRENT_TIMESTAMP;

COMMIT;