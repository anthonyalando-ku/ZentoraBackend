-- ============================================
-- REUSABLE AUTHENTICATION MODULE (Production-Ready)
-- ============================================
\c zentora;
-- Taska database

BEGIN;

-- ============================================
-- CUSTOM TYPES
-- ============================================
CREATE TYPE role_type AS ENUM ('admin', 'user', 'super_admin');
CREATE TYPE session_status AS ENUM ('active', 'expired', 'revoked');
CREATE TYPE notification_type AS ENUM ('system', 'alert', 'info');
CREATE TYPE auth_provider_type AS ENUM ('local', 'google', 'facebook', 'apple', 'github', 'microsoft');
CREATE TYPE account_status AS ENUM ('active', 'inactive', 'suspended', 'pending_verification');

-- ============================================
-- CORE AUTHENTICATION TABLES (Separated Concerns)
-- ============================================

-- Core Identity Table (Minimal, Provider-Agnostic)
CREATE TABLE auth_identities (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NULL, -- Nullable for phone-only auth
    email_verified BOOLEAN DEFAULT FALSE,
    email_verified_at TIMESTAMP NULL,
    phone VARCHAR(20) NULL, -- Nullable for email-only auth
    phone_verified BOOLEAN DEFAULT FALSE,
    phone_verified_at TIMESTAMP NULL,
    status account_status DEFAULT 'pending_verification',
    last_login TIMESTAMP NULL,
    failed_login_attempts INT DEFAULT 0,
    locked_until TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT check_email_or_phone CHECK (email IS NOT NULL OR phone IS NOT NULL)
);

-- User Profiles (Business Data - Separate from Auth)
CREATE TABLE user_profiles (
    id BIGSERIAL PRIMARY KEY,
    identity_id BIGINT UNIQUE NOT NULL,
    full_name VARCHAR(255) NULL,
    avatar_url VARCHAR(500) NULL,
    bio TEXT NULL,
    metadata JSONB NULL, -- Additional flexible fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (identity_id) REFERENCES auth_identities(id) ON DELETE CASCADE
);

-- Authentication Providers (Supports Multiple Auth Methods)
CREATE TABLE auth_providers (
    id BIGSERIAL PRIMARY KEY,
    identity_id BIGINT NOT NULL,
    provider auth_provider_type NOT NULL,
    provider_user_id VARCHAR(255) NULL, -- External provider's user ID (Google ID, Facebook ID, etc.)
    provider_email VARCHAR(255) NULL,
    provider_username VARCHAR(255) NULL,
    password_hash VARCHAR(255) NULL, -- Only for 'local' provider
    access_token TEXT NULL, -- OAuth access token
    refresh_token TEXT NULL, -- OAuth refresh token
    token_expires_at TIMESTAMP NULL,
    provider_data JSONB NULL, -- Store additional provider-specific data
    is_primary BOOLEAN DEFAULT FALSE, -- One primary auth method per identity
    password_changed_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (identity_id) REFERENCES auth_identities(id) ON DELETE CASCADE,
    UNIQUE(provider, provider_user_id),
    CONSTRAINT check_local_password CHECK (
        (provider = 'local' AND password_hash IS NOT NULL) OR 
        (provider != 'local' AND password_hash IS NULL)
    )
);

-- Verification & Reset Tokens
CREATE TABLE auth_verification_tokens (
    id BIGSERIAL PRIMARY KEY,
    identity_id BIGINT NOT NULL,
    token_type VARCHAR(20) NOT NULL, -- 'password_reset', 'email_verify', 'phone_verify'
    token VARCHAR(255) NOT NULL,
    code VARCHAR(10) NULL, -- For OTP
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP NULL,
    verified_at TIMESTAMP NULL,
    attempts INT DEFAULT 0,
    metadata JSONB NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (identity_id) REFERENCES auth_identities(id) ON DELETE CASCADE
);

