\c zentora;

BEGIN;

-- ===========================
-- USERS / ADDRESSES
-- ===========================
CREATE TABLE user_addresses (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    phone_number VARCHAR(30) NOT NULL,
    country VARCHAR(100) NOT NULL,
    county VARCHAR(100),
    city VARCHAR(100) NOT NULL,
    area VARCHAR(255),
    postal_code VARCHAR(20),
    address_line_1 VARCHAR(255) NOT NULL,
    address_line_2 VARCHAR(255),
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES auth_identities(id) ON DELETE CASCADE
);
CREATE INDEX idx_user_addresses_user ON user_addresses(user_id);

-- ===========================
-- CATEGORY HIERARCHY
-- ===========================
CREATE TABLE product_categories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    parent_id BIGINT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (parent_id) REFERENCES product_categories(id)
);
CREATE INDEX idx_categories_parent ON product_categories(parent_id);
CREATE INDEX idx_categories_slug ON product_categories(slug);

CREATE TABLE category_closure (
    ancestor_id BIGINT NOT NULL,
    descendant_id BIGINT NOT NULL,
    depth INT NOT NULL,
    PRIMARY KEY (ancestor_id, descendant_id),
    FOREIGN KEY (ancestor_id) REFERENCES product_categories(id) ON DELETE CASCADE,
    FOREIGN KEY (descendant_id) REFERENCES product_categories(id) ON DELETE CASCADE
);
CREATE INDEX idx_category_closure_desc ON category_closure(descendant_id);

