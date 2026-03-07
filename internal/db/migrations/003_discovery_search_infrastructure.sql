\c zentora;

BEGIN;

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ===========================
-- SEARCH EVENTS
-- ===========================
CREATE TABLE search_events (
    id BIGSERIAL PRIMARY KEY,
    query TEXT NOT NULL,
    normalized_query TEXT NOT NULL,
    user_id BIGINT,
    session_id VARCHAR(255),
    result_count INT NOT NULL DEFAULT 0 CHECK (result_count >= 0),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES auth_identities(id) ON DELETE SET NULL
);
CREATE INDEX idx_search_events_created_query ON search_events(created_at, normalized_query);
CREATE INDEX idx_search_events_user_created ON search_events(user_id, created_at);
CREATE INDEX idx_search_events_session_created ON search_events(session_id, created_at);
CREATE INDEX idx_search_events_normalized_query_trgm ON search_events USING GIN(normalized_query gin_trgm_ops);

CREATE TABLE search_clicks (
    id BIGSERIAL PRIMARY KEY,
    search_event_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    position INT NOT NULL CHECK (position > 0),
    user_id BIGINT,
    session_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (search_event_id) REFERENCES search_events(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES auth_identities(id) ON DELETE SET NULL
);
CREATE INDEX idx_search_clicks_event_position ON search_clicks(search_event_id, position);
CREATE INDEX idx_search_clicks_product_created ON search_clicks(product_id, created_at);

CREATE TABLE search_result_positions (
    search_event_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    position INT NOT NULL CHECK (position > 0),
    score DOUBLE PRECISION NOT NULL,
    PRIMARY KEY (search_event_id, product_id),
    FOREIGN KEY (search_event_id) REFERENCES search_events(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX idx_search_result_positions_event_position ON search_result_positions(search_event_id, position);

-- ===========================
-- SEARCH DOCUMENTS
-- ===========================
CREATE TABLE product_search_documents (
    product_id BIGINT PRIMARY KEY,
    search_document TEXT NOT NULL,
    search_vector TSVECTOR NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_search_documents_vector ON product_search_documents USING GIN(search_vector);
CREATE INDEX idx_product_search_documents_document_trgm ON product_search_documents USING GIN(search_document gin_trgm_ops);
CREATE INDEX idx_product_search_documents_updated_at ON product_search_documents(updated_at);

-- ===========================
-- MERCHANDISING BOOSTS
-- ===========================
CREATE TABLE product_boosts (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL,
    feed_type VARCHAR(50) NOT NULL CHECK (feed_type ~ '^[a-z0-9_]+$'),
    boost_score DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (boost_score >= 0),
    start_at TIMESTAMP NOT NULL,
    end_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (end_at > start_at),
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_boosts_feed_window ON product_boosts(feed_type, start_at, end_at);
CREATE INDEX idx_product_boosts_product_feed ON product_boosts(product_id, feed_type);

COMMIT;
