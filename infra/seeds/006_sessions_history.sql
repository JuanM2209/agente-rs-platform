-- Seed: 006_sessions_history
-- Description: Sample active sessions and export history for demo/testing
-- Idempotent: ON CONFLICT (id) DO NOTHING for both tables

-- ---------------------------------------------------------------------------
-- Active Sessions
-- 3 sessions open right now across N-1001 and N-1002 for Alpha tenant users
-- ---------------------------------------------------------------------------
INSERT INTO sessions (
    id,
    device_id,
    endpoint_id,
    user_id,
    tenant_id,
    status,
    local_port,
    remote_port,
    delivery_mode,
    ttl_seconds,
    idle_timeout_seconds,
    started_at,
    expires_at,
    last_activity_at,
    tunnel_url,
    audit_data,
    created_at
)
VALUES
    -- Session 1: admin@alpha.com on N-1001 port 80 (WEB)
    (
        'a1000000-0000-0000-0006-000000000001',
        'a1000000-0000-0000-0003-000000000001',
        'a1000000-0000-0000-0004-100100000001',
        'a1000000-0000-0000-0001-000000000001',
        'a1000000-0000-0000-0000-000000000001',
        'active',
        58001,
        80,
        'web',
        3600,
        1800,
        NOW() - INTERVAL '20 minutes',
        NOW() + INTERVAL '40 minutes',
        NOW() - INTERVAL '2 minutes',
        'https://tunnel-a1001-web.nucleus.local',
        '{"initiated_by": "dashboard", "browser": "Chrome"}',
        NOW() - INTERVAL '20 minutes'
    ),
    -- Session 2: operator@alpha.com on N-1001 port 502 (Modbus TCP — export)
    (
        'a1000000-0000-0000-0006-000000000002',
        'a1000000-0000-0000-0003-000000000001',
        'a1000000-0000-0000-0004-100100000003',
        'a1000000-0000-0000-0001-000000000002',
        'a1000000-0000-0000-0000-000000000001',
        'active',
        58002,
        502,
        'export',
        7200,
        3600,
        NOW() - INTERVAL '45 minutes',
        NOW() + INTERVAL '75 minutes',
        NOW() - INTERVAL '5 minutes',
        NULL,
        '{"initiated_by": "cli", "client_os": "Windows"}',
        NOW() - INTERVAL '45 minutes'
    ),
    -- Session 3: operator@alpha.com on N-1002 port 9090 (Device UI)
    (
        'a1000000-0000-0000-0006-000000000003',
        'a1000000-0000-0000-0003-000000000002',
        'a1000000-0000-0000-0004-100200000002',
        'a1000000-0000-0000-0001-000000000002',
        'a1000000-0000-0000-0000-000000000001',
        'active',
        58003,
        9090,
        'web',
        3600,
        1800,
        NOW() - INTERVAL '10 minutes',
        NOW() + INTERVAL '50 minutes',
        NOW() - INTERVAL '1 minute',
        'https://tunnel-a1002-ui.nucleus.local',
        '{"initiated_by": "dashboard", "browser": "Firefox"}',
        NOW() - INTERVAL '10 minutes'
    )
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Export History — 10 records spanning the last 30 days
-- Mix of stop reasons: ttl_expired, user_stopped, idle_timeout
-- ---------------------------------------------------------------------------
INSERT INTO export_history (
    id,
    session_id,
    user_id,
    device_id,
    endpoint_id,
    tenant_id,
    site_id,
    started_at,
    stopped_at,
    stop_reason,
    local_bind_port,
    delivery_mode,
    duration_seconds,
    bytes_transferred,
    metadata,
    created_at
)
VALUES
    -- Record 1: admin on N-1001 port 80, 28 days ago, user_stopped
    (
        'a1000000-0000-0000-0007-000000000001',
        NULL,
        'a1000000-0000-0000-0001-000000000001',
        'a1000000-0000-0000-0003-000000000001',
        'a1000000-0000-0000-0004-100100000001',
        'a1000000-0000-0000-0000-000000000001',
        'a1000000-0000-0000-0002-000000000001',
        NOW() - INTERVAL '28 days',
        NOW() - INTERVAL '28 days' + INTERVAL '22 minutes',
        'user_stopped',
        57100,
        'web',
        1320,
        245760,
        '{"reason_detail": "task completed"}',
        NOW() - INTERVAL '28 days'
    ),
    -- Record 2: operator on N-1002 port 502, 26 days ago, ttl_expired
    (
        'a1000000-0000-0000-0007-000000000002',
        NULL,
        'a1000000-0000-0000-0001-000000000002',
        'a1000000-0000-0000-0003-000000000002',
        'a1000000-0000-0000-0004-100200000003',
        'a1000000-0000-0000-0000-000000000001',
        'a1000000-0000-0000-0002-000000000002',
        NOW() - INTERVAL '26 days',
        NOW() - INTERVAL '26 days' + INTERVAL '60 minutes',
        'ttl_expired',
        57101,
        'export',
        3600,
        1048576,
        '{"modbus_registers_polled": 1200}',
        NOW() - INTERVAL '26 days'
    ),
    -- Record 3: operator on N-1001 port 1880, 23 days ago, idle_timeout
    (
        'a1000000-0000-0000-0007-000000000003',
        NULL,
        'a1000000-0000-0000-0001-000000000002',
        'a1000000-0000-0000-0003-000000000001',
        'a1000000-0000-0000-0004-100100000002',
        'a1000000-0000-0000-0000-000000000001',
        'a1000000-0000-0000-0002-000000000001',
        NOW() - INTERVAL '23 days',
        NOW() - INTERVAL '23 days' + INTERVAL '30 minutes',
        'idle_timeout',
        57102,
        'web',
        1800,
        81920,
        '{"browser": "Chrome", "flows_deployed": 0}',
        NOW() - INTERVAL '23 days'
    ),
    -- Record 4: admin on N-1004 port 44818, 20 days ago, user_stopped
    (
        'a1000000-0000-0000-0007-000000000004',
        NULL,
        'a1000000-0000-0000-0001-000000000001',
        'a1000000-0000-0000-0003-000000000004',
        'a1000000-0000-0000-0004-100400000003',
        'a1000000-0000-0000-0000-000000000001',
        'a1000000-0000-0000-0002-000000000001',
        NOW() - INTERVAL '20 days',
        NOW() - INTERVAL '20 days' + INTERVAL '15 minutes',
        'user_stopped',
        57103,
        'export',
        900,
        327680,
        '{"plc_tags_read": 48}',
        NOW() - INTERVAL '20 days'
    ),
    -- Record 5: viewer on N-1001 port 80, 18 days ago, ttl_expired
    (
        'a1000000-0000-0000-0007-000000000005',
        NULL,
        'a1000000-0000-0000-0001-000000000003',
        'a1000000-0000-0000-0003-000000000001',
        'a1000000-0000-0000-0004-100100000001',
        'a1000000-0000-0000-0000-000000000001',
        'a1000000-0000-0000-0002-000000000001',
        NOW() - INTERVAL '18 days',
        NOW() - INTERVAL '18 days' + INTERVAL '60 minutes',
        'ttl_expired',
        57104,
        'web',
        3600,
        512000,
        '{"browser": "Safari"}',
        NOW() - INTERVAL '18 days'
    ),
    -- Record 6: operator on N-1002 port 9090, 14 days ago, idle_timeout
    (
        'a1000000-0000-0000-0007-000000000006',
        NULL,
        'a1000000-0000-0000-0001-000000000002',
        'a1000000-0000-0000-0003-000000000002',
        'a1000000-0000-0000-0004-100200000002',
        'a1000000-0000-0000-0000-000000000001',
        'a1000000-0000-0000-0002-000000000002',
        NOW() - INTERVAL '14 days',
        NOW() - INTERVAL '14 days' + INTERVAL '30 minutes',
        'idle_timeout',
        57105,
        'web',
        1800,
        163840,
        '{"browser": "Edge"}',
        NOW() - INTERVAL '14 days'
    ),
    -- Record 7: admin on N-1004 port 1880, 11 days ago, user_stopped
    (
        'a1000000-0000-0000-0007-000000000007',
        NULL,
        'a1000000-0000-0000-0001-000000000001',
        'a1000000-0000-0000-0003-000000000004',
        'a1000000-0000-0000-0004-100400000001',
        'a1000000-0000-0000-0000-000000000001',
        'a1000000-0000-0000-0002-000000000001',
        NOW() - INTERVAL '11 days',
        NOW() - INTERVAL '11 days' + INTERVAL '47 minutes',
        'user_stopped',
        57106,
        'web',
        2820,
        409600,
        '{"flows_deployed": 3, "browser": "Chrome"}',
        NOW() - INTERVAL '11 days'
    ),
    -- Record 8: operator on N-1001 port 502, 7 days ago, ttl_expired
    (
        'a1000000-0000-0000-0007-000000000008',
        NULL,
        'a1000000-0000-0000-0001-000000000002',
        'a1000000-0000-0000-0003-000000000001',
        'a1000000-0000-0000-0004-100100000003',
        'a1000000-0000-0000-0000-000000000001',
        'a1000000-0000-0000-0002-000000000001',
        NOW() - INTERVAL '7 days',
        NOW() - INTERVAL '7 days' + INTERVAL '120 minutes',
        'ttl_expired',
        57107,
        'export',
        7200,
        2097152,
        '{"modbus_registers_polled": 4800}',
        NOW() - INTERVAL '7 days'
    ),
    -- Record 9: admin on N-1002 port 443, 4 days ago, user_stopped
    (
        'a1000000-0000-0000-0007-000000000009',
        NULL,
        'a1000000-0000-0000-0001-000000000001',
        'a1000000-0000-0000-0003-000000000002',
        'a1000000-0000-0000-0004-100200000001',
        'a1000000-0000-0000-0000-000000000001',
        'a1000000-0000-0000-0002-000000000002',
        NOW() - INTERVAL '4 days',
        NOW() - INTERVAL '4 days' + INTERVAL '8 minutes',
        'user_stopped',
        57108,
        'web',
        480,
        65536,
        '{"browser": "Chrome", "reason_detail": "configuration check complete"}',
        NOW() - INTERVAL '4 days'
    ),
    -- Record 10: linked to active session 2 (operator, N-1001 port 502)
    (
        'a1000000-0000-0000-0007-000000000010',
        'a1000000-0000-0000-0006-000000000002',
        'a1000000-0000-0000-0001-000000000002',
        'a1000000-0000-0000-0003-000000000001',
        'a1000000-0000-0000-0004-100100000003',
        'a1000000-0000-0000-0000-000000000001',
        'a1000000-0000-0000-0002-000000000001',
        NOW() - INTERVAL '45 minutes',
        NULL,
        NULL,
        58002,
        'export',
        NULL,
        0,
        '{"note": "session still active"}',
        NOW() - INTERVAL '45 minutes'
    )
ON CONFLICT (id) DO NOTHING;