-- ===========================
-- BRANDS
-- ===========================
CREATE TABLE product_brands (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    slug VARCHAR(255) NOT NULL UNIQUE,
    logo_url VARCHAR(500),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ===========================
-- TAGS
-- ===========================
CREATE TABLE tags (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE
);

-- ===========================
-- PRODUCTS
-- ===========================
CREATE TABLE products (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    short_description TEXT,
    brand_id BIGINT,
    base_price DECIMAL(12,2) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, draft, archived
    is_featured BOOLEAN DEFAULT FALSE,
    is_digital BOOLEAN DEFAULT FALSE,
    rating DECIMAL(3,2) DEFAULT 0,
    review_count INT DEFAULT 0,
    created_by BIGINT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (brand_id) REFERENCES product_brands(id),
    FOREIGN KEY (created_by) REFERENCES auth_identities(id)
);
CREATE INDEX idx_products_status ON products(status);
CREATE INDEX idx_products_featured ON products(is_featured);
CREATE INDEX idx_products_brand ON products(brand_id);
CREATE INDEX idx_products_created_at ON products(created_at);

CREATE TABLE product_category_map (
    product_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL,
    PRIMARY KEY (product_id, category_id),
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES product_categories(id) ON DELETE CASCADE
);
CREATE INDEX idx_pcm_category ON product_category_map(category_id);

CREATE TABLE product_tags (
    product_id BIGINT NOT NULL,
    tag_id BIGINT NOT NULL,
    PRIMARY KEY (product_id, tag_id),
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE TABLE product_images (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL,
    image_url VARCHAR(500) NOT NULL,
    is_primary BOOLEAN DEFAULT FALSE,
    sort_order INT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_images_product ON product_images(product_id, is_primary, sort_order);

-- ===========================
-- ATTRIBUTES / VARIANTS
-- ===========================
CREATE TABLE attributes (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    is_variant_dimension BOOLEAN DEFAULT FALSE
);

CREATE TABLE attribute_values (
    id BIGSERIAL PRIMARY KEY,
    attribute_id BIGINT NOT NULL,
    value VARCHAR(100) NOT NULL,
    FOREIGN KEY (attribute_id) REFERENCES attributes(id) ON DELETE CASCADE,
    UNIQUE(attribute_id, value)
);

CREATE TABLE product_attribute_values (
    product_id BIGINT NOT NULL,
    attribute_value_id BIGINT NOT NULL,
    PRIMARY KEY (product_id, attribute_value_id),
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    FOREIGN KEY (attribute_value_id) REFERENCES attribute_values(id) ON DELETE CASCADE
);

CREATE TABLE product_variants (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL,
    sku VARCHAR(100) UNIQUE NOT NULL,
    price DECIMAL(12,2) NOT NULL,
    weight DECIMAL(10,2),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);
CREATE INDEX idx_variants_product ON product_variants(product_id, is_active);

CREATE TABLE variant_attribute_values (
    variant_id BIGINT NOT NULL,
    attribute_value_id BIGINT NOT NULL,
    PRIMARY KEY (variant_id, attribute_value_id),
    FOREIGN KEY (variant_id) REFERENCES product_variants(id) ON DELETE CASCADE,
    FOREIGN KEY (attribute_value_id) REFERENCES attribute_values(id)
);

-- ===========================
-- INVENTORY
-- ===========================
CREATE TABLE inventory_locations (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    location_code VARCHAR(50) UNIQUE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE inventory_items (
    id BIGSERIAL PRIMARY KEY,
    variant_id BIGINT NOT NULL,
    location_id BIGINT NOT NULL,
    available_qty INT NOT NULL DEFAULT 0,
    reserved_qty INT NOT NULL DEFAULT 0,
    incoming_qty INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (variant_id) REFERENCES product_variants(id) ON DELETE CASCADE,
    FOREIGN KEY (location_id) REFERENCES inventory_locations(id) ON DELETE CASCADE,
    UNIQUE(variant_id, location_id)
);
CREATE INDEX idx_inventory_variant ON inventory_items(variant_id);
CREATE INDEX idx_inventory_location ON inventory_items(location_id);

-- ===========================
-- DISCOUNTS / PROMOS
-- ===========================
CREATE TABLE discounts (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50),
    discount_type VARCHAR(20) NOT NULL, -- percentage, fixed
    value DECIMAL(10,2) NOT NULL,
    min_order_amount DECIMAL(12,2),
    max_redemptions INT,
    starts_at TIMESTAMP,
    ends_at TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX idx_discounts_code ON discounts(code);

CREATE TABLE discount_targets (
    discount_id BIGINT NOT NULL,
    target_type VARCHAR(20) NOT NULL, -- product, category, brand
    target_id BIGINT NOT NULL,
    PRIMARY KEY (discount_id, target_type, target_id),
    FOREIGN KEY (discount_id) REFERENCES discounts(id) ON DELETE CASCADE
);

CREATE TABLE discount_redemptions (
    id BIGSERIAL PRIMARY KEY,
    discount_id BIGINT NOT NULL,
    order_id BIGINT NOT NULL,
    user_id BIGINT,
    redeemed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (discount_id) REFERENCES discounts(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES auth_identities(id),
    UNIQUE(discount_id, order_id)
);

-- ===========================
-- CARTS
-- ===========================
CREATE TABLE carts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT,
    session_id VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, converted, abandoned
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES auth_identities(id) ON DELETE CASCADE
);
CREATE INDEX idx_carts_user_status ON carts(user_id, status);
CREATE INDEX idx_carts_session_status ON carts(session_id, status);

CREATE TABLE cart_items (
    id BIGSERIAL PRIMARY KEY,
    cart_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    variant_id BIGINT,
    quantity INT NOT NULL CHECK (quantity > 0),
    price_at_added DECIMAL(12,2) NOT NULL,
    added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (cart_id) REFERENCES carts(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id),
    FOREIGN KEY (variant_id) REFERENCES product_variants(id),
    UNIQUE(cart_id, variant_id)
);
CREATE INDEX idx_cart_items_cart ON cart_items(cart_id);

-- ===========================
-- SHIPPING
-- ===========================
CREATE TABLE shipping_methods (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE shipping_zones (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL UNIQUE,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE shipping_zone_countries (
    zone_id BIGINT NOT NULL,
    country VARCHAR(100) NOT NULL,
    PRIMARY KEY (zone_id, country),
    FOREIGN KEY (zone_id) REFERENCES shipping_zones(id) ON DELETE CASCADE
);

CREATE TABLE shipping_method_rates (
    id BIGSERIAL PRIMARY KEY,
    shipping_method_id BIGINT NOT NULL,
    zone_id BIGINT NOT NULL,
    base_fee DECIMAL(12,2) NOT NULL,
    estimated_days_min INT,
    estimated_days_max INT,
    FOREIGN KEY (shipping_method_id) REFERENCES shipping_methods(id) ON DELETE CASCADE,
    FOREIGN KEY (zone_id) REFERENCES shipping_zones(id) ON DELETE CASCADE,
    UNIQUE(shipping_method_id, zone_id)
);

-- ===========================
-- ORDERS
-- ===========================
CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT,
    cart_id BIGINT,
    order_number VARCHAR(50) UNIQUE NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    subtotal DECIMAL(12,2) NOT NULL,
    discount_amount DECIMAL(12,2) DEFAULT 0,
    tax_amount DECIMAL(12,2) DEFAULT 0,
    shipping_fee DECIMAL(12,2) DEFAULT 0,
    total_amount DECIMAL(12,2) NOT NULL,
    currency VARCHAR(10) DEFAULT 'KES',
    shipping_method_id BIGINT,

    shipping_full_name VARCHAR(255) NOT NULL,
    shipping_phone VARCHAR(30) NOT NULL,
    shipping_country VARCHAR(100) NOT NULL,
    shipping_county VARCHAR(100),
    shipping_city VARCHAR(100) NOT NULL,
    shipping_area VARCHAR(255),
    shipping_postal_code VARCHAR(20),
    shipping_address_line_1 VARCHAR(255) NOT NULL,
    shipping_address_line_2 VARCHAR(255),

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES auth_identities(id),
    FOREIGN KEY (shipping_method_id) REFERENCES shipping_methods(id)
);
CREATE INDEX idx_orders_user_created ON orders(user_id, created_at);
CREATE INDEX idx_orders_status ON orders(status);

CREATE TABLE order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    variant_id BIGINT,
    product_name VARCHAR(255) NOT NULL,
    product_slug VARCHAR(255),
    variant_sku VARCHAR(100),
    variant_name VARCHAR(255),
    image_url VARCHAR(500),
    unit_price DECIMAL(12,2) NOT NULL,
    quantity INT NOT NULL,
    discount_amount DECIMAL(12,2) DEFAULT 0,
    tax_rate DECIMAL(6,4) DEFAULT 0,
    total_price DECIMAL(12,2) NOT NULL,
    currency VARCHAR(10) DEFAULT 'KES',
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
);
CREATE INDEX idx_order_items_order ON order_items(order_id);

CREATE TABLE order_payments (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL,
    provider VARCHAR(50), -- mpesa, card, paypal
    transaction_reference VARCHAR(255),
    amount DECIMAL(12,2) NOT NULL,
    status VARCHAR(30) DEFAULT 'pending',
    attempt_number INT DEFAULT 1,
    paid_at TIMESTAMP,
    captured_at TIMESTAMP,
    refunded_at TIMESTAMP,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    UNIQUE(provider, transaction_reference)
);
CREATE INDEX idx_order_payments_order ON order_payments(order_id, status);

-- ===========================
-- FULFILLMENT / TRACKING
-- ===========================
CREATE TABLE order_fulfillments (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending', -- pending, shipped, delivered, cancelled
    shipped_at TIMESTAMP,
    delivered_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
);

CREATE TABLE shipment_tracking (
    id BIGSERIAL PRIMARY KEY,
    fulfillment_id BIGINT NOT NULL,
    carrier VARCHAR(100),
    tracking_number VARCHAR(100),
    status VARCHAR(50),
    event_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    location VARCHAR(255),
    FOREIGN KEY (fulfillment_id) REFERENCES order_fulfillments(id) ON DELETE CASCADE
);
CREATE INDEX idx_tracking_fulfillment ON shipment_tracking(fulfillment_id, event_time);

-- ===========================
-- REVIEWS
-- ===========================
CREATE TABLE reviews (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    order_item_id BIGINT,
    rating INT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment TEXT,
    is_verified_purchase BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, product_id),
    FOREIGN KEY (user_id) REFERENCES auth_identities(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    FOREIGN KEY (order_item_id) REFERENCES order_items(id) ON DELETE SET NULL
);
CREATE INDEX idx_reviews_product ON reviews(product_id);

-- ===========================
-- WISHLISTS
-- ===========================
CREATE TABLE wishlists (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES auth_identities(id) ON DELETE CASCADE
);

CREATE TABLE wishlist_items (
    wishlist_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    variant_id BIGINT,
    added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (wishlist_id, product_id, variant_id),
    FOREIGN KEY (wishlist_id) REFERENCES wishlists(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    FOREIGN KEY (variant_id) REFERENCES product_variants(id) ON DELETE CASCADE
);

-- ===========================
-- ANALYTICS EVENTS (partition-ready)
-- ===========================
CREATE TABLE product_events (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL,
    user_id BIGINT,
    session_id VARCHAR(255),
    event_type VARCHAR(30) NOT NULL, -- view, add_to_cart, purchase
    quantity INT DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_product_events_product_time ON product_events(product_id, created_at);
CREATE INDEX idx_product_events_user_time ON product_events(user_id, created_at);
CREATE INDEX idx_product_events_type_time ON product_events(event_type, created_at);

-- ===========================
-- METRICS CACHE
-- ===========================
CREATE TABLE product_metrics (
    product_id BIGINT PRIMARY KEY,
    trending_score FLOAT DEFAULT 0,
    daily_views INT DEFAULT 0,
    weekly_views INT DEFAULT 0,
    weekly_purchases INT DEFAULT 0,
    conversion_rate FLOAT DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_metrics_trending ON product_metrics(trending_score DESC);

-- ===========================
-- RECOMMENDATION SUPPORT
-- ===========================
CREATE TABLE product_co_views (
    product_id BIGINT NOT NULL,
    related_product_id BIGINT NOT NULL,
    score FLOAT,
    PRIMARY KEY (product_id, related_product_id)
);

CREATE TABLE user_category_affinity (
    user_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL,
    score FLOAT DEFAULT 0,
    PRIMARY KEY (user_id, category_id)
);

-- ===========================
-- HOMEPAGE CMS
-- ===========================
CREATE TABLE banners (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255),
    image_url VARCHAR(500),
    redirect_type VARCHAR(50), -- product, category, external
    redirect_id BIGINT,
    is_active BOOLEAN DEFAULT TRUE,
    start_date TIMESTAMP,
    end_date TIMESTAMP,
    priority INT DEFAULT 0
);

CREATE TABLE homepage_sections (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255),
    type VARCHAR(50), -- trending, featured, category, custom
    reference_id BIGINT,
    sort_order INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE
);

COMMIT;