-- Roles Definition
CREATE TABLE auth_roles (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL, -- Changed from role_type ENUM for flexibility
    display_name VARCHAR(100) NOT NULL,
    description TEXT NULL,
    is_system BOOLEAN DEFAULT FALSE, -- System roles can't be deleted
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User Roles Assignment
CREATE TABLE auth_identity_roles (
    id BIGSERIAL PRIMARY KEY,
    identity_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    assigned_by BIGINT NULL,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NULL,
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB NULL,
    FOREIGN KEY (identity_id) REFERENCES auth_identities(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES auth_roles(id) ON DELETE CASCADE,
    FOREIGN KEY (assigned_by) REFERENCES auth_identities(id) ON DELETE SET NULL,
    UNIQUE(identity_id, role_id)
);

-- Permissions Definition
CREATE TABLE auth_permissions (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    display_name VARCHAR(150) NOT NULL,
    description TEXT NULL,
    resource VARCHAR(50) NULL, -- What resource (users, posts, etc.)
    action VARCHAR(50) NULL, -- What action (create, read, update, delete)
    conditions JSONB NULL, -- Advanced: conditional permissions
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Role Permissions
CREATE TABLE auth_role_permissions (
    id BIGSERIAL PRIMARY KEY,
    role_id BIGINT NOT NULL,
    permission_id BIGINT NOT NULL,
    granted_by BIGINT NULL,
    granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (role_id) REFERENCES auth_roles(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES auth_permissions(id) ON DELETE CASCADE,
    FOREIGN KEY (granted_by) REFERENCES auth_identities(id) ON DELETE SET NULL,
    UNIQUE(role_id, permission_id)
);

-- User-specific Permissions Override
CREATE TABLE auth_identity_permissions (
    id BIGSERIAL PRIMARY KEY,
    identity_id BIGINT NOT NULL,
    permission_id BIGINT NOT NULL,
    is_granted BOOLEAN DEFAULT TRUE,
    granted_by BIGINT NULL,
    granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NULL,
    metadata JSONB NULL,
    FOREIGN KEY (identity_id) REFERENCES auth_identities(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES auth_permissions(id) ON DELETE CASCADE,
    FOREIGN KEY (granted_by) REFERENCES auth_identities(id) ON DELETE SET NULL,
    UNIQUE(identity_id, permission_id)
);

-- Sessions Management
CREATE TABLE auth_sessions (
    id BIGSERIAL PRIMARY KEY,
    identity_id BIGINT NOT NULL,
    session_token VARCHAR(500) UNIQUE NOT NULL,
    refresh_token VARCHAR(500) UNIQUE NULL,
    provider auth_provider_type NOT NULL, -- Track which provider was used for this session
    ip_address INET NULL, -- Changed to INET for better IP handling
    user_agent TEXT NULL,
    device_id VARCHAR(255) NULL,
    device_name VARCHAR(255) NULL,
    device_fingerprint VARCHAR(500) NULL,
    status session_status DEFAULT 'active',
    login_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    logout_at TIMESTAMP NULL,
    metadata JSONB NULL,
    FOREIGN KEY (identity_id) REFERENCES auth_identities(id) ON DELETE CASCADE
);

-- Audit Trail
CREATE TABLE auth_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    identity_id BIGINT NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NULL,
    resource_id BIGINT NULL,
    old_values JSONB NULL,
    new_values JSONB NULL,
    ip_address INET NULL,
    user_agent TEXT NULL,
    status VARCHAR(20) NULL,
    error_message TEXT NULL,
    metadata JSONB NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (identity_id) REFERENCES auth_identities(id) ON DELETE SET NULL
);

-- Notifications
CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,
    identity_id BIGINT NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    type notification_type DEFAULT 'system',
    metadata JSONB NULL,
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    read_at TIMESTAMP NULL,
    expires_at TIMESTAMP NULL,
    FOREIGN KEY (identity_id) REFERENCES auth_identities(id) ON DELETE CASCADE
);

-- ============================================
-- INDEXES (Optimized for Performance)
-- ============================================

-- auth_identities indexes
CREATE UNIQUE INDEX idx_identities_email_active ON auth_identities(LOWER(email)) WHERE deleted_at IS NULL AND email IS NOT NULL;
CREATE UNIQUE INDEX idx_identities_phone_active ON auth_identities(phone) WHERE deleted_at IS NULL AND phone IS NOT NULL;
CREATE INDEX idx_identities_status ON auth_identities(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_identities_last_login ON auth_identities(last_login) WHERE deleted_at IS NULL;

-- user_profiles indexes
CREATE INDEX idx_profiles_identity ON user_profiles(identity_id);
CREATE INDEX idx_profiles_full_name ON user_profiles(full_name) WHERE full_name IS NOT NULL;

-- auth_providers indexes
CREATE INDEX idx_providers_identity ON auth_providers(identity_id);
CREATE INDEX idx_providers_type ON auth_providers(provider);
CREATE INDEX idx_providers_email ON auth_providers(LOWER(provider_email)) WHERE provider_email IS NOT NULL;
CREATE INDEX idx_providers_primary ON auth_providers(identity_id, is_primary) WHERE is_primary = TRUE;

-- auth_verification_tokens indexes
CREATE INDEX idx_verification_identity ON auth_verification_tokens(identity_id);
CREATE INDEX idx_verification_token ON auth_verification_tokens(token) WHERE used_at IS NULL;
CREATE INDEX idx_verification_type_expires ON auth_verification_tokens(token_type, expires_at) WHERE used_at IS NULL;

-- auth_identity_roles indexes
CREATE INDEX idx_identity_roles_identity ON auth_identity_roles(identity_id) WHERE is_active = TRUE;
CREATE INDEX idx_identity_roles_role ON auth_identity_roles(role_id) WHERE is_active = TRUE;
CREATE INDEX idx_identity_roles_expires ON auth_identity_roles(expires_at) WHERE is_active = TRUE AND expires_at IS NOT NULL;

-- auth_role_permissions indexes
CREATE INDEX idx_role_permissions_role ON auth_role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission ON auth_role_permissions(permission_id);

-- auth_identity_permissions indexes
CREATE INDEX idx_identity_permissions_identity ON auth_identity_permissions(identity_id);
CREATE INDEX idx_identity_permissions_permission ON auth_identity_permissions(permission_id);
CREATE INDEX idx_identity_permissions_expires ON auth_identity_permissions(expires_at) WHERE expires_at IS NOT NULL;

-- auth_sessions indexes
CREATE INDEX idx_sessions_identity ON auth_sessions(identity_id) WHERE status = 'active';
CREATE INDEX idx_sessions_token ON auth_sessions(session_token) WHERE status = 'active';
CREATE INDEX idx_sessions_refresh ON auth_sessions(refresh_token) WHERE status = 'active' AND refresh_token IS NOT NULL;
CREATE INDEX idx_sessions_status_expires ON auth_sessions(status, expires_at);
CREATE INDEX idx_sessions_device ON auth_sessions(device_id) WHERE device_id IS NOT NULL;

-- auth_audit_logs indexes (Partitioning recommended for production)
CREATE INDEX idx_audit_identity_created ON auth_audit_logs(identity_id, created_at DESC);
CREATE INDEX idx_audit_action ON auth_audit_logs(action, created_at DESC);
CREATE INDEX idx_audit_resource ON auth_audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_created ON auth_audit_logs(created_at DESC);

-- notifications indexes
CREATE INDEX idx_notifications_identity_unread ON notifications(identity_id, is_read, created_at DESC);
CREATE INDEX idx_notifications_type ON notifications(type, created_at DESC);

-- ============================================
-- TRIGGERS
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_auth_identities_updated_at 
    BEFORE UPDATE ON auth_identities
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_profiles_updated_at 
    BEFORE UPDATE ON user_profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_auth_providers_updated_at 
    BEFORE UPDATE ON auth_providers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_auth_roles_updated_at 
    BEFORE UPDATE ON auth_roles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- HELPER FUNCTIONS (Updated for new schema)
-- ============================================

-- Check if user has permission
CREATE OR REPLACE FUNCTION auth_user_has_permission(
    p_identity_id BIGINT,
    p_permission_name VARCHAR
)
RETURNS BOOLEAN AS $$
DECLARE
    has_perm BOOLEAN;
BEGIN
    SELECT COALESCE(
        (SELECT is_granted 
         FROM auth_identity_permissions ip
         JOIN auth_permissions p ON ip.permission_id = p.id
         WHERE ip.identity_id = p_identity_id 
           AND p.name = p_permission_name
           AND p.is_active = TRUE
           AND (ip.expires_at IS NULL OR ip.expires_at > NOW())
         LIMIT 1),
        EXISTS(
            SELECT 1
            FROM auth_identity_roles ir
            JOIN auth_role_permissions rp ON ir.role_id = rp.role_id
            JOIN auth_permissions p ON rp.permission_id = p.id
            WHERE ir.identity_id = p_identity_id
              AND p.name = p_permission_name
              AND p.is_active = TRUE
              AND ir.is_active = TRUE
              AND (ir.expires_at IS NULL OR ir.expires_at > NOW())
        )
    ) INTO has_perm;
    
    RETURN COALESCE(has_perm, FALSE);
END;
$$ LANGUAGE plpgsql;

-- Get user roles
CREATE OR REPLACE FUNCTION auth_get_user_roles(p_identity_id BIGINT)
RETURNS TABLE(role_name VARCHAR, display_name VARCHAR) AS $$
BEGIN
    RETURN QUERY
    SELECT r.name, r.display_name
    FROM auth_identity_roles ir
    JOIN auth_roles r ON ir.role_id = r.id
    WHERE ir.identity_id = p_identity_id
      AND ir.is_active = TRUE
      AND r.is_active = TRUE
      AND (ir.expires_at IS NULL OR ir.expires_at > NOW());
END;
$$ LANGUAGE plpgsql;

-- Get user permissions
CREATE OR REPLACE FUNCTION auth_get_user_permissions(p_identity_id BIGINT)
RETURNS TABLE(permission_name VARCHAR, resource VARCHAR, action VARCHAR) AS $$
BEGIN
    RETURN QUERY
    SELECT DISTINCT p.name, p.resource, p.action
    FROM auth_permissions p
    WHERE 
        p.is_active = TRUE
        AND (
            p.id IN (
                SELECT rp.permission_id
                FROM auth_identity_roles ir
                JOIN auth_role_permissions rp ON ir.role_id = rp.role_id
                WHERE ir.identity_id = p_identity_id
                  AND ir.is_active = TRUE
                  AND (ir.expires_at IS NULL OR ir.expires_at > NOW())
            )
            OR p.id IN (
                SELECT ip.permission_id
                FROM auth_identity_permissions ip
                WHERE ip.identity_id = p_identity_id
                  AND ip.is_granted = TRUE
                  AND (ip.expires_at IS NULL OR ip.expires_at > NOW())
            )
        )
        AND p.id NOT IN (
            SELECT ip.permission_id
            FROM auth_identity_permissions ip
            WHERE ip.identity_id = p_identity_id
              AND ip.is_granted = FALSE
              AND (ip.expires_at IS NULL OR ip.expires_at > NOW())
        );
END;
$$ LANGUAGE plpgsql;

-- Clean expired sessions
CREATE OR REPLACE FUNCTION auth_clean_expired_sessions()
RETURNS INTEGER AS $$
DECLARE
    affected_rows INTEGER;
BEGIN
    UPDATE auth_sessions
    SET status = 'expired'
    WHERE status = 'active'
      AND expires_at < NOW();
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows;
END;
$$ LANGUAGE plpgsql;

-- Get or create identity (for OAuth flows)
CREATE OR REPLACE FUNCTION auth_get_or_create_identity(
    p_email VARCHAR DEFAULT NULL,
    p_phone VARCHAR DEFAULT NULL,
    p_provider auth_provider_type DEFAULT 'local',
    p_provider_user_id VARCHAR DEFAULT NULL,
    p_full_name VARCHAR DEFAULT NULL
)
RETURNS BIGINT AS $$
DECLARE
    v_identity_id BIGINT;
    v_provider_id BIGINT;
BEGIN
    -- Try to find existing identity
    IF p_email IS NOT NULL THEN
        SELECT id INTO v_identity_id
        FROM auth_identities
        WHERE LOWER(email) = LOWER(p_email)
          AND deleted_at IS NULL
        LIMIT 1;
    ELSIF p_phone IS NOT NULL THEN
        SELECT id INTO v_identity_id
        FROM auth_identities
        WHERE phone = p_phone
          AND deleted_at IS NULL
        LIMIT 1;
    END IF;
    
    -- Create new identity if not found
    IF v_identity_id IS NULL THEN
        INSERT INTO auth_identities (email, phone, email_verified, status)
        VALUES (
            p_email, 
            p_phone,
            CASE WHEN p_provider != 'local' THEN TRUE ELSE FALSE END,
            CASE WHEN p_provider != 'local' THEN 'active'::account_status ELSE 'pending_verification'::account_status END
        )
        RETURNING id INTO v_identity_id;
        
        -- Create user profile
        INSERT INTO user_profiles (identity_id, full_name)
        VALUES (v_identity_id, p_full_name);
    END IF;
    
    -- Check if provider exists
    SELECT id INTO v_provider_id
    FROM auth_providers
    WHERE identity_id = v_identity_id
      AND provider = p_provider
      AND (provider_user_id = p_provider_user_id OR provider_user_id IS NULL)
    LIMIT 1;
    
    -- Create provider if not exists
    IF v_provider_id IS NULL THEN
        INSERT INTO auth_providers (identity_id, provider, provider_user_id, provider_email, is_primary)
        VALUES (
            v_identity_id, 
            p_provider, 
            p_provider_user_id,
            p_email,
            NOT EXISTS(SELECT 1 FROM auth_providers WHERE identity_id = v_identity_id)
        );
    END IF;
    
    RETURN v_identity_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- SEED DEFAULT ROLES
-- ============================================
INSERT INTO auth_roles (name, display_name, description, is_system) VALUES
('super_admin', 'Super Administrator', 'Full system access with all permissions', TRUE),
('admin', 'Administrator', 'Administrative access to manage system', TRUE),
('user', 'User', 'Standard user access', TRUE);

-- ============================================
-- SEED DEFAULT PERMISSIONS
-- ============================================
INSERT INTO auth_permissions (name, display_name, resource, action) VALUES
('users.create', 'Create Users', 'users', 'create'),
('users.read', 'View Users', 'users', 'read'),
('users.update', 'Update Users', 'users', 'update'),
('users.delete', 'Delete Users', 'users', 'delete'),
('settings.read', 'View Settings', 'settings', 'read'),
('settings.update', 'Update Settings', 'settings', 'update'),
('reports.view', 'View Reports', 'reports', 'read'),
('reports.export', 'Export Reports', 'reports', 'export');

-- Super Admin gets all permissions
INSERT INTO auth_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM auth_roles r
CROSS JOIN auth_permissions p
WHERE r.name = 'super_admin';

-- Admin permissions
INSERT INTO auth_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM auth_roles r
CROSS JOIN auth_permissions p
WHERE r.name = 'admin'
  AND p.name IN ('users.read', 'users.update', 'reports.view', 'reports.export');

COMMIT;