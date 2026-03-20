-- Migration: 001_initial_schema
-- Description: Initial database schema for the Nucleus Remote Access Portal
-- Idempotent: Uses CREATE TABLE IF NOT EXISTS throughout

-- ---------------------------------------------------------------------------
-- Tenants
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS tenants (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(100) NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Sites
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sites (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    location    VARCHAR(500),
    timezone    VARCHAR(100) DEFAULT 'UTC',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Users
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email           VARCHAR(320) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    display_name    VARCHAR(255) NOT NULL,
    role            VARCHAR(50)  NOT NULL DEFAULT 'operator'
                        CHECK (role IN ('admin', 'operator', 'viewer', 'support')),
    is_active       BOOLEAN      NOT NULL DEFAULT true,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Devices
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS devices (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    site_id              UUID         REFERENCES sites(id) ON DELETE SET NULL,
    device_id            VARCHAR(50)  NOT NULL UNIQUE,
    display_name         VARCHAR(255),
    status               VARCHAR(20)  NOT NULL DEFAULT 'unknown'
                             CHECK (status IN ('online', 'offline', 'unknown', 'maintenance')),
    last_seen            TIMESTAMPTZ,
    firmware_version     VARCHAR(50),
    ip_address           INET,
    hardware_model       VARCHAR(100),
    serial_number        VARCHAR(100),
    tags                 JSONB        DEFAULT '{}',
    inventory_hash       VARCHAR(64),
    inventory_updated_at TIMESTAMPTZ,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Endpoints
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS endpoints (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id     UUID         NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    type          VARCHAR(20)  NOT NULL CHECK (type IN ('WEB', 'PROGRAM', 'BRIDGE')),
    port          INTEGER      NOT NULL CHECK (port > 0 AND port < 65536),
    label         VARCHAR(100) NOT NULL,
    protocol      VARCHAR(20)  DEFAULT 'tcp',
    description   TEXT,
    enabled       BOOLEAN      NOT NULL DEFAULT true,
    discovered_at TIMESTAMPTZ  DEFAULT NOW(),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (device_id, port)
);

-- ---------------------------------------------------------------------------
-- Bridge Profiles
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS bridge_profiles (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       UUID        NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    serial_port     VARCHAR(100) NOT NULL,
    baud_rate       INTEGER     NOT NULL DEFAULT 9600,
    parity          CHAR(1)     DEFAULT 'N',
    stop_bits       SMALLINT    DEFAULT 1,
    data_bits       SMALLINT    DEFAULT 8,
    tcp_port        INTEGER,
    status          VARCHAR(20) DEFAULT 'idle'
                        CHECK (status IN ('idle', 'active', 'error')),
    last_started_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Sessions
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sessions (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id            UUID        NOT NULL REFERENCES devices(id),
    endpoint_id          UUID        REFERENCES endpoints(id),
    user_id              UUID        NOT NULL REFERENCES users(id),
    tenant_id            UUID        NOT NULL REFERENCES tenants(id),
    status               VARCHAR(20) NOT NULL DEFAULT 'active'
                             CHECK (status IN ('active', 'expired', 'stopped', 'error')),
    local_port           INTEGER,
    remote_port          INTEGER,
    delivery_mode        VARCHAR(20) DEFAULT 'export'
                             CHECK (delivery_mode IN ('web', 'export')),
    ttl_seconds          INTEGER     NOT NULL DEFAULT 3600,
    idle_timeout_seconds INTEGER     DEFAULT 1800,
    started_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at           TIMESTAMPTZ NOT NULL,
    last_activity_at     TIMESTAMPTZ,
    stopped_at           TIMESTAMPTZ,
    stop_reason          VARCHAR(100),
    tunnel_url           TEXT,
    audit_data           JSONB       DEFAULT '{}',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Export History
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS export_history (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id       UUID        REFERENCES sessions(id),
    user_id          UUID        NOT NULL REFERENCES users(id),
    device_id        UUID        NOT NULL REFERENCES devices(id),
    endpoint_id      UUID        REFERENCES endpoints(id),
    tenant_id        UUID        NOT NULL REFERENCES tenants(id),
    site_id          UUID        REFERENCES sites(id),
    started_at       TIMESTAMPTZ NOT NULL,
    stopped_at       TIMESTAMPTZ,
    stop_reason      VARCHAR(100),
    local_bind_port  INTEGER,
    delivery_mode    VARCHAR(20),
    duration_seconds INTEGER,
    bytes_transferred BIGINT     DEFAULT 0,
    metadata         JSONB       DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Audit Logs
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS audit_logs (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID         NOT NULL REFERENCES tenants(id),
    user_id       UUID         REFERENCES users(id),
    action        VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50)  NOT NULL,
    resource_id   UUID,
    ip_address    INET,
    user_agent    TEXT,
    metadata      JSONB        DEFAULT '{}',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Indexes for performance
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_sites_tenant         ON sites(tenant_id);

CREATE INDEX IF NOT EXISTS idx_users_tenant         ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_email          ON users(email);

CREATE INDEX IF NOT EXISTS idx_devices_tenant       ON devices(tenant_id);
CREATE INDEX IF NOT EXISTS idx_devices_device_id    ON devices(device_id);
CREATE INDEX IF NOT EXISTS idx_devices_status       ON devices(status);
CREATE INDEX IF NOT EXISTS idx_devices_site         ON devices(site_id);

CREATE INDEX IF NOT EXISTS idx_endpoints_device     ON endpoints(device_id);
CREATE INDEX IF NOT EXISTS idx_endpoints_type       ON endpoints(type);

CREATE INDEX IF NOT EXISTS idx_bridge_profiles_device ON bridge_profiles(device_id);

CREATE INDEX IF NOT EXISTS idx_sessions_device      ON sessions(device_id);
CREATE INDEX IF NOT EXISTS idx_sessions_user        ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_tenant      ON sessions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sessions_status      ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_started     ON sessions(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_expires     ON sessions(expires_at);

CREATE INDEX IF NOT EXISTS idx_export_history_user    ON export_history(user_id);
CREATE INDEX IF NOT EXISTS idx_export_history_device  ON export_history(device_id);
CREATE INDEX IF NOT EXISTS idx_export_history_tenant  ON export_history(tenant_id);
CREATE INDEX IF NOT EXISTS idx_export_history_started ON export_history(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_export_history_session ON export_history(session_id);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant    ON audit_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user      ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created   ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action    ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource  ON audit_logs(resource_type, resource_id);
