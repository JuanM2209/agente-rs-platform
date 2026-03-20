-- Seed: 004_devices
-- Description: Seed data for the devices table (tenant_alpha test devices)
-- Idempotent: ON CONFLICT (device_id) DO NOTHING

INSERT INTO devices (
    id,
    tenant_id,
    site_id,
    device_id,
    display_name,
    status,
    last_seen,
    firmware_version,
    ip_address,
    hardware_model,
    tags,
    created_at,
    updated_at
)
VALUES
    -- N-1001: Field Unit Alpha-1 — Main Plant, online
    (
        'a1000000-0000-0000-0003-000000000001',
        'a1000000-0000-0000-0000-000000000001',
        'a1000000-0000-0000-0002-000000000001',
        'N-1001',
        'Field Unit Alpha-1',
        'online',
        NOW() - INTERVAL '2 minutes',
        'v1.2.1',
        '192.168.1.101',
        'Nucleus Edge 200',
        '{"environment": "production", "zone": "plant-floor"}',
        NOW() - INTERVAL '170 days',
        NOW() - INTERVAL '2 minutes'
    ),
    -- N-1002: Tank Farm Monitor — Remote Field Site A, online
    (
        'a1000000-0000-0000-0003-000000000002',
        'a1000000-0000-0000-0000-000000000001',
        'a1000000-0000-0000-0002-000000000002',
        'N-1002',
        'Tank Farm Monitor',
        'online',
        NOW() - INTERVAL '1 minute',
        'v1.1.8',
        '10.0.2.15',
        'Nucleus Edge 100',
        '{"environment": "production", "zone": "tank-farm"}',
        NOW() - INTERVAL '145 days',
        NOW() - INTERVAL '1 minute'
    ),
    -- N-1003: Compressor Control Node — Remote Field Site A, offline
    (
        'a1000000-0000-0000-0003-000000000003',
        'a1000000-0000-0000-0000-000000000001',
        'a1000000-0000-0000-0002-000000000002',
        'N-1003',
        'Compressor Control Node',
        'offline',
        NOW() - INTERVAL '3 hours',
        'v1.2.0',
        '10.0.2.20',
        'Nucleus Edge 200',
        '{"environment": "production", "zone": "compression"}',
        NOW() - INTERVAL '130 days',
        NOW() - INTERVAL '3 hours'
    ),
    -- N-1004: Separator Unit Controller — Main Plant, online
    (
        'a1000000-0000-0000-0003-000000000004',
        'a1000000-0000-0000-0000-000000000001',
        'a1000000-0000-0000-0002-000000000001',
        'N-1004',
        'Separator Unit Controller',
        'online',
        NOW() - INTERVAL '30 seconds',
        'v1.2.1',
        '192.168.1.104',
        'Nucleus Edge 200',
        '{"environment": "production", "zone": "separation"}',
        NOW() - INTERVAL '100 days',
        NOW() - INTERVAL '30 seconds'
    )
ON CONFLICT (device_id) DO NOTHING;
