-- =============================================================================
-- ZENTORA MARKETPLACE — COMPLETE MERGED SCHEMA
-- Combines: market_place_migration.sql + missing.sql (extensions)
-- PostgreSQL 14+ | TimescaleDB required
-- =============================================================================
-- Schemas: auth · catalog · accounting
-- All monetary amounts stored as BIGINT in smallest currency unit.
-- Money convention: KES 100.50 → 10050 | USD 1.99 → 199
-- =============================================================================

\c zentora;

-- =============================================================================
-- EXTENSIONS
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- =============================================================================
-- SCHEMAS
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS catalog;
CREATE SCHEMA IF NOT EXISTS accounting;

-- =============================================================================
-- SHARED TRIGGER FUNCTION
-- =============================================================================

BEGIN;

CREATE OR REPLACE FUNCTION public.fn_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMIT;

-- =============================================================================
-- =============================================================================
-- SCHEMA: auth
-- Identity, sessions, RBAC, audit, addresses, notifications
-- =============================================================================
-- =============================================================================

BEGIN;

-- ---------------------------------------------------------------------------
-- auth.identities
-- ---------------------------------------------------------------------------
CREATE TABLE auth.identities (
    id                    BIGSERIAL    PRIMARY KEY,
    email                 VARCHAR(255) NULL,
    email_verified        BOOLEAN      NOT NULL DEFAULT FALSE,
    email_verified_at     TIMESTAMPTZ  NULL,
    phone                 VARCHAR(20)  NULL,
    phone_verified        BOOLEAN      NOT NULL DEFAULT FALSE,
    phone_verified_at     TIMESTAMPTZ  NULL,
    status                VARCHAR(30)  NOT NULL DEFAULT 'pending_verification'
                              CHECK (status IN ('active','inactive','suspended','pending_verification')),
    last_login            TIMESTAMPTZ  NULL,
    failed_login_attempts INT          NOT NULL DEFAULT 0,
    locked_until          TIMESTAMPTZ  NULL,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at            TIMESTAMPTZ  NULL,
    CONSTRAINT chk_email_or_phone CHECK (email IS NOT NULL OR phone IS NOT NULL)
);

CREATE UNIQUE INDEX idx_auth_identities_email
    ON auth.identities (LOWER(email))
    WHERE deleted_at IS NULL AND email IS NOT NULL;
CREATE UNIQUE INDEX idx_auth_identities_phone
    ON auth.identities (phone)
    WHERE deleted_at IS NULL AND phone IS NOT NULL;
CREATE INDEX idx_auth_identities_status
    ON auth.identities (status) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_auth_identities_updated_at
    BEFORE UPDATE ON auth.identities
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

-- ---------------------------------------------------------------------------
-- auth.profiles
-- ---------------------------------------------------------------------------
CREATE TABLE auth.profiles (
    id          BIGSERIAL    PRIMARY KEY,
    identity_id BIGINT       NOT NULL UNIQUE REFERENCES auth.identities(id) ON DELETE CASCADE,
    first_name  VARCHAR(255) NULL,
    last_name   VARCHAR(255) NULL,
    full_name   VARCHAR(255) GENERATED ALWAYS AS (
                    TRIM(COALESCE(first_name,'') || ' ' || COALESCE(last_name,''))
                ) STORED,
    avatar_url  VARCHAR(500) NULL,
    bio         TEXT         NULL,
    metadata    JSONB        NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_auth_profiles_identity  ON auth.profiles(identity_id);
CREATE INDEX idx_auth_profiles_full_name ON auth.profiles(full_name) WHERE full_name IS NOT NULL;

CREATE TRIGGER trg_auth_profiles_updated_at
    BEFORE UPDATE ON auth.profiles
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

-- ---------------------------------------------------------------------------
-- auth.providers
-- ---------------------------------------------------------------------------
CREATE TABLE auth.providers (
    id                  BIGSERIAL   PRIMARY KEY,
    identity_id         BIGINT      NOT NULL REFERENCES auth.identities(id) ON DELETE CASCADE,
    provider            VARCHAR(30) NOT NULL,
    provider_user_id    VARCHAR(255) NULL,
    provider_email      VARCHAR(255) NULL,
    password_hash       VARCHAR(255) NULL,
    access_token        TEXT         NULL,
    refresh_token       TEXT         NULL,
    token_expires_at    TIMESTAMPTZ  NULL,
    provider_data       JSONB        NULL,
    is_primary          BOOLEAN      NOT NULL DEFAULT FALSE,
    password_changed_at TIMESTAMPTZ  NULL,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_user_id),
    CONSTRAINT chk_local_password CHECK (
        (provider = 'local' AND password_hash IS NOT NULL) OR
        (provider <> 'local' AND password_hash IS NULL)
    )
);

CREATE INDEX idx_auth_providers_identity ON auth.providers(identity_id);
CREATE INDEX idx_auth_providers_type     ON auth.providers(provider);
CREATE INDEX idx_auth_providers_primary  ON auth.providers(identity_id, is_primary) WHERE is_primary = TRUE;

CREATE TRIGGER trg_auth_providers_updated_at
    BEFORE UPDATE ON auth.providers
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

