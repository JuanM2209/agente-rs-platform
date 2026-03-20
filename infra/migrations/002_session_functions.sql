-- Migration: 002_session_functions
-- Description: PostgreSQL utility functions and triggers for the Nucleus Remote Access Portal
-- Idempotent: Uses CREATE OR REPLACE FUNCTION throughout

-- ---------------------------------------------------------------------------
-- Function: update_updated_at()
-- Trigger function that sets updated_at to NOW() on every UPDATE.
-- Attach to any table that has an updated_at column.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

-- Attach trigger to tenants
DROP TRIGGER IF EXISTS trg_tenants_updated_at ON tenants;
CREATE TRIGGER trg_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- Attach trigger to users
DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- Attach trigger to devices
DROP TRIGGER IF EXISTS trg_devices_updated_at ON devices;
CREATE TRIGGER trg_devices_updated_at
    BEFORE UPDATE ON devices
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- ---------------------------------------------------------------------------
-- Function: expire_stale_sessions()
-- Marks any session with status='active' and expires_at < NOW() as 'expired'.
-- Sets stopped_at and stop_reason if not already set.
-- Returns the number of sessions expired.
-- Call periodically via pg_cron or an application scheduler.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION expire_stale_sessions()
RETURNS INTEGER
LANGUAGE plpgsql
AS $$
DECLARE
    expired_count INTEGER;
BEGIN
    UPDATE sessions
    SET
        status      = 'expired',
        stopped_at  = COALESCE(stopped_at, NOW()),
        stop_reason = COALESCE(stop_reason, 'ttl_expired')
    WHERE
        status     = 'active'
        AND expires_at < NOW();

    GET DIAGNOSTICS expired_count = ROW_COUNT;

    -- Write an audit entry for the bulk expiry so operators can see it
    IF expired_count > 0 THEN
        INSERT INTO audit_logs (
            tenant_id,
            user_id,
            action,
            resource_type,
            metadata
        )
        SELECT DISTINCT
            s.tenant_id,
            NULL,                               -- system action, no user
            'session.expired',
            'session',
            jsonb_build_object(
                'reason',          'ttl_expired',
                'expired_session', s.id,
                'device_id',       s.device_id,
                'expired_at',      NOW()
            )
        FROM sessions s
        WHERE
            s.status      = 'expired'
            AND s.stopped_at >= NOW() - INTERVAL '5 seconds';
    END IF;

    RETURN expired_count;
END;
$$;

-- ---------------------------------------------------------------------------
-- Function: update_device_status(stale_threshold_minutes INTEGER DEFAULT 5)
-- Sets device status to 'offline' for devices that have not sent a heartbeat
-- within stale_threshold_minutes minutes, unless they are in 'maintenance'.
-- Returns the number of devices whose status changed.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION update_device_status(
    stale_threshold_minutes INTEGER DEFAULT 5
)
RETURNS INTEGER
LANGUAGE plpgsql
AS $$
DECLARE
    updated_count INTEGER;
    stale_cutoff  TIMESTAMPTZ;
BEGIN
    stale_cutoff := NOW() - (stale_threshold_minutes || ' minutes')::INTERVAL;

    -- Mark devices as offline when last_seen has passed the stale threshold
    UPDATE devices
    SET
        status     = 'offline',
        updated_at = NOW()
    WHERE
        status    = 'online'
        AND (last_seen IS NULL OR last_seen < stale_cutoff);

    GET DIAGNOSTICS updated_count = ROW_COUNT;

    -- Mark devices as online when last_seen is recent but they were flagged offline
    UPDATE devices
    SET
        status     = 'online',
        updated_at = NOW()
    WHERE
        status   = 'offline'
        AND last_seen IS NOT NULL
        AND last_seen >= stale_cutoff;

    -- Add those transitions to the count as well
    updated_count := updated_count + (SELECT COUNT(*) FROM devices WHERE status = 'online' AND updated_at >= NOW() - INTERVAL '1 second');

    RETURN updated_count;
END;
$$;

-- ---------------------------------------------------------------------------
-- Function: cleanup_old_audit_logs(retention_days INTEGER DEFAULT 90)
-- Deletes audit log entries older than retention_days days.
-- Returns the number of rows deleted.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION cleanup_old_audit_logs(
    retention_days INTEGER DEFAULT 90
)
RETURNS INTEGER
LANGUAGE plpgsql
AS $$
DECLARE
    deleted_count INTEGER;
    cutoff        TIMESTAMPTZ;
BEGIN
    cutoff := NOW() - (retention_days || ' days')::INTERVAL;

    DELETE FROM audit_logs
    WHERE created_at < cutoff;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;

    RETURN deleted_count;
END;
$$;

-- ---------------------------------------------------------------------------
-- Function: get_tenant_session_stats(p_tenant_id UUID)
-- Returns a summary row with active/expired/stopped session counts and
-- total bytes transferred for the given tenant.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION get_tenant_session_stats(p_tenant_id UUID)
RETURNS TABLE (
    active_sessions   BIGINT,
    expired_sessions  BIGINT,
    stopped_sessions  BIGINT,
    error_sessions    BIGINT,
    total_bytes       BIGINT,
    total_duration_s  BIGINT
)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*) FILTER (WHERE s.status = 'active')                      AS active_sessions,
        COUNT(*) FILTER (WHERE s.status = 'expired')                     AS expired_sessions,
        COUNT(*) FILTER (WHERE s.status = 'stopped')                     AS stopped_sessions,
        COUNT(*) FILTER (WHERE s.status = 'error')                       AS error_sessions,
        COALESCE(SUM(eh.bytes_transferred), 0)::BIGINT                   AS total_bytes,
        COALESCE(SUM(eh.duration_seconds), 0)::BIGINT                    AS total_duration_s
    FROM sessions s
    LEFT JOIN export_history eh ON eh.session_id = s.id
    WHERE s.tenant_id = p_tenant_id;
END;
$$;