-- ---------------------------------------------------------------------------
-- auth.verification_tokens
-- ---------------------------------------------------------------------------
CREATE TABLE auth.verification_tokens (
    id          BIGSERIAL   PRIMARY KEY,
    identity_id BIGINT      NOT NULL REFERENCES auth.identities(id) ON DELETE CASCADE,
    token_type  VARCHAR(30) NOT NULL,
    token       VARCHAR(255) NOT NULL,
    code        VARCHAR(10)  NULL,
    expires_at  TIMESTAMPTZ  NOT NULL,
    used_at     TIMESTAMPTZ  NULL,
    attempts    INT          NOT NULL DEFAULT 0,
    metadata    JSONB        NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_auth_vtokens_identity ON auth.verification_tokens(identity_id);
CREATE INDEX idx_auth_vtokens_token    ON auth.verification_tokens(token) WHERE used_at IS NULL;

-- ---------------------------------------------------------------------------
-- auth.roles / permissions / RBAC
-- ---------------------------------------------------------------------------
CREATE TABLE auth.roles (
    id           BIGSERIAL    PRIMARY KEY,
    name         VARCHAR(50)  NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    description  TEXT,
    is_system    BOOLEAN      NOT NULL DEFAULT FALSE,
    is_active    BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_auth_roles_updated_at
    BEFORE UPDATE ON auth.roles
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE auth.permissions (
    id           BIGSERIAL    PRIMARY KEY,
    name         VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(150) NOT NULL,
    resource     VARCHAR(50)  NULL,
    action       VARCHAR(50)  NULL,
    is_active    BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE auth.role_permissions (
    role_id       BIGINT NOT NULL REFERENCES auth.roles(id)       ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES auth.permissions(id) ON DELETE CASCADE,
    granted_by    BIGINT NULL    REFERENCES auth.identities(id)   ON DELETE SET NULL,
    granted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE auth.identity_roles (
    identity_id BIGINT      NOT NULL REFERENCES auth.identities(id) ON DELETE CASCADE,
    role_id     BIGINT      NOT NULL REFERENCES auth.roles(id)      ON DELETE CASCADE,
    assigned_by BIGINT      NULL     REFERENCES auth.identities(id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NULL,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    PRIMARY KEY (identity_id, role_id)
);

CREATE INDEX idx_auth_identity_roles_identity ON auth.identity_roles(identity_id) WHERE is_active = TRUE;

-- ---------------------------------------------------------------------------
-- auth.sessions
-- ---------------------------------------------------------------------------
CREATE TABLE auth.sessions (
    id                 BIGSERIAL    PRIMARY KEY,
    identity_id        BIGINT       NOT NULL REFERENCES auth.identities(id) ON DELETE CASCADE,
    session_token      VARCHAR(500) NOT NULL UNIQUE,
    refresh_token      VARCHAR(500) NULL UNIQUE,
    provider           VARCHAR(30)  NOT NULL,
    ip_address         INET         NULL,
    user_agent         TEXT         NULL,
    device_id          VARCHAR(255) NULL,
    device_fingerprint VARCHAR(500) NULL,
    status             VARCHAR(20)  NOT NULL DEFAULT 'active'
                           CHECK (status IN ('active','expired','revoked')),
    login_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_activity_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at         TIMESTAMPTZ  NOT NULL,
    logout_at          TIMESTAMPTZ  NULL
);

CREATE INDEX idx_auth_sessions_identity ON auth.sessions(identity_id) WHERE status = 'active';
CREATE INDEX idx_auth_sessions_token    ON auth.sessions(session_token) WHERE status = 'active';
CREATE INDEX idx_auth_sessions_expires  ON auth.sessions(status, expires_at);

-- ---------------------------------------------------------------------------
-- auth.audit_log — TimescaleDB hypertable, append-only
-- ---------------------------------------------------------------------------
CREATE TABLE auth.audit_log (
    id            BIGSERIAL   NOT NULL,
    identity_id   BIGINT      NULL,
    action        VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50)  NULL,
    resource_id   BIGINT       NULL,
    old_values    JSONB        NULL,
    new_values    JSONB        NULL,
    ip_address    INET         NULL,
    user_agent    TEXT         NULL,
    status        VARCHAR(20)  NULL,
    error_msg     TEXT         NULL,
    metadata      JSONB        NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
);

SELECT create_hypertable('auth.audit_log', 'created_at',
    chunk_time_interval => INTERVAL '1 month',
    if_not_exists => TRUE);

CREATE INDEX idx_auth_audit_identity ON auth.audit_log(identity_id, created_at DESC);
CREATE INDEX idx_auth_audit_action   ON auth.audit_log(action, created_at DESC);
CREATE INDEX idx_auth_audit_resource ON auth.audit_log(resource_type, resource_id);

ALTER TABLE auth.audit_log SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'resource_type',
    timescaledb.compress_orderby   = 'created_at DESC'
);
SELECT add_compression_policy('auth.audit_log', INTERVAL '90 days', if_not_exists => TRUE);

-- ---------------------------------------------------------------------------
-- auth.user_addresses
-- ---------------------------------------------------------------------------
CREATE TABLE auth.user_addresses (
    id             BIGSERIAL    PRIMARY KEY,
    identity_id    BIGINT       NOT NULL REFERENCES auth.identities(id) ON DELETE CASCADE,
    full_name      VARCHAR(255) NOT NULL,
    phone_number   VARCHAR(30)  NOT NULL,
    email          VARCHAR(255) NULL,
    country        VARCHAR(100) NOT NULL,
    county         VARCHAR(100) NULL,
    city           VARCHAR(100) NOT NULL,
    area           VARCHAR(255) NULL,
    postal_code    VARCHAR(20)  NULL,
    address_line_1 VARCHAR(255) NOT NULL,
    address_line_2 VARCHAR(255) NULL,
    is_default     BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_auth_addresses_identity ON auth.user_addresses(identity_id);

-- ---------------------------------------------------------------------------
-- auth.notifications
-- ---------------------------------------------------------------------------
CREATE TABLE auth.notifications (
    id          BIGSERIAL   PRIMARY KEY,
    identity_id BIGINT      NOT NULL REFERENCES auth.identities(id) ON DELETE CASCADE,
    title       VARCHAR(255) NOT NULL,
    message     TEXT         NOT NULL,
    type        VARCHAR(30)  NOT NULL DEFAULT 'system',
    channel     VARCHAR(20)  NOT NULL DEFAULT 'in_app',
    entity_type VARCHAR(50)  NULL,
    entity_id   BIGINT       NULL,
    is_read     BOOLEAN      NOT NULL DEFAULT FALSE,
    is_actioned BOOLEAN      NOT NULL DEFAULT FALSE,
    metadata    JSONB        NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    read_at     TIMESTAMPTZ  NULL,
    expires_at  TIMESTAMPTZ  NULL
);

CREATE INDEX idx_auth_notifications_unread ON auth.notifications(identity_id, is_read, created_at DESC);
CREATE INDEX idx_auth_notifications_entity ON auth.notifications(entity_type, entity_id) WHERE entity_type IS NOT NULL;

-- ---------------------------------------------------------------------------
-- device_fingerprints — fraud prevention (auth-scoped)
-- ---------------------------------------------------------------------------
CREATE TABLE auth.device_fingerprints (
    id                  BIGSERIAL    PRIMARY KEY,
    identity_id         BIGINT       NOT NULL REFERENCES auth.identities(id) ON DELETE CASCADE,
    fingerprint_hash    VARCHAR(255) NOT NULL,
    device_type         VARCHAR(20)  NULL,
    user_agent          TEXT         NULL,
    ip_address          INET         NULL,
    country_code        VARCHAR(5)   NULL,
    trust_status        VARCHAR(20)  NOT NULL DEFAULT 'trusted'
                            CHECK (trust_status IN ('trusted','suspicious','blocked')),
    first_seen_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_seen_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    seen_count          INT          NOT NULL DEFAULT 1,
    flagged_at          TIMESTAMPTZ  NULL,
    flag_reason         TEXT         NULL
);

CREATE UNIQUE INDEX idx_auth_device_fingerprints_identity
    ON auth.device_fingerprints(identity_id, fingerprint_hash);
CREATE INDEX idx_auth_device_fingerprints_hash
    ON auth.device_fingerprints(fingerprint_hash);
CREATE INDEX idx_auth_device_fingerprints_suspicious
    ON auth.device_fingerprints(trust_status) WHERE trust_status IN ('suspicious','blocked');

-- ---------------------------------------------------------------------------
-- auth: seed data
-- ---------------------------------------------------------------------------
INSERT INTO auth.roles (name, display_name, description, is_system) VALUES
    ('super_admin', 'Super Administrator', 'Full platform access',   TRUE),
    ('admin',       'Administrator',       'Administrative access',  TRUE),
    ('user',        'User',                'Standard buyer',         TRUE),
    ('seller',      'Seller',             'Marketplace seller',      TRUE)
ON CONFLICT (name) DO NOTHING;

INSERT INTO auth.permissions (name, display_name, resource, action) VALUES
    ('users.read',      'View Users',       'users',   'read'),
    ('users.manage',    'Manage Users',     'users',   'manage'),
    ('stores.manage',   'Manage Stores',    'stores',  'manage'),
    ('orders.manage',   'Manage Orders',    'orders',  'manage'),
    ('payouts.process', 'Process Payouts',  'payouts', 'process'),
    ('reports.view',    'View Reports',     'reports', 'read')
ON CONFLICT (name) DO NOTHING;

COMMIT;

-- =============================================================================
-- =============================================================================
-- SCHEMA: catalog
-- All marketplace domain tables
-- =============================================================================
-- =============================================================================

BEGIN;

-- ---------------------------------------------------------------------------
-- catalog.subscription_plans
-- Defined early — referenced by catalog.stores
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.subscription_plans (
    id                       BIGSERIAL    PRIMARY KEY,
    name                     VARCHAR(100) NOT NULL UNIQUE,
    slug                     VARCHAR(100) NOT NULL UNIQUE,
    description              TEXT,
    monthly_price            BIGINT       NOT NULL DEFAULT 0,
    annual_price             BIGINT       NULL,
    currency                 VARCHAR(10)  NOT NULL DEFAULT 'KES',
    max_products             INT          NOT NULL DEFAULT 50,
    max_staff                INT          NOT NULL DEFAULT 1,
    commission_discount_bps  INT          NOT NULL DEFAULT 0,
    analytics_access         BOOLEAN      NOT NULL DEFAULT FALSE,
    featured_slots           INT          NOT NULL DEFAULT 0,
    ad_credits               BIGINT       NOT NULL DEFAULT 0,
    priority_support         BOOLEAN      NOT NULL DEFAULT FALSE,
    api_access               BOOLEAN      NOT NULL DEFAULT FALSE,
    features_json            JSONB,
    is_active                BOOLEAN      NOT NULL DEFAULT TRUE,
    sort_order               INT          NOT NULL DEFAULT 0,
    -- soft delete
    deleted_at               TIMESTAMPTZ  NULL,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_catalog_sub_plans_updated_at
    BEFORE UPDATE ON catalog.subscription_plans
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

-- ---------------------------------------------------------------------------
-- catalog.stores
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.stores (
    id                   BIGSERIAL   PRIMARY KEY,
    owner_id             BIGINT      NOT NULL REFERENCES auth.identities(id) ON DELETE RESTRICT,
    name                 VARCHAR(255) NOT NULL,
    slug                 VARCHAR(255) NOT NULL UNIQUE,
    description          TEXT,
    logo_url             VARCHAR(500),
    banner_url           VARCHAR(500),
    status               VARCHAR(20)  NOT NULL DEFAULT 'pending'
                             CHECK (status IN ('pending','active','suspended','rejected','closed')),
    verification_status  VARCHAR(20)  NOT NULL DEFAULT 'unverified'
                             CHECK (verification_status IN ('unverified','pending_review','verified','rejected')),
    country              VARCHAR(100) NOT NULL DEFAULT 'KE',
    currency             VARCHAR(10)  NOT NULL DEFAULT 'KES',
    support_email        VARCHAR(255) NULL,
    support_phone        VARCHAR(30)  NULL,
    website_url          VARCHAR(500) NULL,
    total_sales          BIGINT       NOT NULL DEFAULT 0,
    total_orders         INT          NOT NULL DEFAULT 0,
    rating               DECIMAL(3,2) NOT NULL DEFAULT 0,
    review_count         INT          NOT NULL DEFAULT 0,
    subscription_plan_id BIGINT       NULL REFERENCES catalog.subscription_plans(id) ON DELETE SET NULL,
    verified_at          TIMESTAMPTZ  NULL,
    suspended_at         TIMESTAMPTZ  NULL,
    suspension_reason    TEXT         NULL,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ  NULL
);

CREATE INDEX idx_catalog_stores_owner        ON catalog.stores(owner_id);
CREATE INDEX idx_catalog_stores_status       ON catalog.stores(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_catalog_stores_verification ON catalog.stores(verification_status);

CREATE TRIGGER trg_catalog_stores_updated_at
    BEFORE UPDATE ON catalog.stores
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

-- ---------------------------------------------------------------------------
-- catalog.store_settings
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.store_settings (
    store_id                 BIGINT   PRIMARY KEY REFERENCES catalog.stores(id) ON DELETE CASCADE,
    auto_accept_orders       BOOLEAN  NOT NULL DEFAULT TRUE,
    dispatch_days            INT      NOT NULL DEFAULT 3,
    return_policy_days       INT      NOT NULL DEFAULT 14,
    return_policy_text       TEXT,
    processing_time_min      INT      NOT NULL DEFAULT 1,
    processing_time_max      INT      NOT NULL DEFAULT 5,
    vacation_mode            BOOLEAN  NOT NULL DEFAULT FALSE,
    vacation_message         TEXT,
    custom_policies          JSONB,
    notification_preferences JSONB,
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- catalog.store_financial_settings (from extensions)
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.store_financial_settings (
    store_id                  BIGINT    PRIMARY KEY REFERENCES catalog.stores(id) ON DELETE CASCADE,
    payout_schedule           VARCHAR(20) NOT NULL DEFAULT 'weekly',
    payout_day_of_week        SMALLINT  NULL CHECK (payout_day_of_week BETWEEN 0 AND 6),
    min_payout_amount         BIGINT    NOT NULL DEFAULT 0,
    reserve_percentage_bps    INT       NOT NULL DEFAULT 0,
    dispute_hold_days         INT       NOT NULL DEFAULT 7,
    settlement_currency       VARCHAR(10) NOT NULL DEFAULT 'KES',
    auto_payout_enabled       BOOLEAN   NOT NULL DEFAULT TRUE,
    default_payout_account_id BIGINT    NULL,  -- FK to accounting.payout_accounts added after that table
    vat_inclusive_pricing     BOOLEAN   NOT NULL DEFAULT FALSE,
    withholding_tax_rate_bps  INT       NOT NULL DEFAULT 0,
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_catalog_store_financial_settings_updated_at
    BEFORE UPDATE ON catalog.store_financial_settings
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

-- ---------------------------------------------------------------------------
-- catalog.store_permissions / store_staff
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.store_permissions (
    id          BIGSERIAL    PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,
    code        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT
);

CREATE TABLE catalog.store_staff (
    id          BIGSERIAL   PRIMARY KEY,
    store_id    BIGINT      NOT NULL REFERENCES catalog.stores(id) ON DELETE CASCADE,
    identity_id BIGINT      NOT NULL REFERENCES auth.identities(id) ON DELETE CASCADE,
    role        VARCHAR(30) NOT NULL DEFAULT 'staff',
    invited_by  BIGINT      NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    invited_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    accepted_at TIMESTAMPTZ NULL,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    UNIQUE (store_id, identity_id)
);

CREATE INDEX idx_catalog_store_staff_store    ON catalog.store_staff(store_id) WHERE is_active = TRUE;
CREATE INDEX idx_catalog_store_staff_identity ON catalog.store_staff(identity_id);

CREATE TABLE catalog.store_roles_permissions (
    role          VARCHAR(30) NOT NULL,
    permission_id BIGINT      NOT NULL REFERENCES catalog.store_permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role, permission_id)
);

CREATE TABLE catalog.store_staff_permissions (
    staff_id      BIGINT NOT NULL REFERENCES catalog.store_staff(id)       ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES catalog.store_permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (staff_id, permission_id)
);

-- ---------------------------------------------------------------------------
-- catalog.store_documents
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.store_documents (
    id            BIGSERIAL   PRIMARY KEY,
    store_id      BIGINT      NOT NULL REFERENCES catalog.stores(id)       ON DELETE CASCADE,
    document_type VARCHAR(50) NOT NULL,
    document_url  JSONB       NOT NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending','verified','rejected')),
    rejection_note TEXT,
    uploaded_by   BIGINT      NOT NULL REFERENCES auth.identities(id) ON DELETE RESTRICT,
    reviewed_by   BIGINT      NULL     REFERENCES auth.identities(id) ON DELETE SET NULL,
    reviewed_at   TIMESTAMPTZ NULL,
    expires_at    TIMESTAMPTZ NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_catalog_store_docs_store  ON catalog.store_documents(store_id);
CREATE INDEX idx_catalog_store_docs_status ON catalog.store_documents(status);

-- ---------------------------------------------------------------------------
-- catalog.store_suspension_history
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.store_suspension_history (
    id           BIGSERIAL   PRIMARY KEY,
    store_id     BIGINT      NOT NULL REFERENCES catalog.stores(id)       ON DELETE CASCADE,
    action       VARCHAR(20) NOT NULL,
    reason       TEXT        NOT NULL,
    performed_by BIGINT      NOT NULL REFERENCES auth.identities(id) ON DELETE RESTRICT,
    notes        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_catalog_store_suspension ON catalog.store_suspension_history(store_id, created_at DESC);

-- ---------------------------------------------------------------------------
-- catalog.risk_flags / seller_strikes / account_restrictions (from extensions)
-- Placed here (catalog) since they relate to stores and users
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.risk_flags (
    id              BIGSERIAL   PRIMARY KEY,
    entity_type     VARCHAR(30) NOT NULL,
    entity_id       BIGINT      NOT NULL,
    flag_type       VARCHAR(50) NOT NULL,
    severity        VARCHAR(10) NOT NULL DEFAULT 'medium'
                        CHECK (severity IN ('low','medium','high','critical')),
    description     TEXT        NOT NULL,
    evidence        JSONB       NULL,
    status          VARCHAR(25) NOT NULL DEFAULT 'open'
                        CHECK (status IN ('open','under_review','confirmed_fraud','false_positive','resolved')),
    raised_by       VARCHAR(20) NOT NULL DEFAULT 'auto',
    raised_by_id    BIGINT      NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    reviewed_by     BIGINT      NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    reviewed_at     TIMESTAMPTZ NULL,
    resolved_at     TIMESTAMPTZ NULL,
    resolution_note TEXT        NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_catalog_risk_flags_entity ON catalog.risk_flags(entity_type, entity_id, created_at DESC);
CREATE INDEX idx_catalog_risk_flags_open   ON catalog.risk_flags(severity, status) WHERE status IN ('open','under_review');
CREATE INDEX idx_catalog_risk_flags_type   ON catalog.risk_flags(flag_type, created_at DESC);

CREATE TRIGGER trg_catalog_risk_flags_updated_at
    BEFORE UPDATE ON catalog.risk_flags
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE catalog.seller_strikes (
    id              BIGSERIAL   PRIMARY KEY,
    store_id        BIGINT      NOT NULL REFERENCES catalog.stores(id) ON DELETE CASCADE,
    risk_flag_id    BIGINT      NULL REFERENCES catalog.risk_flags(id) ON DELETE SET NULL,
    strike_reason   VARCHAR(50) NOT NULL,
    description     TEXT        NOT NULL,
    issued_by       BIGINT      NOT NULL REFERENCES auth.identities(id) ON DELETE RESTRICT,
    status          VARCHAR(20) NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active','appealed','overturned','expired')),
    expires_at      TIMESTAMPTZ NULL,
    overturned_at   TIMESTAMPTZ NULL,
    overturned_by   BIGINT      NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    overturn_reason TEXT        NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_catalog_seller_strikes_store  ON catalog.seller_strikes(store_id, status);
CREATE INDEX idx_catalog_seller_strikes_active ON catalog.seller_strikes(store_id) WHERE status = 'active';

CREATE TABLE catalog.account_restrictions (
    id               BIGSERIAL   PRIMARY KEY,
    entity_type      VARCHAR(20) NOT NULL,
    entity_id        BIGINT      NOT NULL,
    restriction_type VARCHAR(40) NOT NULL,
    reason           TEXT        NOT NULL,
    limit_value      BIGINT      NULL,
    imposed_by       BIGINT      NOT NULL REFERENCES auth.identities(id) ON DELETE RESTRICT,
    risk_flag_id     BIGINT      NULL REFERENCES catalog.risk_flags(id) ON DELETE SET NULL,
    status           VARCHAR(20) NOT NULL DEFAULT 'active'
                         CHECK (status IN ('active','lifted','expired')),
    expires_at       TIMESTAMPTZ NULL,
    lifted_at        TIMESTAMPTZ NULL,
    lifted_by        BIGINT      NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    lift_reason      TEXT        NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_catalog_account_restrictions_entity ON catalog.account_restrictions(entity_type, entity_id, status);
CREATE INDEX idx_catalog_account_restrictions_active ON catalog.account_restrictions(status) WHERE status = 'active';

-- ---------------------------------------------------------------------------
-- catalog.product_categories + closure trigger
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.product_categories (
    id         BIGSERIAL    PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    slug       VARCHAR(255) NOT NULL UNIQUE,
    logo_url   VARCHAR(500) NULL,
    parent_id  BIGINT       NULL REFERENCES catalog.product_categories(id),
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_catalog_categories_parent ON catalog.product_categories(parent_id);

CREATE TABLE catalog.category_closure (
    ancestor_id   BIGINT NOT NULL REFERENCES catalog.product_categories(id) ON DELETE CASCADE,
    descendant_id BIGINT NOT NULL REFERENCES catalog.product_categories(id) ON DELETE CASCADE,
    depth         INT    NOT NULL,
    PRIMARY KEY (ancestor_id, descendant_id)
);

CREATE INDEX idx_catalog_closure_desc     ON catalog.category_closure(descendant_id);
CREATE INDEX idx_catalog_closure_ancestor ON catalog.category_closure(ancestor_id, descendant_id);

CREATE OR REPLACE FUNCTION catalog.fn_maintain_category_closure()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO catalog.category_closure (ancestor_id, descendant_id, depth)
        VALUES (NEW.id, NEW.id, 0);
        IF NEW.parent_id IS NOT NULL THEN
            INSERT INTO catalog.category_closure (ancestor_id, descendant_id, depth)
            SELECT ancestor_id, NEW.id, depth + 1
            FROM catalog.category_closure WHERE descendant_id = NEW.parent_id;
        END IF;
        RETURN NEW;
    END IF;
    IF TG_OP = 'UPDATE' THEN
        IF OLD.parent_id IS NOT DISTINCT FROM NEW.parent_id THEN RETURN NEW; END IF;
        DELETE FROM catalog.category_closure
        WHERE descendant_id IN (SELECT descendant_id FROM catalog.category_closure WHERE ancestor_id = NEW.id)
          AND ancestor_id NOT IN (SELECT descendant_id FROM catalog.category_closure WHERE ancestor_id = NEW.id);
        IF NEW.parent_id IS NOT NULL THEN
            INSERT INTO catalog.category_closure (ancestor_id, descendant_id, depth)
            SELECT p.ancestor_id, c.descendant_id, p.depth + c.depth + 1
            FROM catalog.category_closure p CROSS JOIN catalog.category_closure c
            WHERE p.descendant_id = NEW.parent_id AND c.ancestor_id = NEW.id;
        END IF;
        RETURN NEW;
    END IF;
    IF TG_OP = 'DELETE' THEN
        DELETE FROM catalog.category_closure WHERE descendant_id = OLD.id OR ancestor_id = OLD.id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS trg_category_closure ON catalog.product_categories;
CREATE TRIGGER trg_category_closure
    AFTER INSERT OR UPDATE OF parent_id OR DELETE ON catalog.product_categories
    FOR EACH ROW EXECUTE FUNCTION catalog.fn_maintain_category_closure();

-- ---------------------------------------------------------------------------
-- catalog.tax_jurisdictions / tax_rates / store_tax_profiles (from extensions)
-- Placed in catalog before products so store_tax_profiles can FK to stores
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.tax_jurisdictions (
    id           BIGSERIAL    PRIMARY KEY,
    name         VARCHAR(150) NOT NULL,
    country_code VARCHAR(5)   NOT NULL,
    region_code  VARCHAR(20)  NULL,
    city_code    VARCHAR(50)  NULL,
    level        VARCHAR(20)  NOT NULL DEFAULT 'country'
                     CHECK (level IN ('country','region','city')),
    is_active    BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_catalog_tax_jurisdictions_geo
    ON catalog.tax_jurisdictions(country_code, COALESCE(region_code,''), COALESCE(city_code,''));

CREATE TABLE catalog.tax_rates (
    id              BIGSERIAL   PRIMARY KEY,
    jurisdiction_id BIGINT      NOT NULL REFERENCES catalog.tax_jurisdictions(id) ON DELETE RESTRICT,
    tax_type        VARCHAR(30) NOT NULL,
    applies_to      VARCHAR(30) NOT NULL DEFAULT 'all',
    rate_bps        INT         NOT NULL CHECK (rate_bps >= 0),
    is_compound     BOOLEAN     NOT NULL DEFAULT FALSE,
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    effective_from  DATE        NOT NULL,
    effective_to    DATE        NULL,
    authority_ref   VARCHAR(100) NULL,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_catalog_tax_rates_jurisdiction ON catalog.tax_rates(jurisdiction_id, is_active);
CREATE INDEX idx_catalog_tax_rates_lookup       ON catalog.tax_rates(tax_type, applies_to, effective_from) WHERE is_active = TRUE;

CREATE TABLE catalog.store_tax_profiles (
    store_id                     BIGINT   PRIMARY KEY REFERENCES catalog.stores(id) ON DELETE CASCADE,
    jurisdiction_id              BIGINT   NULL REFERENCES catalog.tax_jurisdictions(id) ON DELETE SET NULL,
    tax_id_number                VARCHAR(50)  NULL,
    tax_registration_name        VARCHAR(255) NULL,
    is_vat_registered            BOOLEAN  NOT NULL DEFAULT FALSE,
    price_display_mode           VARCHAR(20) NOT NULL DEFAULT 'exclusive',
    default_tax_rate_id          BIGINT   NULL REFERENCES catalog.tax_rates(id) ON DELETE SET NULL,
    withholding_tax_exempt       BOOLEAN  NOT NULL DEFAULT FALSE,
    digital_services_applicable  BOOLEAN  NOT NULL DEFAULT FALSE,
    created_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_catalog_store_tax_profiles_updated_at
    BEFORE UPDATE ON catalog.store_tax_profiles
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE catalog.invoice_sequences (
    store_id        BIGINT      PRIMARY KEY REFERENCES catalog.stores(id) ON DELETE CASCADE,
    prefix          VARCHAR(20) NOT NULL DEFAULT 'INV',
    last_number     BIGINT      NOT NULL DEFAULT 0,
    format_template VARCHAR(50) NOT NULL DEFAULT '{prefix}-{year}-{number:08}',
    fiscal_year     INT         NOT NULL DEFAULT EXTRACT(YEAR FROM NOW())::INT,
    reset_annually  BOOLEAN     NOT NULL DEFAULT TRUE,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_catalog_invoice_sequences_updated_at
    BEFORE UPDATE ON catalog.invoice_sequences
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

-- ---------------------------------------------------------------------------
-- catalog.product_brands / tags / payment_methods
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.product_brands (
    id         BIGSERIAL    PRIMARY KEY,
    name       VARCHAR(255) NOT NULL UNIQUE,
    slug       VARCHAR(255) NOT NULL UNIQUE,
    logo_url   VARCHAR(500) NULL,
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE catalog.product_brand_categories (
    brand_id    BIGINT NOT NULL REFERENCES catalog.product_brands(id)     ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES catalog.product_categories(id) ON DELETE CASCADE,
    PRIMARY KEY (brand_id, category_id)
);

CREATE TABLE catalog.tags (
    id   BIGSERIAL    PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE catalog.payment_methods (
    id           SERIAL       PRIMARY KEY,
    code         VARCHAR(50)  NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    provider     VARCHAR(50)  NOT NULL,
    type         VARCHAR(30)  NOT NULL,
    description  TEXT,
    icon_url     TEXT,
    is_active    BOOLEAN      NOT NULL DEFAULT TRUE,
    metadata     JSONB,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_catalog_payment_methods_active ON catalog.payment_methods(is_active, type);

CREATE TRIGGER trg_catalog_payment_methods_updated_at
    BEFORE UPDATE ON catalog.payment_methods
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

-- ---------------------------------------------------------------------------
-- catalog.products
-- Includes deleted_at (from extensions) directly in definition
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.products (
    id                  BIGSERIAL    PRIMARY KEY,
    store_id            BIGINT       NULL REFERENCES catalog.stores(id) ON DELETE RESTRICT,
    is_platform_product BOOLEAN      NOT NULL DEFAULT TRUE,
    name                VARCHAR(255) NOT NULL,
    slug                VARCHAR(255) NOT NULL UNIQUE,
    description         TEXT,
    short_description   TEXT,
    brand_id            BIGINT       NULL REFERENCES catalog.product_brands(id),
    base_price          BIGINT       NOT NULL,
    status              VARCHAR(20)  NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active','draft','archived')),
    is_featured         BOOLEAN      NOT NULL DEFAULT FALSE,
    is_digital          BOOLEAN      NOT NULL DEFAULT FALSE,
    rating              DECIMAL(3,2) NOT NULL DEFAULT 0,
    review_count        INT          NOT NULL DEFAULT 0,
    approval_status     VARCHAR(20)  NOT NULL DEFAULT 'approved'
                            CHECK (approval_status IN ('draft','pending_review','approved','rejected','suspended')),
    approved_by         BIGINT       NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    approved_at         TIMESTAMPTZ  NULL,
    rejection_reason    TEXT         NULL,
    created_by          BIGINT       NULL REFERENCES auth.identities(id),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ  NULL,   -- soft delete
    CONSTRAINT chk_product_ownership CHECK (
        (is_platform_product = TRUE  AND store_id IS NULL) OR
        (is_platform_product = FALSE AND store_id IS NOT NULL)
    )
);

CREATE INDEX idx_catalog_products_status      ON catalog.products(status);
CREATE INDEX idx_catalog_products_store       ON catalog.products(store_id, approval_status);
CREATE INDEX idx_catalog_products_approval    ON catalog.products(approval_status)
    WHERE approval_status IN ('pending_review','rejected');
CREATE INDEX idx_catalog_products_featured    ON catalog.products(is_featured) WHERE is_featured = TRUE;
CREATE INDEX idx_catalog_products_status_upd  ON catalog.products(status, updated_at DESC) WHERE status = 'active';
CREATE INDEX idx_catalog_products_deleted     ON catalog.products(deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TRIGGER trg_catalog_products_updated_at
    BEFORE UPDATE ON catalog.products
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE catalog.product_configs (
    product_id BIGINT NOT NULL UNIQUE REFERENCES catalog.products(id) ON DELETE CASCADE,
    config     JSONB  NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE catalog.product_category_map (
    product_id  BIGINT NOT NULL REFERENCES catalog.products(id)           ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES catalog.product_categories(id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, category_id)
);
CREATE INDEX idx_catalog_pcm_category ON catalog.product_category_map(category_id);

CREATE TABLE catalog.product_tags (
    product_id BIGINT NOT NULL REFERENCES catalog.products(id) ON DELETE CASCADE,
    tag_id     BIGINT NOT NULL REFERENCES catalog.tags(id)     ON DELETE CASCADE,
    PRIMARY KEY (product_id, tag_id)
);
CREATE INDEX idx_catalog_product_tags ON catalog.product_tags(tag_id, product_id);

CREATE TABLE catalog.product_images (
    id         BIGSERIAL    PRIMARY KEY,
    product_id BIGINT       NOT NULL REFERENCES catalog.products(id) ON DELETE CASCADE,
    image_url  VARCHAR(500) NOT NULL,
    is_primary BOOLEAN      NOT NULL DEFAULT FALSE,
    sort_order INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_catalog_product_images ON catalog.product_images(product_id, is_primary DESC, sort_order ASC);

-- ---------------------------------------------------------------------------
-- catalog.attributes / variants
-- product_variants includes deleted_at (from extensions)
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.attributes (
    id                   BIGSERIAL    PRIMARY KEY,
    name                 VARCHAR(100) NOT NULL,
    slug                 VARCHAR(100) NOT NULL UNIQUE,
    is_variant_dimension BOOLEAN      NOT NULL DEFAULT FALSE
);

CREATE TABLE catalog.attribute_values (
    id           BIGSERIAL    PRIMARY KEY,
    attribute_id BIGINT       NOT NULL REFERENCES catalog.attributes(id) ON DELETE CASCADE,
    value        VARCHAR(100) NOT NULL,
    UNIQUE (attribute_id, value)
);

CREATE TABLE catalog.product_attribute_values (
    product_id         BIGINT NOT NULL REFERENCES catalog.products(id)         ON DELETE CASCADE,
    attribute_value_id BIGINT NOT NULL REFERENCES catalog.attribute_values(id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, attribute_value_id)
);

CREATE TABLE catalog.product_variants (
    id         BIGSERIAL    PRIMARY KEY,
    product_id BIGINT       NOT NULL REFERENCES catalog.products(id) ON DELETE CASCADE,
    sku        VARCHAR(100) NOT NULL UNIQUE,
    price      BIGINT       NOT NULL,
    weight     DECIMAL(10,2) NULL,
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ  NULL    -- soft delete
);

CREATE INDEX idx_catalog_variants_product ON catalog.product_variants(product_id, is_active);

CREATE TABLE catalog.variant_images (
    id         BIGSERIAL    PRIMARY KEY,
    variant_id BIGINT       NOT NULL REFERENCES catalog.product_variants(id) ON DELETE CASCADE,
    image_url  VARCHAR(500) NOT NULL,
    is_primary BOOLEAN      NOT NULL DEFAULT FALSE,
    sort_order INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE catalog.variant_attribute_values (
    variant_id         BIGINT NOT NULL REFERENCES catalog.product_variants(id) ON DELETE CASCADE,
    attribute_value_id BIGINT NOT NULL REFERENCES catalog.attribute_values(id),
    PRIMARY KEY (variant_id, attribute_value_id)
);
CREATE INDEX idx_catalog_vav ON catalog.variant_attribute_values(attribute_value_id, variant_id);

-- ---------------------------------------------------------------------------
-- catalog.inventory_locations / inventory_items
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.inventory_locations (
    id            BIGSERIAL    PRIMARY KEY,
    store_id      BIGINT       NULL REFERENCES catalog.stores(id) ON DELETE SET NULL,
    name          VARCHAR(150) NOT NULL,
    location_code VARCHAR(50)  NULL UNIQUE,
    country       VARCHAR(100) NULL,
    city          VARCHAR(100) NULL,
    address       TEXT         NULL,
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE catalog.inventory_items (
    id            BIGSERIAL PRIMARY KEY,
    variant_id    BIGINT    NOT NULL REFERENCES catalog.product_variants(id)    ON DELETE CASCADE,
    location_id   BIGINT    NOT NULL REFERENCES catalog.inventory_locations(id) ON DELETE CASCADE,
    available_qty INT       NOT NULL DEFAULT 0,
    reserved_qty  INT       NOT NULL DEFAULT 0,
    incoming_qty  INT       NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (variant_id, location_id)
);

CREATE INDEX idx_catalog_inventory_variant  ON catalog.inventory_items(variant_id, available_qty, reserved_qty);
CREATE INDEX idx_catalog_inventory_location ON catalog.inventory_items(location_id);

-- ---------------------------------------------------------------------------
-- catalog.inventory_reservations / inventory_movements / stock_adjustments
-- (from extensions — placed here after inventory tables they depend on)
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.inventory_reservations (
    id                  BIGSERIAL   PRIMARY KEY,
    status              VARCHAR(20) NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending','confirmed','expired','released','cancelled')),
    variant_id          BIGINT      NOT NULL REFERENCES catalog.product_variants(id)    ON DELETE RESTRICT,
    location_id         BIGINT      NOT NULL REFERENCES catalog.inventory_locations(id) ON DELETE RESTRICT,
    quantity            INT         NOT NULL CHECK (quantity > 0),
    reference_type      VARCHAR(30) NOT NULL DEFAULT 'cart_item',
    reference_id        BIGINT      NOT NULL,
    held_by_identity_id BIGINT      NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    session_id          VARCHAR(255) NULL,
    expires_at          TIMESTAMPTZ NOT NULL,
    confirmed_at        TIMESTAMPTZ NULL,
    released_at         TIMESTAMPTZ NULL,
    release_reason      TEXT        NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inv_res_variant   ON catalog.inventory_reservations(variant_id, status);
CREATE INDEX idx_inv_res_reference ON catalog.inventory_reservations(reference_type, reference_id);
CREATE INDEX idx_inv_res_identity  ON catalog.inventory_reservations(held_by_identity_id) WHERE held_by_identity_id IS NOT NULL;
CREATE INDEX idx_inv_res_expires   ON catalog.inventory_reservations(expires_at) WHERE status = 'pending';
CREATE INDEX idx_inv_res_session   ON catalog.inventory_reservations(session_id) WHERE session_id IS NOT NULL;

CREATE TRIGGER trg_catalog_inv_reservations_updated_at
    BEFORE UPDATE ON catalog.inventory_reservations
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE catalog.inventory_movements (
    id             BIGSERIAL   PRIMARY KEY,
    variant_id     BIGINT      NOT NULL REFERENCES catalog.product_variants(id)    ON DELETE RESTRICT,
    location_id    BIGINT      NOT NULL REFERENCES catalog.inventory_locations(id) ON DELETE RESTRICT,
    movement_type  VARCHAR(30) NOT NULL,
    qty_delta      INT         NOT NULL,
    qty_before     INT         NOT NULL,
    qty_after      INT         NOT NULL,
    field_affected VARCHAR(20) NOT NULL DEFAULT 'available_qty'
                       CHECK (field_affected IN ('available_qty','reserved_qty','incoming_qty')),
    reference_type VARCHAR(50) NULL,
    reference_id   BIGINT      NULL,
    performed_by   BIGINT      NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    notes          TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inv_movements_variant   ON catalog.inventory_movements(variant_id, created_at DESC);
CREATE INDEX idx_inv_movements_location  ON catalog.inventory_movements(location_id, created_at DESC);
CREATE INDEX idx_inv_movements_reference ON catalog.inventory_movements(reference_type, reference_id);
CREATE INDEX idx_inv_movements_type      ON catalog.inventory_movements(movement_type, created_at DESC);

CREATE OR REPLACE RULE inv_movements_no_update AS ON UPDATE TO catalog.inventory_movements DO INSTEAD NOTHING;
CREATE OR REPLACE RULE inv_movements_no_delete AS ON DELETE TO catalog.inventory_movements DO INSTEAD NOTHING;

CREATE TABLE catalog.stock_adjustments (
    id              BIGSERIAL   PRIMARY KEY,
    store_id        BIGINT      NULL REFERENCES catalog.stores(id) ON DELETE SET NULL,
    variant_id      BIGINT      NOT NULL REFERENCES catalog.product_variants(id)    ON DELETE RESTRICT,
    location_id     BIGINT      NOT NULL REFERENCES catalog.inventory_locations(id) ON DELETE RESTRICT,
    adjustment_type VARCHAR(30) NOT NULL,
    qty_change      INT         NOT NULL,
    qty_before      INT         NOT NULL,
    qty_after       INT         NOT NULL,
    reason          TEXT        NOT NULL,
    reference_doc   VARCHAR(255) NULL,
    performed_by    BIGINT      NOT NULL REFERENCES auth.identities(id) ON DELETE RESTRICT,
    approved_by     BIGINT      NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    approved_at     TIMESTAMPTZ NULL,
    movement_id     BIGINT      NULL REFERENCES catalog.inventory_movements(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stock_adjustments_variant   ON catalog.stock_adjustments(variant_id, created_at DESC);
CREATE INDEX idx_stock_adjustments_store     ON catalog.stock_adjustments(store_id);
CREATE INDEX idx_stock_adjustments_performer ON catalog.stock_adjustments(performed_by);

-- ---------------------------------------------------------------------------
-- catalog.discounts / carts
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.discounts (
    id               BIGSERIAL    PRIMARY KEY,
    name             VARCHAR(255) NOT NULL,
    code             VARCHAR(50)  NULL,
    discount_type    VARCHAR(20)  NOT NULL CHECK (discount_type IN ('percentage','fixed')),
    value            BIGINT       NOT NULL,
    min_order_amount BIGINT       NULL,
    max_redemptions  INT          NULL,
    starts_at        TIMESTAMPTZ  NULL,
    ends_at          TIMESTAMPTZ  NULL,
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_catalog_discounts_code ON catalog.discounts(code) WHERE code IS NOT NULL;

CREATE TABLE catalog.discount_targets (
    discount_id BIGINT      NOT NULL REFERENCES catalog.discounts(id) ON DELETE CASCADE,
    target_type VARCHAR(20) NOT NULL,
    target_id   BIGINT      NOT NULL,
    PRIMARY KEY (discount_id, target_type, target_id)
);

CREATE TABLE catalog.discount_redemptions (
    id          BIGSERIAL   PRIMARY KEY,
    discount_id BIGINT      NOT NULL REFERENCES catalog.discounts(id) ON DELETE CASCADE,
    order_id    BIGINT      NOT NULL,
    user_id     BIGINT      NULL REFERENCES auth.identities(id),
    redeemed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (discount_id, order_id)
);

CREATE TABLE catalog.carts (
    id         BIGSERIAL    PRIMARY KEY,
    user_id    BIGINT       NULL REFERENCES auth.identities(id) ON DELETE CASCADE,
    session_id VARCHAR(255) NULL,
    status     VARCHAR(20)  NOT NULL DEFAULT 'active'
                   CHECK (status IN ('active','converted','abandoned')),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_catalog_carts_user ON catalog.carts(user_id, status);
CREATE UNIQUE INDEX ux_catalog_carts_user_active ON catalog.carts(user_id) WHERE status = 'active';

CREATE TABLE catalog.cart_items (
    id             BIGSERIAL PRIMARY KEY,
    cart_id        BIGINT    NOT NULL REFERENCES catalog.carts(id) ON DELETE CASCADE,
    product_id     BIGINT    NOT NULL REFERENCES catalog.products(id),
    variant_id     BIGINT    NOT NULL REFERENCES catalog.product_variants(id),
    quantity       INT       NOT NULL CHECK (quantity > 0),
    price_at_added BIGINT    NOT NULL,
    added_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (cart_id, variant_id)
);
CREATE INDEX idx_catalog_cart_items ON catalog.cart_items(cart_id);

-- ---------------------------------------------------------------------------
-- catalog.shipping
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.shipping_methods (
    id          BIGSERIAL    PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    description TEXT,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE catalog.shipping_zones (
    id        BIGSERIAL    PRIMARY KEY,
    name      VARCHAR(150) NOT NULL UNIQUE,
    is_active BOOLEAN      NOT NULL DEFAULT TRUE
);

CREATE TABLE catalog.shipping_zone_countries (
    zone_id BIGINT       NOT NULL REFERENCES catalog.shipping_zones(id) ON DELETE CASCADE,
    country VARCHAR(100) NOT NULL,
    PRIMARY KEY (zone_id, country)
);

CREATE TABLE catalog.shipping_method_rates (
    id                 BIGSERIAL PRIMARY KEY,
    shipping_method_id BIGINT    NOT NULL REFERENCES catalog.shipping_methods(id) ON DELETE CASCADE,
    zone_id            BIGINT    NOT NULL REFERENCES catalog.shipping_zones(id)   ON DELETE CASCADE,
    base_fee           BIGINT    NOT NULL,
    estimated_days_min INT       NULL,
    estimated_days_max INT       NULL,
    UNIQUE (shipping_method_id, zone_id)
);

-- ---------------------------------------------------------------------------
-- catalog.orders / seller_orders / order_items / order_payments
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.orders (
    id                      BIGSERIAL    PRIMARY KEY,
    user_id                 BIGINT       NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    cart_id                 BIGINT       NULL,
    order_number            VARCHAR(50)  NOT NULL UNIQUE,
    status                  VARCHAR(30)  NOT NULL DEFAULT 'pending',
    subtotal                BIGINT       NOT NULL,
    discount_amount         BIGINT       NOT NULL DEFAULT 0,
    tax_amount              BIGINT       NOT NULL DEFAULT 0,
    shipping_fee            BIGINT       NOT NULL DEFAULT 0,
    platform_fee            BIGINT       NOT NULL DEFAULT 0,
    total_amount            BIGINT       NOT NULL,
    currency                VARCHAR(10)  NOT NULL DEFAULT 'KES',
    shipping_method_id      BIGINT       NULL REFERENCES catalog.shipping_methods(id),
    shipping_full_name      VARCHAR(255) NOT NULL,
    shipping_phone          VARCHAR(30)  NOT NULL,
    shipping_email          VARCHAR(255) NULL,
    shipping_country        VARCHAR(100) NOT NULL,
    shipping_county         VARCHAR(100) NULL,
    shipping_city           VARCHAR(100) NOT NULL,
    shipping_area           VARCHAR(255) NULL,
    shipping_postal_code    VARCHAR(20)  NULL,
    shipping_address_line_1 VARCHAR(255) NOT NULL,
    shipping_address_line_2 VARCHAR(255) NULL,
    payment_intent_id       VARCHAR(255) NULL,
    payment_method_id       INT          NULL REFERENCES catalog.payment_methods(id),
    payment_captured_at     TIMESTAMPTZ  NULL,
    notes                   TEXT         NULL,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_catalog_orders_user    ON catalog.orders(user_id, created_at DESC);
CREATE INDEX idx_catalog_orders_status  ON catalog.orders(status);
CREATE INDEX idx_catalog_orders_payment ON catalog.orders(payment_intent_id) WHERE payment_intent_id IS NOT NULL;

CREATE TRIGGER trg_catalog_orders_updated_at
    BEFORE UPDATE ON catalog.orders
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

-- ---------------------------------------------------------------------------
-- catalog.payment_intents / payment_attempts (from extensions)
-- Placed here after catalog.orders which they reference
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.payment_intents (
    id                          BIGSERIAL    PRIMARY KEY,
    intent_ref                  VARCHAR(100) NOT NULL UNIQUE,
    order_id                    BIGINT       NOT NULL REFERENCES catalog.orders(id) ON DELETE RESTRICT,
    user_id                     BIGINT       NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    status                      VARCHAR(30)  NOT NULL DEFAULT 'created'
                                    CHECK (status IN ('created','processing','requires_action',
                                                      'succeeded','failed','cancelled','expired')),
    amount                      BIGINT       NOT NULL,
    amount_captured             BIGINT       NOT NULL DEFAULT 0,
    currency                    VARCHAR(10)  NOT NULL DEFAULT 'KES',
    payment_method_type         VARCHAR(30)  NOT NULL,
    provider_intent_id          VARCHAR(255) NULL,
    provider_response           JSONB        NULL,
    mpesa_phone                 VARCHAR(20)  NULL,
    mpesa_checkout_request_id   VARCHAR(100) NULL,
    failure_code                VARCHAR(50)  NULL,
    failure_message             TEXT         NULL,
    expires_at                  TIMESTAMPTZ  NULL,
    client_secret_hash          VARCHAR(255) NULL,
    metadata                    JSONB        NULL,
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pi_order          ON catalog.payment_intents(order_id);
CREATE INDEX idx_pi_user           ON catalog.payment_intents(user_id);
CREATE INDEX idx_pi_status         ON catalog.payment_intents(status, created_at DESC);
CREATE INDEX idx_pi_provider       ON catalog.payment_intents(provider_intent_id) WHERE provider_intent_id IS NOT NULL;
CREATE INDEX idx_pi_mpesa          ON catalog.payment_intents(mpesa_checkout_request_id) WHERE mpesa_checkout_request_id IS NOT NULL;
CREATE INDEX idx_pi_expires        ON catalog.payment_intents(expires_at) WHERE status IN ('created','processing','requires_action');

CREATE TRIGGER trg_catalog_payment_intents_updated_at
    BEFORE UPDATE ON catalog.payment_intents
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE catalog.payment_attempts (
    id                      BIGSERIAL    PRIMARY KEY,
    intent_id               BIGINT       NOT NULL REFERENCES catalog.payment_intents(id) ON DELETE CASCADE,
    attempt_number          INT          NOT NULL DEFAULT 1,
    status                  VARCHAR(20)  NOT NULL DEFAULT 'pending',
    amount                  BIGINT       NOT NULL,
    currency                VARCHAR(10)  NOT NULL DEFAULT 'KES',
    payment_method_type     VARCHAR(30)  NOT NULL,
    provider_transaction_id VARCHAR(255) NULL,
    provider_request        JSONB        NULL,
    provider_response       JSONB        NULL,
    provider_fee            BIGINT       NULL,
    failure_code            VARCHAR(50)  NULL,
    failure_message         TEXT         NULL,
    initiated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    completed_at            TIMESTAMPTZ  NULL
);

CREATE INDEX idx_pa_intent       ON catalog.payment_attempts(intent_id, attempt_number);
CREATE INDEX idx_pa_provider_txn ON catalog.payment_attempts(provider_transaction_id) WHERE provider_transaction_id IS NOT NULL;
CREATE INDEX idx_pa_status       ON catalog.payment_attempts(status, initiated_at DESC);

-- ---------------------------------------------------------------------------
-- catalog.seller_orders
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.seller_orders (
    id                      BIGSERIAL   PRIMARY KEY,
    order_id                BIGINT      NOT NULL REFERENCES catalog.orders(id)  ON DELETE RESTRICT,
    store_id                BIGINT      NOT NULL REFERENCES catalog.stores(id)  ON DELETE RESTRICT,
    order_number            VARCHAR(80) NOT NULL UNIQUE,
    status                  VARCHAR(20) NOT NULL DEFAULT 'pending',
    subtotal                BIGINT      NOT NULL,
    discount_amount         BIGINT      NOT NULL DEFAULT 0,
    tax_amount              BIGINT      NOT NULL DEFAULT 0,
    shipping_fee            BIGINT      NOT NULL DEFAULT 0,
    total_amount            BIGINT      NOT NULL,
    commission_amount       BIGINT      NOT NULL DEFAULT 0,
    seller_net_amount       BIGINT      NOT NULL DEFAULT 0,
    currency                VARCHAR(10) NOT NULL DEFAULT 'KES',
    settlement_status       VARCHAR(20) NOT NULL DEFAULT 'pending',
    settled_at              TIMESTAMPTZ NULL,
    settlement_id           BIGINT      NULL,
    shipping_method_id      BIGINT      NULL REFERENCES catalog.shipping_methods(id) ON DELETE SET NULL,
    shipping_full_name      VARCHAR(255) NULL,
    shipping_phone          VARCHAR(30)  NULL,
    shipping_email          VARCHAR(255) NULL,
    shipping_country        VARCHAR(100) NULL,
    shipping_city           VARCHAR(100) NULL,
    shipping_address_line_1 VARCHAR(255) NULL,
    shipping_address_line_2 VARCHAR(255) NULL,
    confirmed_at            TIMESTAMPTZ NULL,
    shipped_at              TIMESTAMPTZ NULL,
    delivered_at            TIMESTAMPTZ NULL,
    cancelled_at            TIMESTAMPTZ NULL,
    cancel_reason           TEXT        NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_catalog_seller_orders_order      ON catalog.seller_orders(order_id);
CREATE INDEX idx_catalog_seller_orders_store      ON catalog.seller_orders(store_id, created_at DESC);
CREATE INDEX idx_catalog_seller_orders_status     ON catalog.seller_orders(status);
CREATE INDEX idx_catalog_seller_orders_settlement ON catalog.seller_orders(settlement_status)
    WHERE settlement_status = 'pending';

CREATE TRIGGER trg_catalog_seller_orders_updated_at
    BEFORE UPDATE ON catalog.seller_orders
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

-- ---------------------------------------------------------------------------
-- catalog.order_items
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.order_items (
    id                BIGSERIAL PRIMARY KEY,
    order_id          BIGINT    NOT NULL REFERENCES catalog.orders(id)        ON DELETE CASCADE,
    seller_order_id   BIGINT    NULL     REFERENCES catalog.seller_orders(id) ON DELETE SET NULL,
    store_id          BIGINT    NULL     REFERENCES catalog.stores(id)         ON DELETE RESTRICT,
    product_id        BIGINT    NOT NULL REFERENCES catalog.products(id),
    variant_id        BIGINT    NULL     REFERENCES catalog.product_variants(id),
    product_name      VARCHAR(255) NOT NULL,
    product_slug      VARCHAR(255) NULL,
    variant_sku       VARCHAR(100) NULL,
    variant_name      VARCHAR(255) NULL,
    image_url         VARCHAR(500) NULL,
    unit_price        BIGINT    NOT NULL,
    quantity          INT       NOT NULL,
    discount_amount   BIGINT    NOT NULL DEFAULT 0,
    tax_rate_bps      INT       NOT NULL DEFAULT 0,
    total_price       BIGINT    NOT NULL,
    commission_bps    INT       NULL,
    commission_amount BIGINT    NULL,
    seller_amount     BIGINT    NULL,
    currency          VARCHAR(10) NOT NULL DEFAULT 'KES'
);

CREATE INDEX idx_catalog_order_items_order        ON catalog.order_items(order_id);
CREATE INDEX idx_catalog_order_items_seller_order ON catalog.order_items(seller_order_id);
CREATE INDEX idx_catalog_order_items_store        ON catalog.order_items(store_id);

-- ---------------------------------------------------------------------------
-- catalog.order_price_snapshots / order_item_price_snapshots (from extensions)
-- Immutable: insert-only via RULE
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.order_price_snapshots (
    id                      BIGSERIAL    PRIMARY KEY,
    order_id                BIGINT       NOT NULL UNIQUE REFERENCES catalog.orders(id) ON DELETE RESTRICT,
    subtotal                BIGINT       NOT NULL,
    discount_amount         BIGINT       NOT NULL DEFAULT 0,
    tax_amount              BIGINT       NOT NULL DEFAULT 0,
    shipping_fee            BIGINT       NOT NULL DEFAULT 0,
    platform_fee            BIGINT       NOT NULL DEFAULT 0,
    total_amount            BIGINT       NOT NULL,
    currency                VARCHAR(10)  NOT NULL,
    applied_discount_codes  TEXT[]       NULL,
    shipping_method_name    VARCHAR(100) NULL,
    snapshotted_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE RULE order_price_snapshot_no_update AS ON UPDATE TO catalog.order_price_snapshots DO INSTEAD NOTHING;
CREATE OR REPLACE RULE order_price_snapshot_no_delete AS ON DELETE TO catalog.order_price_snapshots DO INSTEAD NOTHING;

CREATE TABLE catalog.order_item_price_snapshots (
    id                BIGSERIAL    PRIMARY KEY,
    order_item_id     BIGINT       NOT NULL UNIQUE REFERENCES catalog.order_items(id) ON DELETE RESTRICT,
    order_id          BIGINT       NOT NULL REFERENCES catalog.orders(id) ON DELETE RESTRICT,
    product_id        BIGINT       NOT NULL,
    variant_id        BIGINT       NULL,
    product_name      VARCHAR(255) NOT NULL,
    variant_sku       VARCHAR(100) NULL,
    unit_price        BIGINT       NOT NULL,
    quantity          INT          NOT NULL,
    discount_amount   BIGINT       NOT NULL DEFAULT 0,
    tax_rate_bps      INT          NOT NULL DEFAULT 0,
    tax_amount        BIGINT       NOT NULL DEFAULT 0,
    total_price       BIGINT       NOT NULL,
    commission_bps    INT          NULL,
    commission_amount BIGINT       NULL,
    seller_net_amount BIGINT       NULL,
    currency          VARCHAR(10)  NOT NULL,
    snapshotted_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE RULE order_item_snapshot_no_update AS ON UPDATE TO catalog.order_item_price_snapshots DO INSTEAD NOTHING;
CREATE OR REPLACE RULE order_item_snapshot_no_delete AS ON DELETE TO catalog.order_item_price_snapshots DO INSTEAD NOTHING;

CREATE INDEX idx_oi_price_snapshots_order ON catalog.order_item_price_snapshots(order_id);

-- ---------------------------------------------------------------------------
-- catalog.tax_snapshots (from extensions)
-- Placed here after order_items which they reference
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.tax_snapshots (
    id              BIGSERIAL   PRIMARY KEY,
    order_item_id   BIGINT      NOT NULL UNIQUE REFERENCES catalog.order_items(id) ON DELETE RESTRICT,
    order_id        BIGINT      NOT NULL REFERENCES catalog.orders(id) ON DELETE RESTRICT,
    tax_rate_id     BIGINT      NULL REFERENCES catalog.tax_rates(id) ON DELETE SET NULL,
    jurisdiction_id BIGINT      NULL REFERENCES catalog.tax_jurisdictions(id) ON DELETE SET NULL,
    tax_type        VARCHAR(30) NOT NULL,
    rate_bps        INT         NOT NULL,
    taxable_amount  BIGINT      NOT NULL,
    tax_amount      BIGINT      NOT NULL,
    currency        VARCHAR(10) NOT NULL,
    snapshotted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE RULE tax_snapshots_no_update AS ON UPDATE TO catalog.tax_snapshots DO INSTEAD NOTHING;
CREATE OR REPLACE RULE tax_snapshots_no_delete AS ON DELETE TO catalog.tax_snapshots DO INSTEAD NOTHING;

CREATE INDEX idx_catalog_tax_snapshots_order ON catalog.tax_snapshots(order_id);
CREATE INDEX idx_catalog_tax_snapshots_rate  ON catalog.tax_snapshots(tax_rate_id);

-- ---------------------------------------------------------------------------
-- catalog.order_payments
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.order_payments (
    id                    BIGSERIAL    PRIMARY KEY,
    order_id              BIGINT       NOT NULL REFERENCES catalog.orders(id) ON DELETE CASCADE,
    payment_method_id     INT          NULL REFERENCES catalog.payment_methods(id),
    transaction_reference VARCHAR(255) NULL,
    amount                BIGINT       NOT NULL,
    status                VARCHAR(30)  NOT NULL DEFAULT 'pending',
    attempt_number        INT          NOT NULL DEFAULT 1,
    gateway_response      JSONB        NULL,
    failure_reason        TEXT         NULL,
    metadata              JSONB        NULL,
    paid_at               TIMESTAMPTZ  NULL,
    captured_at           TIMESTAMPTZ  NULL,
    refunded_at           TIMESTAMPTZ  NULL,
    UNIQUE (payment_method_id, transaction_reference)
);
CREATE INDEX idx_catalog_order_payments ON catalog.order_payments(order_id, status);

-- ---------------------------------------------------------------------------
-- catalog.payment_provider_events / processed_idempotency_keys (from extensions)
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.payment_provider_events (
    id                  BIGSERIAL    PRIMARY KEY,
    provider            VARCHAR(50)  NOT NULL,
    event_type          VARCHAR(100) NOT NULL,
    provider_event_id   VARCHAR(255) NOT NULL,
    payload             JSONB        NOT NULL,
    status              VARCHAR(20)  NOT NULL DEFAULT 'received'
                            CHECK (status IN ('received','processing','processed','ignored','failed')),
    processing_error    TEXT         NULL,
    processed_at        TIMESTAMPTZ  NULL,
    matched_order_id    BIGINT       NULL,
    matched_payment_id  BIGINT       NULL,
    ip_address          INET         NULL,
    received_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_event_id)
);

CREATE INDEX idx_ppe_provider_status ON catalog.payment_provider_events(provider, status);
CREATE INDEX idx_ppe_order           ON catalog.payment_provider_events(matched_order_id) WHERE matched_order_id IS NOT NULL;
CREATE INDEX idx_ppe_received        ON catalog.payment_provider_events(received_at DESC);
CREATE INDEX idx_ppe_pending         ON catalog.payment_provider_events(status, received_at)
    WHERE status IN ('received','processing');

CREATE TABLE catalog.processed_idempotency_keys (
    key             VARCHAR(255) PRIMARY KEY,
    operation_type  VARCHAR(50)  NOT NULL,
    result_summary  TEXT         NULL,
    result_payload  JSONB        NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ  NOT NULL DEFAULT (NOW() + INTERVAL '72 hours')
);

CREATE INDEX idx_idempotency_expires ON catalog.processed_idempotency_keys(expires_at);
CREATE INDEX idx_idempotency_type    ON catalog.processed_idempotency_keys(operation_type);

-- ---------------------------------------------------------------------------
-- catalog.order_fulfillments / shipment_tracking
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.order_fulfillments (
    id                BIGSERIAL   PRIMARY KEY,
    order_id          BIGINT      NOT NULL REFERENCES catalog.orders(id)        ON DELETE CASCADE,
    seller_order_id   BIGINT      NULL     REFERENCES catalog.seller_orders(id) ON DELETE SET NULL,
    store_id          BIGINT      NULL     REFERENCES catalog.stores(id)         ON DELETE SET NULL,
    status            VARCHAR(30) NOT NULL DEFAULT 'pending',
    estimated_delivery TIMESTAMPTZ NULL,
    shipped_at        TIMESTAMPTZ NULL,
    delivered_at      TIMESTAMPTZ NULL,
    notes             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE catalog.shipment_tracking (
    id              BIGSERIAL    PRIMARY KEY,
    fulfillment_id  BIGINT       NOT NULL REFERENCES catalog.order_fulfillments(id) ON DELETE CASCADE,
    carrier         VARCHAR(100) NULL,
    tracking_number VARCHAR(100) NULL,
    status          VARCHAR(50)  NULL,
    event_time      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    location        VARCHAR(255) NULL
);
CREATE INDEX idx_catalog_shipment_tracking ON catalog.shipment_tracking(fulfillment_id, event_time);

-- ---------------------------------------------------------------------------
-- catalog.commission_rules / commission_snapshots
-- commission_rules includes deleted_at (from extensions)
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.commission_rules (
    id              BIGSERIAL    PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    store_id        BIGINT       NULL REFERENCES catalog.stores(id)             ON DELETE CASCADE,
    category_id     BIGINT       NULL REFERENCES catalog.product_categories(id) ON DELETE CASCADE,
    commission_type VARCHAR(30)  NOT NULL CHECK (commission_type IN ('percentage','fixed','percentage_plus_fixed')),
    rate_bps        INT          NULL,
    fixed_amount    BIGINT       NULL,
    min_fee         BIGINT       NULL,
    max_fee         BIGINT       NULL,
    priority        INT          NOT NULL DEFAULT 100,
    starts_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    ends_at         TIMESTAMPTZ  NULL,
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    created_by      BIGINT       NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ  NULL,   -- soft delete
    CONSTRAINT chk_commission_has_value CHECK (rate_bps IS NOT NULL OR fixed_amount IS NOT NULL)
);

CREATE INDEX idx_catalog_commission_rules_store    ON catalog.commission_rules(store_id)    WHERE is_active = TRUE;
CREATE INDEX idx_catalog_commission_rules_category ON catalog.commission_rules(category_id) WHERE is_active = TRUE;
CREATE INDEX idx_catalog_commission_rules_active   ON catalog.commission_rules(is_active, priority, starts_at);

CREATE TRIGGER trg_catalog_commission_rules_updated_at
    BEFORE UPDATE ON catalog.commission_rules
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE catalog.commission_snapshots (
    id                 BIGSERIAL   PRIMARY KEY,
    seller_order_id    BIGINT      NOT NULL UNIQUE REFERENCES catalog.seller_orders(id) ON DELETE RESTRICT,
    commission_rule_id BIGINT      NULL REFERENCES catalog.commission_rules(id) ON DELETE SET NULL,
    commission_type    VARCHAR(30) NOT NULL,
    rate_bps           INT         NULL,
    fixed_amount       BIGINT      NULL,
    gross_amount       BIGINT      NOT NULL,
    commission_amount  BIGINT      NOT NULL,
    seller_net_amount  BIGINT      NOT NULL,
    currency           VARCHAR(10) NOT NULL DEFAULT 'KES',
    calculated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- catalog.disputes / dispute_messages / dispute_evidence / refunds
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.disputes (
    id                  BIGSERIAL   PRIMARY KEY,
    seller_order_id     BIGINT      NOT NULL REFERENCES catalog.seller_orders(id) ON DELETE RESTRICT,
    order_id            BIGINT      NOT NULL REFERENCES catalog.orders(id)         ON DELETE RESTRICT,
    store_id            BIGINT      NOT NULL REFERENCES catalog.stores(id)         ON DELETE RESTRICT,
    buyer_id            BIGINT      NOT NULL REFERENCES auth.identities(id)        ON DELETE RESTRICT,
    reason              VARCHAR(30) NOT NULL,
    description         TEXT        NOT NULL,
    status              VARCHAR(25) NOT NULL DEFAULT 'open',
    resolution          TEXT,
    resolved_by         BIGINT      NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    resolved_at         TIMESTAMPTZ NULL,
    refund_amount       BIGINT      NULL,
    seller_deduction    BIGINT      NULL,
    seller_response_due TIMESTAMPTZ NULL,
    auto_resolve_at     TIMESTAMPTZ NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_catalog_disputes_seller_order ON catalog.disputes(seller_order_id);
CREATE INDEX idx_catalog_disputes_store        ON catalog.disputes(store_id, status);
CREATE INDEX idx_catalog_disputes_buyer        ON catalog.disputes(buyer_id);
CREATE INDEX idx_catalog_disputes_open         ON catalog.disputes(status)
    WHERE status NOT IN ('resolved_buyer','resolved_seller','closed');

CREATE TRIGGER trg_catalog_disputes_updated_at
    BEFORE UPDATE ON catalog.disputes
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE catalog.dispute_messages (
    id          BIGSERIAL   PRIMARY KEY,
    dispute_id  BIGINT      NOT NULL REFERENCES catalog.disputes(id)     ON DELETE CASCADE,
    sender_id   BIGINT      NOT NULL REFERENCES auth.identities(id)      ON DELETE RESTRICT,
    sender_role VARCHAR(20) NOT NULL,
    message     TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_catalog_dispute_messages ON catalog.dispute_messages(dispute_id, created_at);

CREATE TABLE catalog.dispute_evidence (
    id          BIGSERIAL    PRIMARY KEY,
    dispute_id  BIGINT       NOT NULL REFERENCES catalog.disputes(id)     ON DELETE CASCADE,
    uploaded_by BIGINT       NOT NULL REFERENCES auth.identities(id)      ON DELETE RESTRICT,
    file_url    VARCHAR(500) NOT NULL,
    file_type   VARCHAR(50)  NULL,
    description TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_catalog_dispute_evidence ON catalog.dispute_evidence(dispute_id);

CREATE TABLE catalog.refunds (
    id                 BIGSERIAL   PRIMARY KEY,
    order_id           BIGINT      NOT NULL REFERENCES catalog.orders(id)       ON DELETE RESTRICT,
    seller_order_id    BIGINT      NULL     REFERENCES catalog.seller_orders(id) ON DELETE SET NULL,
    order_item_id      BIGINT      NULL     REFERENCES catalog.order_items(id)   ON DELETE SET NULL,
    dispute_id         BIGINT      NULL     REFERENCES catalog.disputes(id)      ON DELETE SET NULL,
    requester_id       BIGINT      NOT NULL REFERENCES auth.identities(id)       ON DELETE RESTRICT,
    reason             TEXT        NOT NULL,
    amount             BIGINT      NOT NULL,
    currency           VARCHAR(10) NOT NULL DEFAULT 'KES',
    status             VARCHAR(20) NOT NULL DEFAULT 'requested',
    approved_by        BIGINT      NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    approved_at        TIMESTAMPTZ NULL,
    completed_at       TIMESTAMPTZ NULL,
    failure_reason     TEXT,
    provider_reference VARCHAR(255) NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_catalog_refunds_order        ON catalog.refunds(order_id);
CREATE INDEX idx_catalog_refunds_seller_order ON catalog.refunds(seller_order_id);
CREATE INDEX idx_catalog_refunds_status       ON catalog.refunds(status);

CREATE TRIGGER trg_catalog_refunds_updated_at
    BEFORE UPDATE ON catalog.refunds
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

-- ---------------------------------------------------------------------------
-- catalog.reviews / wishlists
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.reviews (
    id                   BIGSERIAL   PRIMARY KEY,
    user_id              BIGINT      NOT NULL REFERENCES auth.identities(id)  ON DELETE CASCADE,
    product_id           BIGINT      NOT NULL REFERENCES catalog.products(id) ON DELETE CASCADE,
    store_id             BIGINT      NULL     REFERENCES catalog.stores(id)   ON DELETE SET NULL,
    order_item_id        BIGINT      NULL     REFERENCES catalog.order_items(id) ON DELETE SET NULL,
    rating               INT         NOT NULL CHECK (rating BETWEEN 1 AND 5),
    seller_rating        INT         NULL     CHECK (seller_rating IS NULL OR seller_rating BETWEEN 1 AND 5),
    comment              TEXT,
    is_verified_purchase BOOLEAN     NOT NULL DEFAULT FALSE,
    is_moderated         BOOLEAN     NOT NULL DEFAULT FALSE,
    moderation_reason    TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_catalog_reviews_product ON catalog.reviews(product_id);
CREATE INDEX idx_catalog_reviews_store   ON catalog.reviews(store_id) WHERE store_id IS NOT NULL;
CREATE UNIQUE INDEX ux_catalog_reviews_user_order_item ON catalog.reviews(user_id, order_item_id)
    WHERE order_item_id IS NOT NULL;

CREATE TABLE catalog.wishlists (
    id         BIGSERIAL   PRIMARY KEY,
    user_id    BIGINT      NOT NULL UNIQUE REFERENCES auth.identities(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE catalog.wishlist_items (
    wishlist_id BIGINT      NOT NULL REFERENCES catalog.wishlists(id)          ON DELETE CASCADE,
    product_id  BIGINT      NOT NULL REFERENCES catalog.products(id)           ON DELETE CASCADE,
    variant_id  BIGINT      NULL     REFERENCES catalog.product_variants(id)   ON DELETE CASCADE,
    added_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (wishlist_id, product_id, COALESCE(variant_id, 0))
);

-- ---------------------------------------------------------------------------
-- catalog.store_subscriptions / subscription_invoices
-- store_subscriptions includes deleted_at (from extensions)
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.store_subscriptions (
    id                   BIGSERIAL   PRIMARY KEY,
    store_id             BIGINT      NOT NULL REFERENCES catalog.stores(id)             ON DELETE CASCADE,
    plan_id              BIGINT      NOT NULL REFERENCES catalog.subscription_plans(id) ON DELETE RESTRICT,
    status               VARCHAR(20) NOT NULL DEFAULT 'active'
                             CHECK (status IN ('active','past_due','cancelled','expired','trialing')),
    billing_cycle        VARCHAR(20) NOT NULL DEFAULT 'monthly',
    amount               BIGINT      NOT NULL,
    currency             VARCHAR(10) NOT NULL DEFAULT 'KES',
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end   TIMESTAMPTZ NOT NULL,
    cancelled_at         TIMESTAMPTZ NULL,
    cancel_reason        TEXT,
    trial_ends_at        TIMESTAMPTZ NULL,
    payment_method_id    INT         NULL REFERENCES catalog.payment_methods(id) ON DELETE SET NULL,
    metadata             JSONB,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ NULL    -- soft delete
);

CREATE INDEX idx_catalog_store_subs_store  ON catalog.store_subscriptions(store_id);
CREATE INDEX idx_catalog_store_subs_status ON catalog.store_subscriptions(status);
CREATE UNIQUE INDEX ux_catalog_store_active_subscription ON catalog.store_subscriptions(store_id)
    WHERE status IN ('active','trialing');

CREATE TRIGGER trg_catalog_store_subs_updated_at
    BEFORE UPDATE ON catalog.store_subscriptions
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE catalog.subscription_invoices (
    id              BIGSERIAL    PRIMARY KEY,
    subscription_id BIGINT       NOT NULL REFERENCES catalog.store_subscriptions(id) ON DELETE RESTRICT,
    store_id        BIGINT       NOT NULL REFERENCES catalog.stores(id)              ON DELETE RESTRICT,
    invoice_number  VARCHAR(100) NOT NULL UNIQUE,
    amount          BIGINT       NOT NULL,
    currency        VARCHAR(10)  NOT NULL DEFAULT 'KES',
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending',
    due_at          TIMESTAMPTZ  NOT NULL,
    paid_at         TIMESTAMPTZ  NULL,
    payment_ref     VARCHAR(255) NULL,
    metadata        JSONB,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_catalog_sub_invoices_store  ON catalog.subscription_invoices(store_id, created_at DESC);
CREATE INDEX idx_catalog_sub_invoices_status ON catalog.subscription_invoices(status);

-- ---------------------------------------------------------------------------
-- catalog.ad_campaigns / ad_campaign_products / ad_impressions (hypertable)
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.ad_campaigns (
    id            BIGSERIAL   PRIMARY KEY,
    store_id      BIGINT      NOT NULL REFERENCES catalog.stores(id) ON DELETE CASCADE,
    name          VARCHAR(255) NOT NULL,
    pricing_model VARCHAR(20) NOT NULL DEFAULT 'cpc'
                      CHECK (pricing_model IN ('cpc','cpm','flat_fee')),
    daily_budget  BIGINT      NOT NULL DEFAULT 0,
    total_budget  BIGINT      NULL,
    amount_spent  BIGINT      NOT NULL DEFAULT 0,
    status        VARCHAR(20) NOT NULL DEFAULT 'draft',
    starts_at     TIMESTAMPTZ NOT NULL,
    ends_at       TIMESTAMPTZ NULL,
    targeting     JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_catalog_ad_campaigns_store  ON catalog.ad_campaigns(store_id);
CREATE INDEX idx_catalog_ad_campaigns_active ON catalog.ad_campaigns(status, starts_at, ends_at);

CREATE TRIGGER trg_catalog_ad_campaigns_updated_at
    BEFORE UPDATE ON catalog.ad_campaigns
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE catalog.ad_campaign_products (
    id          BIGSERIAL PRIMARY KEY,
    campaign_id BIGINT    NOT NULL REFERENCES catalog.ad_campaigns(id) ON DELETE CASCADE,
    product_id  BIGINT    NOT NULL REFERENCES catalog.products(id)     ON DELETE CASCADE,
    bid_amount  BIGINT    NOT NULL DEFAULT 0,
    impressions BIGINT    NOT NULL DEFAULT 0,
    clicks      BIGINT    NOT NULL DEFAULT 0,
    conversions INT       NOT NULL DEFAULT 0,
    spend       BIGINT    NOT NULL DEFAULT 0,
    is_active   BOOLEAN   NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (campaign_id, product_id)
);
CREATE INDEX idx_catalog_acp_campaign ON catalog.ad_campaign_products(campaign_id) WHERE is_active = TRUE;
CREATE INDEX idx_catalog_acp_product  ON catalog.ad_campaign_products(product_id);

CREATE TABLE catalog.ad_impressions (
    id                  BIGSERIAL   NOT NULL,
    campaign_product_id BIGINT      NOT NULL REFERENCES catalog.ad_campaign_products(id) ON DELETE CASCADE,
    event_type          VARCHAR(20) NOT NULL,
    user_id             BIGINT      NULL,
    session_id          VARCHAR(255) NULL,
    cost                BIGINT      NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
);

SELECT create_hypertable('catalog.ad_impressions', 'created_at',
    chunk_time_interval => INTERVAL '1 week', if_not_exists => TRUE);

CREATE INDEX idx_catalog_ad_impressions_campaign ON catalog.ad_impressions(campaign_product_id, created_at);
CREATE INDEX idx_catalog_ad_impressions_created  ON catalog.ad_impressions(created_at DESC);

ALTER TABLE catalog.ad_impressions SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'campaign_product_id, event_type',
    timescaledb.compress_orderby   = 'created_at DESC'
);
SELECT add_compression_policy('catalog.ad_impressions', INTERVAL '30 days', if_not_exists => TRUE);

-- ---------------------------------------------------------------------------
-- catalog.analytics: product_events (hypertable) / product_metrics / seller metrics
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.product_events (
    id         BIGSERIAL   NOT NULL,
    product_id BIGINT      NOT NULL REFERENCES catalog.products(id) ON DELETE CASCADE,
    user_id    BIGINT      NULL,
    session_id VARCHAR(255) NULL,
    event_type VARCHAR(30) NOT NULL,
    quantity   INT         NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
);

SELECT create_hypertable('catalog.product_events', 'created_at',
    chunk_time_interval => INTERVAL '1 week', if_not_exists => TRUE);

CREATE INDEX idx_catalog_product_events_product ON catalog.product_events(product_id, created_at);
CREATE INDEX idx_catalog_product_events_type    ON catalog.product_events(event_type, created_at);

ALTER TABLE catalog.product_events SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'product_id, event_type',
    timescaledb.compress_orderby   = 'created_at DESC'
);
SELECT add_compression_policy('catalog.product_events', INTERVAL '30 days', if_not_exists => TRUE);

CREATE TABLE catalog.product_metrics (
    product_id       BIGINT    PRIMARY KEY REFERENCES catalog.products(id) ON DELETE CASCADE,
    trending_score   FLOAT     NOT NULL DEFAULT 0,
    daily_views      INT       NOT NULL DEFAULT 0,
    weekly_views     INT       NOT NULL DEFAULT 0,
    weekly_purchases INT       NOT NULL DEFAULT 0,
    conversion_rate  FLOAT     NOT NULL DEFAULT 0,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_catalog_product_metrics_trending ON catalog.product_metrics(trending_score DESC);

CREATE TABLE catalog.seller_daily_metrics (
    store_id        BIGINT   NOT NULL REFERENCES catalog.stores(id) ON DELETE CASCADE,
    metric_date     DATE     NOT NULL,
    orders_count    INT      NOT NULL DEFAULT 0,
    gmv             BIGINT   NOT NULL DEFAULT 0,
    net_revenue     BIGINT   NOT NULL DEFAULT 0,
    commission_paid BIGINT   NOT NULL DEFAULT 0,
    refund_amount   BIGINT   NOT NULL DEFAULT 0,
    dispute_count   INT      NOT NULL DEFAULT 0,
    product_views   BIGINT   NOT NULL DEFAULT 0,
    add_to_cart     BIGINT   NOT NULL DEFAULT 0,
    conversion_rate DECIMAL(6,4) NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (store_id, metric_date)
);
CREATE INDEX idx_catalog_seller_daily_metrics ON catalog.seller_daily_metrics(store_id, metric_date DESC);

-- ---------------------------------------------------------------------------
-- catalog.click_through_events (hypertable) / ranking_features / recommendation_signals
-- (from extensions — search ranking feedback)
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.click_through_events (
    id            BIGSERIAL   NOT NULL,
    event_type    VARCHAR(20) NOT NULL,
    product_id    BIGINT      NOT NULL REFERENCES catalog.products(id) ON DELETE CASCADE,
    store_id      BIGINT      NULL,
    user_id       BIGINT      NULL,
    session_id    VARCHAR(255) NULL,
    surface       VARCHAR(30) NOT NULL DEFAULT 'search',
    search_query  TEXT        NULL,
    position      INT         NULL,
    experiment_id VARCHAR(50) NULL,
    variant_id_exp VARCHAR(50) NULL,
    metadata      JSONB       NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
);

SELECT create_hypertable('catalog.click_through_events', 'created_at',
    chunk_time_interval => INTERVAL '1 week', if_not_exists => TRUE);

CREATE INDEX idx_cte_product  ON catalog.click_through_events(product_id, created_at DESC);
CREATE INDEX idx_cte_type     ON catalog.click_through_events(event_type, surface, created_at DESC);
CREATE INDEX idx_cte_session  ON catalog.click_through_events(session_id, created_at) WHERE session_id IS NOT NULL;
CREATE INDEX idx_cte_query    ON catalog.click_through_events(search_query, created_at DESC) WHERE search_query IS NOT NULL;

ALTER TABLE catalog.click_through_events SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'event_type, surface',
    timescaledb.compress_orderby   = 'created_at DESC'
);
SELECT add_compression_policy('catalog.click_through_events', INTERVAL '14 days', if_not_exists => TRUE);

CREATE TABLE catalog.ranking_features (
    product_id              BIGINT      PRIMARY KEY REFERENCES catalog.products(id) ON DELETE CASCADE,
    impressions_7d          BIGINT      NOT NULL DEFAULT 0,
    clicks_7d               BIGINT      NOT NULL DEFAULT 0,
    ctr_7d                  DECIMAL(8,6) NOT NULL DEFAULT 0,
    add_to_cart_7d          BIGINT      NOT NULL DEFAULT 0,
    purchases_7d            INT         NOT NULL DEFAULT 0,
    conversion_rate_7d      DECIMAL(8,6) NOT NULL DEFAULT 0,
    avg_rating              DECIMAL(3,2) NOT NULL DEFAULT 0,
    review_count            INT         NOT NULL DEFAULT 0,
    refund_rate_30d         DECIMAL(8,6) NOT NULL DEFAULT 0,
    dispute_rate_30d        DECIMAL(8,6) NOT NULL DEFAULT 0,
    seller_rating           DECIMAL(3,2) NOT NULL DEFAULT 0,
    seller_fulfillment_rate DECIMAL(8,6) NOT NULL DEFAULT 0,
    days_since_last_order   INT         NULL,
    days_since_created      INT         NOT NULL DEFAULT 0,
    current_boost_score     DECIMAL(10,4) NOT NULL DEFAULT 0,
    relevance_score         DECIMAL(10,6) NOT NULL DEFAULT 0,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_catalog_ranking_score  ON catalog.ranking_features(relevance_score DESC);
CREATE INDEX idx_catalog_ranking_rating ON catalog.ranking_features(avg_rating DESC);

CREATE TABLE catalog.recommendation_signals (
    identity_id   BIGINT      NOT NULL REFERENCES auth.identities(id)  ON DELETE CASCADE,
    product_id    BIGINT      NOT NULL REFERENCES catalog.products(id) ON DELETE CASCADE,
    signal_type   VARCHAR(30) NOT NULL,
    signal_weight DECIMAL(6,4) NOT NULL DEFAULT 1.0,
    event_count   INT         NOT NULL DEFAULT 1,
    last_event_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (identity_id, product_id, signal_type)
);
CREATE INDEX idx_rec_signals_identity ON catalog.recommendation_signals(identity_id, signal_weight DESC);
CREATE INDEX idx_rec_signals_product  ON catalog.recommendation_signals(product_id, signal_type);

-- ---------------------------------------------------------------------------
-- catalog.moderation_logs
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.moderation_logs (
    id           BIGSERIAL   PRIMARY KEY,
    entity_type  VARCHAR(50) NOT NULL,
    entity_id    BIGINT      NOT NULL,
    action       VARCHAR(20) NOT NULL,
    reason       TEXT,
    old_status   VARCHAR(50) NULL,
    new_status   VARCHAR(50) NULL,
    performed_by BIGINT      NOT NULL REFERENCES auth.identities(id) ON DELETE RESTRICT,
    notes        TEXT,
    metadata     JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_catalog_moderation_entity ON catalog.moderation_logs(entity_type, entity_id, created_at DESC);
CREATE INDEX idx_catalog_moderation_admin  ON catalog.moderation_logs(performed_by, created_at DESC);

-- ---------------------------------------------------------------------------
-- catalog.soft_delete_log (from extensions)
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.soft_delete_log (
    id              BIGSERIAL   PRIMARY KEY,
    table_name      VARCHAR(100) NOT NULL,
    record_id       BIGINT      NOT NULL,
    deleted_by      BIGINT      NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    deletion_reason VARCHAR(30) NOT NULL,
    notes           TEXT        NULL,
    snapshot        JSONB       NULL,
    deleted_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_soft_delete_log_table  ON catalog.soft_delete_log(table_name, record_id);
CREATE INDEX idx_soft_delete_log_by     ON catalog.soft_delete_log(deleted_by);
CREATE INDEX idx_soft_delete_log_reason ON catalog.soft_delete_log(deletion_reason, deleted_at DESC);

-- ---------------------------------------------------------------------------
-- catalog.media_assets (from extensions)
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.media_assets (
    id              BIGSERIAL    PRIMARY KEY,
    storage_key     VARCHAR(500) NOT NULL UNIQUE,
    asset_type      VARCHAR(50)  NOT NULL,
    mime_type       VARCHAR(100) NOT NULL,
    file_name       VARCHAR(255) NOT NULL,
    file_size_bytes BIGINT       NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending',
    uploaded_by     BIGINT       NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    owner_type      VARCHAR(30)  NULL,
    owner_id        BIGINT       NULL,
    is_moderated    BOOLEAN      NOT NULL DEFAULT FALSE,
    moderation_flag VARCHAR(30)  NULL,
    width_px        INT          NULL,
    height_px       INT          NULL,
    duration_secs   INT          NULL,
    metadata        JSONB        NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ  NULL
);
CREATE INDEX idx_media_assets_owner    ON catalog.media_assets(owner_type, owner_id) WHERE owner_type IS NOT NULL;
CREATE INDEX idx_media_assets_uploader ON catalog.media_assets(uploaded_by);
CREATE INDEX idx_media_assets_type     ON catalog.media_assets(asset_type, created_at DESC);
CREATE INDEX idx_media_assets_active   ON catalog.media_assets(status) WHERE status != 'deleted';

-- ---------------------------------------------------------------------------
-- catalog.outbox_events / event_consumers / event_delivery_attempts / dead_letter_events
-- (from extensions — outbox pattern for NATS/Kafka/async workers)
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.outbox_events (
    id              BIGSERIAL    PRIMARY KEY,
    event_type      VARCHAR(100) NOT NULL,
    aggregate_type  VARCHAR(50)  NOT NULL,
    aggregate_id    TEXT         NOT NULL,
    payload         JSONB        NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','published','failed','skipped')),
    attempt_count   INT          NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMPTZ  NULL,
    published_at    TIMESTAMPTZ  NULL,
    error_message   TEXT         NULL,
    topic           VARCHAR(200) NULL,
    partition_key   TEXT         NULL,
    idempotency_key VARCHAR(255) NULL UNIQUE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_outbox_pending    ON catalog.outbox_events(status, created_at ASC) WHERE status = 'pending';
CREATE INDEX idx_outbox_aggregate  ON catalog.outbox_events(aggregate_type, aggregate_id);
CREATE INDEX idx_outbox_event_type ON catalog.outbox_events(event_type, created_at DESC);
CREATE INDEX idx_outbox_retry      ON catalog.outbox_events(status, last_attempt_at)
    WHERE status = 'failed' AND attempt_count < 5;

CREATE TABLE catalog.event_consumers (
    id                BIGSERIAL    PRIMARY KEY,
    consumer_name     VARCHAR(100) NOT NULL UNIQUE,
    description       TEXT,
    last_event_id     BIGINT       NULL,
    last_processed_at TIMESTAMPTZ  NULL,
    is_active         BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_catalog_event_consumers_updated_at
    BEFORE UPDATE ON catalog.event_consumers
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE catalog.event_delivery_attempts (
    id              BIGSERIAL   PRIMARY KEY,
    outbox_event_id BIGINT      NOT NULL REFERENCES catalog.outbox_events(id) ON DELETE CASCADE,
    consumer_name   VARCHAR(100) NOT NULL,
    result          VARCHAR(20) NOT NULL CHECK (result IN ('success','failure','skipped')),
    error_message   TEXT        NULL,
    duration_ms     INT         NULL,
    attempted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_event_delivery_event    ON catalog.event_delivery_attempts(outbox_event_id);
CREATE INDEX idx_event_delivery_consumer ON catalog.event_delivery_attempts(consumer_name, attempted_at DESC);
CREATE INDEX idx_event_delivery_failures ON catalog.event_delivery_attempts(consumer_name, result) WHERE result = 'failure';

CREATE TABLE catalog.dead_letter_events (
    id              BIGSERIAL    PRIMARY KEY,
    outbox_event_id BIGINT       NOT NULL REFERENCES catalog.outbox_events(id) ON DELETE CASCADE,
    consumer_name   VARCHAR(100) NOT NULL,
    event_type      VARCHAR(100) NOT NULL,
    payload         JSONB        NOT NULL,
    final_error     TEXT         NOT NULL,
    attempt_count   INT          NOT NULL,
    review_status   VARCHAR(20)  NOT NULL DEFAULT 'pending_review',
    reviewed_by     BIGINT       NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    reviewed_at     TIMESTAMPTZ  NULL,
    review_notes    TEXT         NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_dead_letter_consumer ON catalog.dead_letter_events(consumer_name, created_at DESC);
CREATE INDEX idx_dead_letter_review   ON catalog.dead_letter_events(review_status) WHERE review_status = 'pending_review';

-- ---------------------------------------------------------------------------
-- catalog.webhook_subscriptions / webhook_deliveries (from extensions)
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.webhook_subscriptions (
    id                   BIGSERIAL    PRIMARY KEY,
    store_id             BIGINT       NOT NULL REFERENCES catalog.stores(id) ON DELETE CASCADE,
    endpoint_url         VARCHAR(500) NOT NULL,
    event_types          TEXT[]       NOT NULL,
    secret_hash          VARCHAR(255) NOT NULL,
    is_active            BOOLEAN      NOT NULL DEFAULT TRUE,
    health_status        VARCHAR(20)  NOT NULL DEFAULT 'active',
    consecutive_failures INT          NOT NULL DEFAULT 0,
    last_success_at      TIMESTAMPTZ  NULL,
    last_failure_at      TIMESTAMPTZ  NULL,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_webhook_subs_store ON catalog.webhook_subscriptions(store_id) WHERE is_active = TRUE;

CREATE TRIGGER trg_catalog_webhook_subs_updated_at
    BEFORE UPDATE ON catalog.webhook_subscriptions
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE catalog.webhook_deliveries (
    id               BIGSERIAL    PRIMARY KEY,
    subscription_id  BIGINT       NOT NULL REFERENCES catalog.webhook_subscriptions(id) ON DELETE CASCADE,
    outbox_event_id  BIGINT       NULL REFERENCES catalog.outbox_events(id) ON DELETE SET NULL,
    event_type       VARCHAR(100) NOT NULL,
    payload          JSONB        NOT NULL,
    status           VARCHAR(20)  NOT NULL DEFAULT 'pending',
    http_status_code INT          NULL,
    response_body    TEXT         NULL,
    attempt_count    INT          NOT NULL DEFAULT 0,
    next_retry_at    TIMESTAMPTZ  NULL,
    delivered_at     TIMESTAMPTZ  NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_webhook_deliveries_sub   ON catalog.webhook_deliveries(subscription_id, created_at DESC);
CREATE INDEX idx_webhook_deliveries_retry ON catalog.webhook_deliveries(status, next_retry_at)
    WHERE status = 'failed' AND next_retry_at IS NOT NULL;

-- ---------------------------------------------------------------------------
-- catalog.buyer_wallets / store_credits / wallet_transactions (from extensions)
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.buyer_wallets (
    id                BIGSERIAL   PRIMARY KEY,
    identity_id       BIGINT      NOT NULL REFERENCES auth.identities(id) ON DELETE RESTRICT,
    currency          VARCHAR(10) NOT NULL DEFAULT 'KES',
    balance           BIGINT      NOT NULL DEFAULT 0 CHECK (balance >= 0),
    pending_credit    BIGINT      NOT NULL DEFAULT 0 CHECK (pending_credit >= 0),
    is_active         BOOLEAN     NOT NULL DEFAULT TRUE,
    is_locked         BOOLEAN     NOT NULL DEFAULT FALSE,
    ledger_account_id BIGINT      NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (identity_id, currency)
);
CREATE INDEX idx_catalog_buyer_wallets_identity ON catalog.buyer_wallets(identity_id);

CREATE TRIGGER trg_catalog_buyer_wallets_updated_at
    BEFORE UPDATE ON catalog.buyer_wallets
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE catalog.store_credits (
    id               BIGSERIAL   PRIMARY KEY,
    identity_id      BIGINT      NOT NULL REFERENCES auth.identities(id) ON DELETE CASCADE,
    store_id         BIGINT      NOT NULL REFERENCES catalog.stores(id)  ON DELETE CASCADE,
    original_amount  BIGINT      NOT NULL CHECK (original_amount > 0),
    remaining_amount BIGINT      NOT NULL CHECK (remaining_amount >= 0),
    currency         VARCHAR(10) NOT NULL DEFAULT 'KES',
    status           VARCHAR(20) NOT NULL DEFAULT 'active'
                         CHECK (status IN ('active','exhausted','expired','revoked')),
    credit_type      VARCHAR(20) NOT NULL,
    reason           TEXT,
    issued_by        BIGINT      NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    expires_at       TIMESTAMPTZ NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_catalog_store_credits_identity ON catalog.store_credits(identity_id, store_id, status);
CREATE INDEX idx_catalog_store_credits_active   ON catalog.store_credits(identity_id, status, expires_at)
    WHERE status = 'active';

CREATE TRIGGER trg_catalog_store_credits_updated_at
    BEFORE UPDATE ON catalog.store_credits
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

-- wallet_transactions: insert-only audit log
CREATE TABLE catalog.wallet_transactions (
    id               BIGSERIAL   PRIMARY KEY,
    wallet_type      VARCHAR(20) NOT NULL CHECK (wallet_type IN ('wallet','store_credit')),
    wallet_id        BIGINT      NOT NULL,
    identity_id      BIGINT      NOT NULL REFERENCES auth.identities(id) ON DELETE RESTRICT,
    direction        VARCHAR(10) NOT NULL CHECK (direction IN ('credit','debit')),
    amount           BIGINT      NOT NULL CHECK (amount > 0),
    balance_after    BIGINT      NOT NULL,
    currency         VARCHAR(10) NOT NULL,
    transaction_type VARCHAR(30) NOT NULL,
    reference_type   VARCHAR(30) NULL,
    reference_id     BIGINT      NULL,
    description      TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE RULE wallet_transactions_no_update AS ON UPDATE TO catalog.wallet_transactions DO INSTEAD NOTHING;
CREATE OR REPLACE RULE wallet_transactions_no_delete AS ON DELETE TO catalog.wallet_transactions DO INSTEAD NOTHING;

CREATE INDEX idx_wallet_txns_identity  ON catalog.wallet_transactions(identity_id, created_at DESC);
CREATE INDEX idx_wallet_txns_wallet    ON catalog.wallet_transactions(wallet_type, wallet_id, created_at DESC);
CREATE INDEX idx_wallet_txns_reference ON catalog.wallet_transactions(reference_type, reference_id);

-- ---------------------------------------------------------------------------
-- catalog.external_integrations / integration_logs / api_credentials (from extensions)
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.external_integrations (
    id                   BIGSERIAL    PRIMARY KEY,
    store_id             BIGINT       NULL REFERENCES catalog.stores(id) ON DELETE CASCADE,
    integration_type     VARCHAR(40)  NOT NULL,
    provider_name        VARCHAR(100) NOT NULL,
    status               VARCHAR(20)  NOT NULL DEFAULT 'pending_auth',
    config               JSONB        NULL,
    secret_ref           VARCHAR(255) NULL,
    last_health_check_at TIMESTAMPTZ  NULL,
    last_health_status   VARCHAR(20)  NULL,
    consecutive_failures INT          NOT NULL DEFAULT 0,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ext_integrations_store ON catalog.external_integrations(store_id) WHERE store_id IS NOT NULL;
CREATE INDEX idx_ext_integrations_type  ON catalog.external_integrations(integration_type, status);

CREATE TRIGGER trg_catalog_ext_integrations_updated_at
    BEFORE UPDATE ON catalog.external_integrations
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE catalog.integration_logs (
    id             BIGSERIAL   NOT NULL,
    integration_id BIGINT      NOT NULL REFERENCES catalog.external_integrations(id) ON DELETE CASCADE,
    log_type       VARCHAR(30) NOT NULL,
    method         VARCHAR(20) NULL,
    endpoint       VARCHAR(500) NULL,
    request_body   JSONB       NULL,
    response_body  JSONB       NULL,
    http_status    INT         NULL,
    result         VARCHAR(20) NOT NULL,
    duration_ms    INT         NULL,
    error_message  TEXT        NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
);

SELECT create_hypertable('catalog.integration_logs', 'created_at',
    chunk_time_interval => INTERVAL '1 week', if_not_exists => TRUE);

CREATE INDEX idx_integration_logs_integration ON catalog.integration_logs(integration_id, created_at DESC);
CREATE INDEX idx_integration_logs_failures    ON catalog.integration_logs(result, created_at DESC) WHERE result = 'failure';

ALTER TABLE catalog.integration_logs SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'integration_id, result',
    timescaledb.compress_orderby   = 'created_at DESC'
);
SELECT add_compression_policy('catalog.integration_logs', INTERVAL '30 days', if_not_exists => TRUE);

CREATE TABLE catalog.api_credentials (
    id           BIGSERIAL    PRIMARY KEY,
    store_id     BIGINT       NOT NULL REFERENCES catalog.stores(id) ON DELETE CASCADE,
    name         VARCHAR(100) NOT NULL,
    key_hash     VARCHAR(64)  NOT NULL UNIQUE,
    key_prefix   VARCHAR(12)  NOT NULL,
    scope        VARCHAR(20)  NOT NULL DEFAULT 'read',
    status       VARCHAR(20)  NOT NULL DEFAULT 'active',
    last_used_at TIMESTAMPTZ  NULL,
    expires_at   TIMESTAMPTZ  NULL,
    revoked_at   TIMESTAMPTZ  NULL,
    revoked_by   BIGINT       NULL REFERENCES auth.identities(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_api_credentials_store  ON catalog.api_credentials(store_id, status);
CREATE INDEX idx_api_credentials_active ON catalog.api_credentials(key_hash) WHERE status = 'active';

-- ---------------------------------------------------------------------------
-- catalog.scheduled_jobs / job_runs (from extensions)
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.scheduled_jobs (
    id              BIGSERIAL    PRIMARY KEY,
    job_name        VARCHAR(100) NOT NULL UNIQUE,
    description     TEXT,
    schedule        VARCHAR(100) NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active','paused','disabled')),
    last_run_at     TIMESTAMPTZ  NULL,
    last_run_status VARCHAR(20)  NULL,
    next_run_at     TIMESTAMPTZ  NULL,
    timeout_seconds INT          NOT NULL DEFAULT 300,
    max_retries     INT          NOT NULL DEFAULT 3,
    metadata        JSONB        NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_catalog_scheduled_jobs_updated_at
    BEFORE UPDATE ON catalog.scheduled_jobs
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE catalog.job_runs (
    id              BIGSERIAL    PRIMARY KEY,
    job_id          BIGINT       NOT NULL REFERENCES catalog.scheduled_jobs(id) ON DELETE CASCADE,
    job_name        VARCHAR(100) NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'running'
                        CHECK (status IN ('running','succeeded','failed','timed_out','skipped')),
    items_processed INT          NULL,
    error_message   TEXT         NULL,
    error_detail    JSONB        NULL,
    duration_ms     INT          NULL,
    worker_id       VARCHAR(100) NULL,
    started_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ  NULL
);
CREATE INDEX idx_job_runs_job      ON catalog.job_runs(job_id, started_at DESC);
CREATE INDEX idx_job_runs_status   ON catalog.job_runs(status, started_at DESC);
CREATE INDEX idx_job_runs_name     ON catalog.job_runs(job_name, started_at DESC);
CREATE INDEX idx_job_runs_failures ON catalog.job_runs(job_name, status) WHERE status = 'failed';

-- ---------------------------------------------------------------------------
-- catalog.data_retention_policies (from extensions)
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.data_retention_policies (
    id               BIGSERIAL    PRIMARY KEY,
    table_name       VARCHAR(100) NOT NULL UNIQUE,
    schema_name      VARCHAR(50)  NOT NULL DEFAULT 'public',
    retention_action VARCHAR(20)  NOT NULL
                         CHECK (retention_action IN ('soft_delete','hard_delete','archive','timescale_drop')),
    retention_days   INT          NOT NULL,
    age_column       VARCHAR(50)  NOT NULL DEFAULT 'created_at',
    filter_condition TEXT         NULL,
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    last_run_at      TIMESTAMPTZ  NULL,
    last_rows_affected BIGINT     NULL,
    notes            TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- catalog.search infrastructure
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.search_events (
    id               BIGSERIAL    NOT NULL,
    query            TEXT         NOT NULL,
    normalized_query TEXT         NOT NULL,
    user_id          BIGINT       NULL,
    session_id       VARCHAR(255) NULL,
    result_count     INT          NOT NULL DEFAULT 0 CHECK (result_count >= 0),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
);

SELECT create_hypertable('catalog.search_events', 'created_at',
    chunk_time_interval => INTERVAL '1 week', if_not_exists => TRUE);

CREATE INDEX idx_catalog_search_events_query ON catalog.search_events
    USING GIN (normalized_query gin_trgm_ops);
CREATE INDEX idx_catalog_search_events_user  ON catalog.search_events(user_id, created_at);

ALTER TABLE catalog.search_events SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'normalized_query',
    timescaledb.compress_orderby   = 'created_at DESC'
);
SELECT add_compression_policy('catalog.search_events', INTERVAL '60 days', if_not_exists => TRUE);

CREATE TABLE catalog.product_search_documents (
    product_id      BIGINT    PRIMARY KEY REFERENCES catalog.products(id) ON DELETE CASCADE,
    search_document TEXT      NOT NULL,
    search_vector   TSVECTOR  NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_catalog_search_vector ON catalog.product_search_documents USING GIN(search_vector);
CREATE INDEX idx_catalog_search_trgm   ON catalog.product_search_documents
    USING GIN (search_document gin_trgm_ops);

CREATE TABLE catalog.product_boosts (
    id          BIGSERIAL        PRIMARY KEY,
    product_id  BIGINT           NOT NULL REFERENCES catalog.products(id) ON DELETE CASCADE,
    feed_type   VARCHAR(50)      NOT NULL CHECK (feed_type ~ '^[a-z0-9_]+$'),
    boost_score DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (boost_score >= 0),
    start_at    TIMESTAMPTZ      NOT NULL,
    end_at      TIMESTAMPTZ      NOT NULL,
    created_at  TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    CHECK (end_at > start_at)
);
CREATE INDEX idx_catalog_product_boosts_feed ON catalog.product_boosts(feed_type, start_at, end_at);

-- ---------------------------------------------------------------------------
-- catalog.banners / homepage_sections / merchant feed infrastructure
-- ---------------------------------------------------------------------------
CREATE TABLE catalog.banners (
    id            BIGSERIAL    PRIMARY KEY,
    title         VARCHAR(255) NULL,
    image_url     VARCHAR(500) NULL,
    redirect_type VARCHAR(50)  NULL,
    redirect_id   BIGINT       NULL,
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    start_date    TIMESTAMPTZ  NULL,
    end_date      TIMESTAMPTZ  NULL,
    priority      INT          NOT NULL DEFAULT 0
);

CREATE TABLE catalog.homepage_sections (
    id           BIGSERIAL    PRIMARY KEY,
    title        VARCHAR(255) NULL,
    type         VARCHAR(50)  NOT NULL,
    reference_id BIGINT       NULL,
    sort_order   INT          NOT NULL DEFAULT 0,
    is_active    BOOLEAN      NOT NULL DEFAULT TRUE
);
CREATE INDEX idx_catalog_homepage_active ON catalog.homepage_sections(is_active, sort_order ASC, id ASC)
    WHERE is_active = TRUE;

CREATE TABLE catalog.merchant_feed_runs (
    id             BIGSERIAL    PRIMARY KEY,
    feed_id        VARCHAR(100) NOT NULL,
    export_mode    VARCHAR(20)  NOT NULL DEFAULT 'full',
    total_items    BIGINT       NOT NULL DEFAULT 0,
    valid_items    BIGINT       NOT NULL DEFAULT 0,
    invalid_items  BIGINT       NOT NULL DEFAULT 0,
    error_count    INT          NOT NULL DEFAULT 0,
    duration_ms    INT          NULL,
    schema_version VARCHAR(20)  NOT NULL DEFAULT '1.0',
    started_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    completed_at   TIMESTAMPTZ  NULL,
    status         VARCHAR(20)  NOT NULL DEFAULT 'running',
    error_message  TEXT         NULL
);

CREATE TABLE catalog.merchant_item_overrides (
    variant_id              BIGINT       PRIMARY KEY REFERENCES catalog.product_variants(id) ON DELETE CASCADE,
    gtin                    VARCHAR(14)  NULL,
    mpn                     VARCHAR(100) NULL,
    condition               VARCHAR(20)  NULL,
    google_product_category VARCHAR(500) NULL,
    custom_label_0          VARCHAR(100) NULL,
    custom_label_1          VARCHAR(100) NULL,
    custom_label_2          VARCHAR(100) NULL,
    custom_label_3          VARCHAR(100) NULL,
    custom_label_4          VARCHAR(100) NULL,
    adult                   BOOLEAN      NOT NULL DEFAULT FALSE,
    expiration_date         DATE         NULL,
    cost_of_goods_sold      BIGINT       NULL,
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE catalog.merchant_category_map (
    category_id             BIGINT       PRIMARY KEY REFERENCES catalog.product_categories(id) ON DELETE CASCADE,
    google_product_category VARCHAR(500) NOT NULL,
    gmc_category_id         INT          NULL,
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- catalog: TimescaleDB continuous aggregate
-- ---------------------------------------------------------------------------
CREATE MATERIALIZED VIEW catalog.cte_daily_stats
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 day', created_at) AS day,
    product_id,
    surface,
    event_type,
    COUNT(*)                          AS event_count,
    COUNT(DISTINCT session_id)        AS unique_sessions,
    COUNT(DISTINCT user_id)           AS unique_users
FROM catalog.click_through_events
GROUP BY day, product_id, surface, event_type
WITH NO DATA;

SELECT add_continuous_aggregate_policy(
    'catalog.cte_daily_stats',
    start_offset      => INTERVAL '2 days',
    end_offset        => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists     => TRUE
);

-- ---------------------------------------------------------------------------
-- catalog: seed data
-- ---------------------------------------------------------------------------
INSERT INTO catalog.subscription_plans
    (name, slug, description, monthly_price, annual_price, currency,
     max_products, max_staff, analytics_access, featured_slots, ad_credits,
     priority_support, commission_discount_bps, is_active, sort_order)
VALUES
    ('Free',       'free',       'Zero cost, basic features.',                  0,       0,       'KES', 20,   1,  FALSE, 0,  0,       FALSE, 0,   TRUE, 1),
    ('Basic',      'basic',      'Growing sellers: more listings + analytics.', 99900,   999000,  'KES', 200,  3,  TRUE,  2,  50000,   FALSE, 50,  TRUE, 2),
    ('Pro',        'pro',        'Established sellers: full features.',         249900,  2499000, 'KES', 1000, 10, TRUE,  10, 250000,  TRUE,  100, TRUE, 3),
    ('Enterprise', 'enterprise', 'Unlimited scale, dedicated support.',         999900,  9999000, 'KES', -1,   50, TRUE,  50, 1000000, TRUE,  200, TRUE, 4)
ON CONFLICT (slug) DO NOTHING;

INSERT INTO catalog.store_permissions (name, code, description) VALUES
    ('View Orders',     'orders.view',     'View store orders'),
    ('Manage Orders',   'orders.manage',   'Process and manage orders'),
    ('Manage Products', 'products.manage', 'Create, edit, delete products'),
    ('View Analytics',  'analytics.view',  'Access store analytics'),
    ('Manage Payouts',  'payouts.manage',  'View and request payouts'),
    ('Manage Staff',    'staff.manage',    'Invite and manage store staff'),
    ('Manage Settings', 'settings.manage', 'Update store settings')
ON CONFLICT (code) DO NOTHING;

INSERT INTO catalog.scheduled_jobs (job_name, description, schedule, timeout_seconds, max_retries) VALUES
    ('escrow_auto_release',          'Release escrow holds past auto_release_at',         '*/5 * * * *',  120,  3),
    ('settlement_generation',        'Generate pending settlements for ready seller orders','0 2 * * *',  600,  2),
    ('payout_processing',            'Process approved payout batches',                   '0 9 * * 1-5', 900,  2),
    ('reconciliation_snapshots',     'Take daily balance snapshots for reconciliation',   '30 1 * * *',  300,  2),
    ('inventory_reservation_expiry', 'Expire stale cart reservations',                   '*/3 * * * *',  60,  3),
    ('outbox_relay',                 'Publish pending outbox events to message broker',   '*/1 * * * *',  60,  5),
    ('dead_letter_review',           'Alert team to events in dead-letter queue',         '0 8 * * *',    60,  1),
    ('fund_lock_expiry',             'Auto-release expired fund locks',                   '*/10 * * * *', 120, 3),
    ('merchant_feed_export',         'Export Google Merchant Center product feed',        '0 3 * * *',   1800, 1),
    ('seller_metrics_rollup',        'Aggregate seller_daily_metrics from events',        '15 2 * * *',  600,  2),
    ('notification_dispatch',        'Dispatch queued notifications',                     '*/2 * * * *',  90,  3),
    ('subscription_billing',         'Charge due subscription invoices',                  '0 10 * * *',  300,  2),
    ('idempotency_key_cleanup',      'Purge expired processed_idempotency_keys rows',     '0 4 * * *',    60,  1),
    ('data_retention_sweep',         'Apply data_retention_policies across all tables',   '0 3 * * *',   1800, 1)
ON CONFLICT (job_name) DO NOTHING;

INSERT INTO catalog.data_retention_policies
    (table_name, schema_name, retention_action, retention_days, age_column, filter_condition, notes)
VALUES
    ('audit_log',                 'auth',      'timescale_drop', 730,  'created_at', NULL,                                  '2-year security log'),
    ('sessions',                  'auth',      'hard_delete',    90,   'expires_at', 'status != ''active''',                'Expired/revoked sessions'),
    ('verification_tokens',       'auth',      'hard_delete',    30,   'created_at', 'used_at IS NOT NULL',                 'Used tokens'),
    ('product_events',            'catalog',   'timescale_drop', 365,  'created_at', NULL,                                  '1-year analytics'),
    ('search_events',             'catalog',   'timescale_drop', 180,  'created_at', NULL,                                  '6-month search events'),
    ('ad_impressions',            'catalog',   'timescale_drop', 180,  'created_at', NULL,                                  '6-month ad data'),
    ('click_through_events',      'catalog',   'timescale_drop', 365,  'created_at', NULL,                                  '1-year ranking signals'),
    ('outbox_events',             'catalog',   'hard_delete',    30,   'created_at', 'status IN (''published'',''skipped'')', 'Processed outbox events'),
    ('event_delivery_attempts',   'catalog',   'hard_delete',    60,   'attempted_at', NULL,                                '60-day delivery log'),
    ('processed_idempotency_keys','catalog',   'hard_delete',    3,    'expires_at', 'expires_at < NOW()',                  'Expired keys'),
    ('inventory_movements',       'catalog',   'archive',        1095, 'created_at', NULL,                                  '3-year inventory audit'),
    ('integration_logs',          'catalog',   'timescale_drop', 90,   'created_at', NULL,                                  '90-day integration logs'),
    ('job_runs',                  'catalog',   'hard_delete',    90,   'started_at', 'status != ''running''',               'Completed job history'),
    ('payment_provider_events',   'catalog',   'archive',        1095, 'received_at', NULL,                                 '3-year payment audit'),
    ('entries',                   'accounting','timescale_drop', 2555, 'created_at', NULL,                                  '7-year financial records'),
    ('audit_log',                 'accounting','timescale_drop', 1825, 'created_at', NULL,                                  '5-year financial audit')
ON CONFLICT (table_name) DO NOTHING;

COMMIT;

-- =============================================================================
-- =============================================================================
-- SCHEMA: accounting
-- Double-entry ledger, escrow, payouts, settlements, fee rules
-- =============================================================================
-- =============================================================================

BEGIN;

-- ---------------------------------------------------------------------------
-- accounting.currencies
-- ---------------------------------------------------------------------------
CREATE TABLE accounting.currencies (
    code        VARCHAR(8)  PRIMARY KEY,
    name        TEXT        NOT NULL,
    symbol      VARCHAR(10) NULL,
    decimals    SMALLINT    NOT NULL DEFAULT 2 CHECK (decimals >= 0 AND decimals <= 8),
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounting_currencies_active ON accounting.currencies(is_active) WHERE is_active = TRUE;

CREATE TRIGGER trg_accounting_currencies_updated_at
    BEFORE UPDATE ON accounting.currencies
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

INSERT INTO accounting.currencies (code, name, symbol, decimals) VALUES
    ('KES', 'Kenyan Shilling',    'KSh', 2),
    ('USD', 'US Dollar',          '$',   2),
    ('EUR', 'Euro',               '€',   2),
    ('GBP', 'British Pound',      '£',   2),
    ('UGX', 'Ugandan Shilling',   'USh', 0),
    ('TZS', 'Tanzanian Shilling', 'TSh', 0)
ON CONFLICT (code) DO NOTHING;

-- ---------------------------------------------------------------------------
-- accounting.fx_rates — hypertable, insert-only historical rates
-- ---------------------------------------------------------------------------
CREATE TABLE accounting.fx_rates (
    id             BIGSERIAL    NOT NULL,
    base_currency  VARCHAR(8)   NOT NULL REFERENCES accounting.currencies(code),
    quote_currency VARCHAR(8)   NOT NULL REFERENCES accounting.currencies(code),
    -- rate × 1,000,000 implied decimals: USDKES 130.50 → 130500000
    rate           BIGINT       NOT NULL CHECK (rate > 0),
    source         VARCHAR(50)  NULL,
    valid_from     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    valid_to       TIMESTAMPTZ  NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, valid_from)
);

SELECT create_hypertable('accounting.fx_rates', 'valid_from',
    chunk_time_interval => INTERVAL '7 days', if_not_exists => TRUE);

CREATE INDEX idx_accounting_fx_rates_current
    ON accounting.fx_rates(base_currency, quote_currency, valid_from DESC)
    WHERE valid_to IS NULL;

ALTER TABLE accounting.fx_rates SET (
    timescaledb.compress, timescaledb.compress_orderby = 'valid_from DESC'
);
SELECT add_compression_policy('accounting.fx_rates', INTERVAL '90 days', if_not_exists => TRUE);

-- ---------------------------------------------------------------------------
-- accounting.accounts
-- ---------------------------------------------------------------------------
CREATE TABLE accounting.accounts (
    id              BIGSERIAL    PRIMARY KEY,
    account_number  VARCHAR(30)  NOT NULL UNIQUE,
    owner_type      VARCHAR(20)  NOT NULL CHECK (owner_type IN ('user','store','system')),
    owner_id        TEXT         NOT NULL,
    currency        VARCHAR(8)   NOT NULL REFERENCES accounting.currencies(code),
    purpose         VARCHAR(20)  NOT NULL
                        CHECK (purpose IN ('wallet','escrow','revenue','fees','settlement','refund_reserve')),
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    is_locked       BOOLEAN      NOT NULL DEFAULT FALSE,
    overdraft_limit BIGINT       NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (owner_type, owner_id, currency, purpose)
);

CREATE INDEX idx_accounting_accounts_owner    ON accounting.accounts(owner_type, owner_id);
CREATE INDEX idx_accounting_accounts_currency ON accounting.accounts(currency) WHERE is_active = TRUE;

CREATE TRIGGER trg_accounting_accounts_updated_at
    BEFORE UPDATE ON accounting.accounts
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

-- ---------------------------------------------------------------------------
-- accounting.balances
-- ---------------------------------------------------------------------------
CREATE TABLE accounting.balances (
    account_id        BIGINT  PRIMARY KEY REFERENCES accounting.accounts(id) ON DELETE CASCADE,
    balance           BIGINT  NOT NULL DEFAULT 0,
    available_balance BIGINT  NOT NULL DEFAULT 0,
    pending_debit     BIGINT  NOT NULL DEFAULT 0,
    pending_credit    BIGINT  NOT NULL DEFAULT 0,
    last_entry_id     BIGINT  NULL,
    version           BIGINT  NOT NULL DEFAULT 0,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_balance_non_negative   CHECK (balance >= 0),
    CONSTRAINT chk_available_non_negative CHECK (available_balance >= 0)
);

-- ---------------------------------------------------------------------------
-- accounting.journals
-- ---------------------------------------------------------------------------
CREATE TABLE accounting.journals (
    id              BIGSERIAL    PRIMARY KEY,
    idempotency_key VARCHAR(255) NULL UNIQUE,
    journal_type    VARCHAR(50)  NOT NULL,
    reference_type  VARCHAR(50)  NULL,
    reference_id    TEXT         NULL,
    description     TEXT,
    created_by      TEXT         NOT NULL DEFAULT 'system',
    ip_address      INET         NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounting_journals_type      ON accounting.journals(journal_type, created_at DESC);
CREATE INDEX idx_accounting_journals_reference ON accounting.journals(reference_type, reference_id)
    WHERE reference_type IS NOT NULL;
CREATE INDEX idx_accounting_journals_created   ON accounting.journals(created_at DESC);

-- ---------------------------------------------------------------------------
-- accounting.journal_reversals (from extensions)
-- Placed immediately after journals since it self-references them
-- ---------------------------------------------------------------------------
CREATE TABLE accounting.journal_reversals (
    id                  BIGSERIAL   PRIMARY KEY,
    reversal_journal_id BIGINT      NOT NULL REFERENCES accounting.journals(id) ON DELETE RESTRICT,
    original_journal_id BIGINT      NOT NULL REFERENCES accounting.journals(id) ON DELETE RESTRICT,
    reversal_type       VARCHAR(10) NOT NULL DEFAULT 'full'
                            CHECK (reversal_type IN ('full','partial')),
    reversed_amount     BIGINT      NOT NULL CHECK (reversed_amount > 0),
    currency            VARCHAR(10) NOT NULL,
    reason              TEXT        NOT NULL,
    reversal_category   VARCHAR(30) NOT NULL,
    initiated_by        BIGINT      NOT NULL REFERENCES auth.identities(id) ON DELETE RESTRICT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (reversal_journal_id),
    CONSTRAINT chk_not_self_reversal CHECK (reversal_journal_id <> original_journal_id)
);

CREATE INDEX idx_accounting_journal_reversals_original ON accounting.journal_reversals(original_journal_id);

-- ---------------------------------------------------------------------------
-- accounting.entries — double-entry ledger, TimescaleDB hypertable, insert-only
-- ---------------------------------------------------------------------------
CREATE TABLE accounting.entries (
    id            BIGSERIAL   NOT NULL,
    journal_id    BIGINT      NOT NULL REFERENCES accounting.journals(id),
    account_id    BIGINT      NOT NULL REFERENCES accounting.accounts(id),
    dr_cr         VARCHAR(2)  NOT NULL CHECK (dr_cr IN ('DR','CR')),
    amount        BIGINT      NOT NULL CHECK (amount > 0),
    currency      VARCHAR(8)  NOT NULL REFERENCES accounting.currencies(code),
    balance_after BIGINT      NOT NULL,
    description   TEXT,
    metadata      JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
);

CREATE OR REPLACE RULE accounting_entries_no_update
    AS ON UPDATE TO accounting.entries DO INSTEAD NOTHING;
CREATE OR REPLACE RULE accounting_entries_no_delete
    AS ON DELETE TO accounting.entries DO INSTEAD NOTHING;

SELECT create_hypertable('accounting.entries', 'created_at',
    chunk_time_interval => INTERVAL '1 week',
    number_partitions   => 64,
    partitioning_column => 'account_id',
    if_not_exists       => TRUE);

CREATE INDEX idx_accounting_entries_account  ON accounting.entries(account_id, created_at DESC);
CREATE INDEX idx_accounting_entries_journal  ON accounting.entries(journal_id);
CREATE INDEX idx_accounting_entries_currency ON accounting.entries(currency, created_at DESC);
CREATE INDEX idx_accounting_entries_created  ON accounting.entries(created_at DESC);

ALTER TABLE accounting.entries SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'account_id, currency, dr_cr',
    timescaledb.compress_orderby   = 'created_at DESC'
);
SELECT add_compression_policy('accounting.entries', INTERVAL '90 days', if_not_exists => TRUE);

-- ---------------------------------------------------------------------------
-- accounting.ledger_entry_checksums (from extensions)
-- Placed immediately after entries since it references them
-- ---------------------------------------------------------------------------
CREATE TABLE accounting.ledger_entry_checksums (
    entry_id         BIGINT      PRIMARY KEY REFERENCES accounting.entries(id) ON DELETE RESTRICT,
    content_hash     VARCHAR(64) NOT NULL,
    chain_hash       VARCHAR(64) NOT NULL,
    prev_entry_id    BIGINT      NULL REFERENCES accounting.entries(id) ON DELETE RESTRICT,
    integrity_status VARCHAR(20) NOT NULL DEFAULT 'unverified',
    last_verified_at TIMESTAMPTZ NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounting_checksums_prev     ON accounting.ledger_entry_checksums(prev_entry_id) WHERE prev_entry_id IS NOT NULL;
CREATE INDEX idx_accounting_checksums_tampered ON accounting.ledger_entry_checksums(integrity_status)
    WHERE integrity_status = 'tampered';

CREATE OR REPLACE RULE ledger_checksums_no_update AS ON UPDATE TO accounting.ledger_entry_checksums DO INSTEAD NOTHING;
CREATE OR REPLACE RULE ledger_checksums_no_delete AS ON DELETE TO accounting.ledger_entry_checksums DO INSTEAD NOTHING;

-- ---------------------------------------------------------------------------
-- accounting.escrow_holds
-- ---------------------------------------------------------------------------
CREATE TABLE accounting.escrow_holds (
    id                      BIGSERIAL    PRIMARY KEY,
    seller_order_id         BIGINT       NOT NULL UNIQUE,
    store_id                BIGINT       NOT NULL,
    order_id                BIGINT       NOT NULL,
    escrow_account_id       BIGINT       NOT NULL REFERENCES accounting.accounts(id) ON DELETE RESTRICT,
    amount                  BIGINT       NOT NULL,
    commission_amount       BIGINT       NOT NULL DEFAULT 0,
    seller_net_amount       BIGINT       NOT NULL,
    currency                VARCHAR(8)   NOT NULL REFERENCES accounting.currencies(code),
    status                  VARCHAR(25)  NOT NULL DEFAULT 'holding'
                                CHECK (status IN ('holding','released','refunded','disputed','partially_refunded')),
    auto_release_at         TIMESTAMPTZ  NULL,
    released_at             TIMESTAMPTZ  NULL,
    released_by             TEXT         NULL,
    hold_journal_id         BIGINT       NULL REFERENCES accounting.journals(id) ON DELETE SET NULL,
    release_journal_id      BIGINT       NULL REFERENCES accounting.journals(id) ON DELETE SET NULL,
    notes                   TEXT,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounting_escrow_store        ON accounting.escrow_holds(store_id, status);
CREATE INDEX idx_accounting_escrow_auto_release ON accounting.escrow_holds(auto_release_at)
    WHERE status = 'holding' AND auto_release_at IS NOT NULL;
CREATE INDEX idx_accounting_escrow_order        ON accounting.escrow_holds(order_id);

CREATE TRIGGER trg_accounting_escrow_holds_updated_at
    BEFORE UPDATE ON accounting.escrow_holds
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

-- ---------------------------------------------------------------------------
-- accounting.settlement_fx_snapshots (from extensions)
-- Insert-only: rate locked at settlement time, never changed
-- ---------------------------------------------------------------------------
CREATE TABLE accounting.settlement_fx_snapshots (
    id                  BIGSERIAL   PRIMARY KEY,
    seller_order_id     BIGINT      NOT NULL UNIQUE,
    base_currency       VARCHAR(10) NOT NULL,
    settlement_currency VARCHAR(10) NOT NULL,
    -- rate × 1,000,000: USDKES 130.50 → 130500000
    rate                BIGINT      NOT NULL CHECK (rate > 0),
    rate_source         VARCHAR(50) NULL,
    rate_valid_from     TIMESTAMPTZ NOT NULL,
    base_amount         BIGINT      NOT NULL,
    converted_amount    BIGINT      NOT NULL,
    snapshotted_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE RULE settlement_fx_no_update AS ON UPDATE TO accounting.settlement_fx_snapshots DO INSTEAD NOTHING;
CREATE OR REPLACE RULE settlement_fx_no_delete AS ON DELETE TO accounting.settlement_fx_snapshots DO INSTEAD NOTHING;

CREATE INDEX idx_accounting_sfx_seller_order ON accounting.settlement_fx_snapshots(seller_order_id);
CREATE INDEX idx_accounting_sfx_base_curr    ON accounting.settlement_fx_snapshots(base_currency, snapshotted_at DESC);

-- ---------------------------------------------------------------------------
-- accounting.settlements / settlement_items
-- ---------------------------------------------------------------------------
CREATE TABLE accounting.settlements (
    id           BIGSERIAL    PRIMARY KEY,
    store_id     BIGINT       NOT NULL,
    status       VARCHAR(20)  NOT NULL DEFAULT 'pending',
    total_amount BIGINT       NOT NULL,
    currency     VARCHAR(8)   NOT NULL REFERENCES accounting.currencies(code),
    period_start TIMESTAMPTZ  NOT NULL,
    period_end   TIMESTAMPTZ  NOT NULL,
    journal_id   BIGINT       NULL REFERENCES accounting.journals(id) ON DELETE SET NULL,
    notes        TEXT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounting_settlements_store  ON accounting.settlements(store_id, status);
CREATE INDEX idx_accounting_settlements_status ON accounting.settlements(status) WHERE status IN ('pending','processing');

CREATE TRIGGER trg_accounting_settlements_updated_at
    BEFORE UPDATE ON accounting.settlements
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE accounting.settlement_items (
    id              BIGSERIAL PRIMARY KEY,
    settlement_id   BIGINT    NOT NULL REFERENCES accounting.settlements(id) ON DELETE CASCADE,
    seller_order_id BIGINT    NOT NULL UNIQUE,
    amount          BIGINT    NOT NULL
);
CREATE INDEX idx_accounting_settlement_items ON accounting.settlement_items(settlement_id);

-- ---------------------------------------------------------------------------
-- accounting.payout_accounts
-- ---------------------------------------------------------------------------
CREATE TABLE accounting.payout_accounts (
    id                BIGSERIAL    PRIMARY KEY,
    store_id          BIGINT       NOT NULL,
    provider          VARCHAR(30)  NOT NULL,
    payment_method_id INT          NULL,
    account_details   JSONB        NOT NULL,
    is_default        BOOLEAN      NOT NULL DEFAULT FALSE,
    is_verified       BOOLEAN      NOT NULL DEFAULT FALSE,
    verified_at       TIMESTAMPTZ  NULL,
    metadata          JSONB,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ  NULL    -- soft delete
);

CREATE INDEX idx_accounting_payout_accounts_store   ON accounting.payout_accounts(store_id);
CREATE INDEX idx_accounting_payout_accounts_default ON accounting.payout_accounts(store_id)
    WHERE is_default = TRUE AND is_verified = TRUE AND deleted_at IS NULL;

CREATE TRIGGER trg_accounting_payout_accounts_updated_at
    BEFORE UPDATE ON accounting.payout_accounts
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

-- Now that payout_accounts exists, add the deferred FK from store_financial_settings
ALTER TABLE catalog.store_financial_settings
    ADD CONSTRAINT fk_sfs_default_payout_account
    FOREIGN KEY (default_payout_account_id)
    REFERENCES accounting.payout_accounts(id) ON DELETE SET NULL;

-- ---------------------------------------------------------------------------
-- accounting.payout_batches / payouts / payout_items
-- ---------------------------------------------------------------------------
CREATE TABLE accounting.payout_batches (
    id           BIGSERIAL    PRIMARY KEY,
    batch_ref    VARCHAR(100) NOT NULL UNIQUE,
    total_amount BIGINT       NOT NULL,
    currency     VARCHAR(8)   NOT NULL REFERENCES accounting.currencies(code),
    payout_count INT          NOT NULL DEFAULT 0,
    status       VARCHAR(20)  NOT NULL DEFAULT 'pending',
    initiated_by TEXT         NULL,
    processed_at TIMESTAMPTZ  NULL,
    notes        TEXT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_accounting_payout_batches_status ON accounting.payout_batches(status);

CREATE TABLE accounting.payouts (
    id                 BIGSERIAL    PRIMARY KEY,
    store_id           BIGINT       NOT NULL,
    payout_account_id  BIGINT       NOT NULL REFERENCES accounting.payout_accounts(id) ON DELETE RESTRICT,
    batch_id           BIGINT       NULL REFERENCES accounting.payout_batches(id) ON DELETE SET NULL,
    settlement_id      BIGINT       NULL REFERENCES accounting.settlements(id) ON DELETE SET NULL,
    payout_ref         VARCHAR(100) NOT NULL UNIQUE,
    provider           VARCHAR(30)  NOT NULL,
    gross_amount       BIGINT       NOT NULL,
    fee_amount         BIGINT       NOT NULL DEFAULT 0,
    net_amount         BIGINT       NOT NULL,
    currency           VARCHAR(8)   NOT NULL REFERENCES accounting.currencies(code),
    status             VARCHAR(20)  NOT NULL DEFAULT 'pending',
    failure_reason     TEXT,
    provider_reference VARCHAR(255) NULL,
    provider_response  JSONB        NULL,
    initiated_by       TEXT         NULL,
    attempt_count      INT          NOT NULL DEFAULT 0,
    last_attempt_at    TIMESTAMPTZ  NULL,
    completed_at       TIMESTAMPTZ  NULL,
    journal_id         BIGINT       NULL REFERENCES accounting.journals(id) ON DELETE SET NULL,
    metadata           JSONB,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounting_payouts_store   ON accounting.payouts(store_id, created_at DESC);
CREATE INDEX idx_accounting_payouts_status  ON accounting.payouts(status);
CREATE INDEX idx_accounting_payouts_batch   ON accounting.payouts(batch_id);
CREATE INDEX idx_accounting_payouts_pending ON accounting.payouts(status, created_at)
    WHERE status IN ('pending','processing');

CREATE TRIGGER trg_accounting_payouts_updated_at
    BEFORE UPDATE ON accounting.payouts
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE accounting.payout_items (
    id              BIGSERIAL PRIMARY KEY,
    payout_id       BIGINT    NOT NULL REFERENCES accounting.payouts(id) ON DELETE CASCADE,
    seller_order_id BIGINT    NOT NULL UNIQUE,
    amount          BIGINT    NOT NULL
);
CREATE INDEX idx_accounting_payout_items ON accounting.payout_items(payout_id);

-- ---------------------------------------------------------------------------
-- accounting.fee_rules / applied_fees
-- fee_rules includes deleted_at (from extensions soft-delete consistency)
-- ---------------------------------------------------------------------------
CREATE TABLE accounting.fee_rules (
    id                 BIGSERIAL    PRIMARY KEY,
    rule_name          VARCHAR(100) NOT NULL,
    fee_category       VARCHAR(40)  NOT NULL,
    calculation_method VARCHAR(20)  NOT NULL CHECK (calculation_method IN ('percentage','fixed','tiered')),
    rate_bps           INT          NULL,
    fixed_fee          BIGINT       NULL,
    min_fee            BIGINT       NULL,
    max_fee            BIGINT       NULL,
    tiers              JSONB        NULL,
    currency           VARCHAR(8)   NULL REFERENCES accounting.currencies(code),
    store_id           BIGINT       NULL,
    category_id        BIGINT       NULL,
    priority           INT          NOT NULL DEFAULT 0,
    valid_from         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    valid_to           TIMESTAMPTZ  NULL,
    is_active          BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at         TIMESTAMPTZ  NULL,    -- soft delete
    CONSTRAINT chk_fee_rule_has_value CHECK (
        (calculation_method = 'percentage' AND rate_bps IS NOT NULL) OR
        (calculation_method = 'fixed'      AND fixed_fee IS NOT NULL) OR
        (calculation_method = 'tiered'     AND tiers IS NOT NULL)
    ),
    CONSTRAINT chk_min_max CHECK (min_fee IS NULL OR max_fee IS NULL OR min_fee <= max_fee),
    CONSTRAINT chk_valid_dates CHECK (valid_to IS NULL OR valid_from < valid_to)
);

CREATE UNIQUE INDEX idx_accounting_fee_rules_active
    ON accounting.fee_rules(fee_category, priority, store_id, category_id)
    WHERE is_active = TRUE AND valid_to IS NULL AND deleted_at IS NULL;
CREATE INDEX idx_accounting_fee_rules_lookup
    ON accounting.fee_rules(fee_category, priority DESC, valid_from)
    WHERE is_active = TRUE;

CREATE TRIGGER trg_accounting_fee_rules_updated_at
    BEFORE UPDATE ON accounting.fee_rules
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

CREATE TABLE accounting.applied_fees (
    id             BIGSERIAL    PRIMARY KEY,
    journal_id     BIGINT       NOT NULL REFERENCES accounting.journals(id),
    fee_rule_id    BIGINT       NULL REFERENCES accounting.fee_rules(id) ON DELETE SET NULL,
    fee_category   VARCHAR(40)  NOT NULL,
    amount         BIGINT       NOT NULL CHECK (amount >= 0),
    currency       VARCHAR(8)   NOT NULL REFERENCES accounting.currencies(code),
    fee_account_id BIGINT       NULL REFERENCES accounting.accounts(id),
    entry_id       BIGINT       NULL,
    metadata       JSONB,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounting_applied_fees_journal  ON accounting.applied_fees(journal_id);
CREATE INDEX idx_accounting_applied_fees_category ON accounting.applied_fees(fee_category, created_at DESC);

-- ---------------------------------------------------------------------------
-- accounting.fund_locks — TimescaleDB hypertable
-- ---------------------------------------------------------------------------
CREATE TABLE accounting.fund_locks (
    id              BIGSERIAL    NOT NULL,
    lock_key        TEXT         NOT NULL,
    account_id      BIGINT       NOT NULL REFERENCES accounting.accounts(id) ON DELETE CASCADE,
    locked_amount   BIGINT       NOT NULL CHECK (locked_amount > 0),
    currency        VARCHAR(8)   NOT NULL REFERENCES accounting.currencies(code),
    lock_type       TEXT         NOT NULL DEFAULT 'order_escrow',
    reference_id    TEXT         NOT NULL,
    reference_type  TEXT         NOT NULL DEFAULT 'seller_order',
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    released        BOOLEAN      NOT NULL DEFAULT FALSE,
    released_amount BIGINT       NOT NULL DEFAULT 0,
    released_at     TIMESTAMPTZ  NULL,
    released_by     TEXT         NULL,
    release_reason  TEXT         NULL,
    expires_at      TIMESTAMPTZ  NULL,
    auto_release    BOOLEAN      NOT NULL DEFAULT TRUE,
    metadata        JSONB,
    notes           TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_fund_lock_released CHECK (
        (released = FALSE AND released_at IS NULL  AND released_amount = 0) OR
        (released = TRUE  AND released_at IS NOT NULL AND released_amount > 0)
    ),
    CONSTRAINT chk_released_lte_locked CHECK (released_amount <= locked_amount),
    PRIMARY KEY (id, created_at)
);

SELECT create_hypertable('accounting.fund_locks', 'created_at',
    chunk_time_interval => INTERVAL '7 days', if_not_exists => TRUE);

CREATE UNIQUE INDEX idx_accounting_fund_locks_key
    ON accounting.fund_locks(lock_key, created_at);
CREATE INDEX idx_accounting_fund_locks_account
    ON accounting.fund_locks(account_id, created_at DESC) WHERE is_active = TRUE;
CREATE INDEX idx_accounting_fund_locks_reference
    ON accounting.fund_locks(reference_type, reference_id, created_at DESC) WHERE is_active = TRUE;
CREATE INDEX idx_accounting_fund_locks_expires
    ON accounting.fund_locks(expires_at, created_at DESC)
    WHERE is_active = TRUE AND released = FALSE AND expires_at IS NOT NULL;

ALTER TABLE accounting.fund_locks SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'account_id, currency, lock_type',
    timescaledb.compress_orderby   = 'created_at DESC'
);
SELECT add_compression_policy('accounting.fund_locks', INTERVAL '30 days', if_not_exists => TRUE);

CREATE TRIGGER trg_accounting_fund_locks_updated_at
    BEFORE UPDATE ON accounting.fund_locks
    FOR EACH ROW EXECUTE FUNCTION public.fn_set_updated_at();

-- ---------------------------------------------------------------------------
-- accounting.balance_snapshots
-- ---------------------------------------------------------------------------
CREATE TABLE accounting.balance_snapshots (
    id               BIGSERIAL   PRIMARY KEY,
    account_id       BIGINT      NOT NULL REFERENCES accounting.accounts(id) ON DELETE CASCADE,
    snapshot_at      TIMESTAMPTZ NOT NULL,
    balance          BIGINT      NOT NULL,
    computed_balance BIGINT      NOT NULL,
    delta            BIGINT      GENERATED ALWAYS AS (balance - computed_balance) STORED,
    currency         VARCHAR(8)  NOT NULL REFERENCES accounting.currencies(code),
    entry_count      BIGINT      NOT NULL,
    is_reconciled    BOOLEAN     NOT NULL DEFAULT FALSE,
    notes            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (account_id, snapshot_at)
);

CREATE INDEX idx_accounting_balance_snapshots ON accounting.balance_snapshots(account_id, snapshot_at DESC);
CREATE INDEX idx_accounting_balance_snapshots_unreconciled
    ON accounting.balance_snapshots(is_reconciled) WHERE is_reconciled = FALSE;

-- ---------------------------------------------------------------------------
-- accounting.audit_log — TimescaleDB hypertable
-- ---------------------------------------------------------------------------
CREATE TABLE accounting.audit_log (
    id          BIGSERIAL   NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id   TEXT        NOT NULL,
    action      VARCHAR(50) NOT NULL,
    actor_type  VARCHAR(20) NULL,
    actor_id    TEXT        NULL,
    ip_address  INET        NULL,
    old_values  JSONB       NULL,
    new_values  JSONB       NULL,
    metadata    JSONB       NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
);

SELECT create_hypertable('accounting.audit_log', 'created_at',
    chunk_time_interval => INTERVAL '1 month', if_not_exists => TRUE);

CREATE INDEX idx_accounting_audit_entity ON accounting.audit_log(entity_type, entity_id, created_at DESC);
CREATE INDEX idx_accounting_audit_actor  ON accounting.audit_log(actor_type, actor_id, created_at DESC);

ALTER TABLE accounting.audit_log SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'entity_type, actor_type',
    timescaledb.compress_orderby   = 'created_at DESC'
);
SELECT add_compression_policy('accounting.audit_log', INTERVAL '180 days', if_not_exists => TRUE);

-- ---------------------------------------------------------------------------
-- accounting: TimescaleDB continuous aggregates
-- ---------------------------------------------------------------------------
CREATE MATERIALIZED VIEW accounting.txn_stats_hourly
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', e.created_at) AS hour,
    e.currency,
    j.journal_type,
    e.dr_cr,
    COUNT(*)                            AS entry_count,
    SUM(e.amount)                       AS total_amount,
    MIN(e.amount)                       AS min_amount,
    MAX(e.amount)                       AS max_amount
FROM accounting.entries e
JOIN accounting.journals j ON e.journal_id = j.id
GROUP BY hour, e.currency, j.journal_type, e.dr_cr
WITH NO DATA;

SELECT add_continuous_aggregate_policy(
    'accounting.txn_stats_hourly',
    start_offset      => INTERVAL '3 hours',
    end_offset        => INTERVAL '30 minutes',
    schedule_interval => INTERVAL '30 minutes',
    if_not_exists     => TRUE
);

CREATE MATERIALIZED VIEW accounting.settlement_daily
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 day', e.created_at)                              AS day,
    a.owner_id                                                       AS store_id,
    e.currency,
    COUNT(DISTINCT j.id)                                             AS journal_count,
    SUM(e.amount) FILTER (WHERE e.dr_cr = 'CR')                     AS total_credits,
    SUM(e.amount) FILTER (WHERE e.dr_cr = 'DR')                     AS total_debits
FROM accounting.entries e
JOIN accounting.accounts a ON e.account_id = a.id
JOIN accounting.journals j ON e.journal_id = j.id
WHERE a.purpose IN ('settlement','escrow') AND a.owner_type = 'store'
GROUP BY day, a.owner_id, e.currency
WITH NO DATA;

SELECT add_continuous_aggregate_policy(
    'accounting.settlement_daily',
    start_offset      => INTERVAL '2 days',
    end_offset        => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists     => TRUE
);

-- ---------------------------------------------------------------------------
-- accounting: views
-- ---------------------------------------------------------------------------
CREATE OR REPLACE VIEW accounting.v_store_balances AS
SELECT
    a.id               AS account_id,
    a.account_number,
    a.owner_id         AS store_id,
    a.currency,
    a.purpose,
    COALESCE(b.balance,           0) AS balance,
    COALESCE(b.available_balance, 0) AS available_balance,
    COALESCE(b.pending_debit,     0) AS pending_debit,
    COALESCE(b.pending_credit,    0) AS pending_credit,
    a.is_active,
    a.is_locked
FROM accounting.accounts a
LEFT JOIN accounting.balances b ON a.id = b.account_id
WHERE a.owner_type = 'store';

CREATE OR REPLACE VIEW accounting.v_platform_accounts AS
SELECT
    a.id, a.account_number, a.currency, a.purpose,
    COALESCE(b.balance, 0) AS balance,
    a.is_active
FROM accounting.accounts a
LEFT JOIN accounting.balances b ON a.id = b.account_id
WHERE a.owner_type = 'system';

CREATE OR REPLACE VIEW accounting.v_pending_escrow_releases AS
SELECT
    eh.id,
    eh.seller_order_id,
    eh.store_id,
    eh.amount,
    eh.commission_amount,
    eh.seller_net_amount,
    eh.currency,
    eh.auto_release_at,
    EXTRACT(EPOCH FROM (NOW() - eh.auto_release_at)) / 3600 AS hours_overdue
FROM accounting.escrow_holds eh
WHERE eh.status = 'holding'
  AND eh.auto_release_at IS NOT NULL
  AND eh.auto_release_at <= NOW()
ORDER BY eh.auto_release_at ASC;

CREATE OR REPLACE VIEW accounting.v_unreconciled_accounts AS
SELECT
    a.id,
    a.account_number,
    a.owner_type,
    a.owner_id,
    a.currency,
    b.balance,
    b.updated_at AS balance_updated_at,
    s.snapshot_at AS last_snapshot_at,
    s.delta       AS last_reconciliation_delta
FROM accounting.accounts a
JOIN accounting.balances b ON a.id = b.account_id
LEFT JOIN LATERAL (
    SELECT snapshot_at, delta
    FROM accounting.balance_snapshots
    WHERE account_id = a.id
    ORDER BY snapshot_at DESC
    LIMIT 1
) s ON TRUE
WHERE a.is_active = TRUE;

-- ---------------------------------------------------------------------------
-- accounting: functions
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION accounting.generate_payout_ref()
RETURNS TEXT AS $$
BEGIN
    RETURN 'PAY-' || TO_CHAR(NOW(), 'YYYYMMDD') || '-' ||
           UPPER(SUBSTR(MD5(NOW()::TEXT || random()::TEXT), 1, 8));
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION accounting.get_locked_amount(p_account_id BIGINT)
RETURNS BIGINT AS $$
BEGIN
    RETURN COALESCE(
        (SELECT SUM(locked_amount - released_amount)
         FROM accounting.fund_locks
         WHERE account_id = p_account_id
           AND is_active = TRUE
           AND released = FALSE
           AND created_at > NOW() - INTERVAL '90 days'),
        0
    );
END;
$$ LANGUAGE plpgsql STABLE;

COMMIT;
