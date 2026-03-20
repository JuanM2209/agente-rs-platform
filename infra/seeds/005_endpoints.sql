-- Seed: 005_endpoints
-- Description: Seed data for the endpoints and bridge_profiles tables
-- Idempotent: ON CONFLICT (device_id, port) DO NOTHING for endpoints;
--             ON CONFLICT (id) DO NOTHING for bridge_profiles

-- ---------------------------------------------------------------------------
-- N-1001: Field Unit Alpha-1
-- ---------------------------------------------------------------------------
INSERT INTO endpoints (id, device_id, type, port, label, protocol, description, enabled, created_at)
VALUES
    (
        'a1000000-0000-0000-0004-100100000001',
        'a1000000-0000-0000-0003-000000000001',
        'WEB',
        80,
        'HTTP',
        'tcp',
        'Standard HTTP web interface',
        true,
        NOW() - INTERVAL '170 days'
    ),
    (
        'a1000000-0000-0000-0004-100100000002',
        'a1000000-0000-0000-0003-000000000001',
        'WEB',
        1880,
        'Node-RED',
        'tcp',
        'Node-RED flow editor and dashboard',
        true,
        NOW() - INTERVAL '170 days'
    ),
    (
        'a1000000-0000-0000-0004-100100000003',
        'a1000000-0000-0000-0003-000000000001',
        'PROGRAM',
        502,
        'Modbus TCP',
        'tcp',
        'Modbus TCP register access',
        true,
        NOW() - INTERVAL '170 days'
    )
ON CONFLICT (device_id, port) DO NOTHING;

-- ---------------------------------------------------------------------------
-- N-1002: Tank Farm Monitor
-- ---------------------------------------------------------------------------
INSERT INTO endpoints (id, device_id, type, port, label, protocol, description, enabled, created_at)
VALUES
    (
        'a1000000-0000-0000-0004-100200000001',
        'a1000000-0000-0000-0003-000000000002',
        'WEB',
        443,
        'HTTPS',
        'tcp',
        'Secure HTTPS web interface',
        true,
        NOW() - INTERVAL '145 days'
    ),
    (
        'a1000000-0000-0000-0004-100200000002',
        'a1000000-0000-0000-0003-000000000002',
        'WEB',
        9090,
        'Device UI',
        'tcp',
        'Embedded device management UI',
        true,
        NOW() - INTERVAL '145 days'
    ),
    (
        'a1000000-0000-0000-0004-100200000003',
        'a1000000-0000-0000-0003-000000000002',
        'PROGRAM',
        502,
        'Modbus TCP',
        'tcp',
        'Modbus TCP register access',
        true,
        NOW() - INTERVAL '145 days'
    ),
    -- BRIDGE endpoint for N-1002 (serial /dev/ttyUSB0)
    -- Port 8502 reserved as the TCP-side of the bridge
    (
        'a1000000-0000-0000-0004-100200000004',
        'a1000000-0000-0000-0003-000000000002',
        'BRIDGE',
        8502,
        'Serial Bridge (/dev/ttyUSB0)',
        'tcp',
        'Serial-to-TCP bridge for /dev/ttyUSB0',
        true,
        NOW() - INTERVAL '145 days'
    )
ON CONFLICT (device_id, port) DO NOTHING;

-- Bridge profile for N-1002 /dev/ttyUSB0
INSERT INTO bridge_profiles (id, device_id, serial_port, baud_rate, parity, stop_bits, data_bits, tcp_port, status, created_at)
VALUES
    (
        'a1000000-0000-0000-0005-100200000001',
        'a1000000-0000-0000-0003-000000000002',
        '/dev/ttyUSB0',
        9600,
        'N',
        1,
        8,
        8502,
        'idle',
        NOW() - INTERVAL '145 days'
    )
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- N-1003: Compressor Control Node
-- ---------------------------------------------------------------------------
INSERT INTO endpoints (id, device_id, type, port, label, protocol, description, enabled, created_at)
VALUES
    (
        'a1000000-0000-0000-0004-100300000001',
        'a1000000-0000-0000-0003-000000000003',
        'WEB',
        80,
        'HTTP',
        'tcp',
        'Standard HTTP web interface',
        true,
        NOW() - INTERVAL '130 days'
    ),
    (
        'a1000000-0000-0000-0004-100300000002',
        'a1000000-0000-0000-0003-000000000003',
        'PROGRAM',
        22,
        'SSH',
        'tcp',
        'SSH remote shell access',
        true,
        NOW() - INTERVAL '130 days'
    ),
    (
        'a1000000-0000-0000-0004-100300000003',
        'a1000000-0000-0000-0003-000000000003',
        'PROGRAM',
        502,
        'Modbus TCP',
        'tcp',
        'Modbus TCP register access',
        true,
        NOW() - INTERVAL '130 days'
    )
ON CONFLICT (device_id, port) DO NOTHING;

-- ---------------------------------------------------------------------------
-- N-1004: Separator Unit Controller
-- ---------------------------------------------------------------------------
INSERT INTO endpoints (id, device_id, type, port, label, protocol, description, enabled, created_at)
VALUES
    (
        'a1000000-0000-0000-0004-100400000001',
        'a1000000-0000-0000-0003-000000000004',
        'WEB',
        1880,
        'Node-RED',
        'tcp',
        'Node-RED flow editor and dashboard',
        true,
        NOW() - INTERVAL '100 days'
    ),
    (
        'a1000000-0000-0000-0004-100400000002',
        'a1000000-0000-0000-0003-000000000004',
        'WEB',
        9090,
        'Device UI',
        'tcp',
        'Embedded device management UI',
        true,
        NOW() - INTERVAL '100 days'
    ),
    (
        'a1000000-0000-0000-0004-100400000003',
        'a1000000-0000-0000-0003-000000000004',
        'PROGRAM',
        44818,
        'EtherNet/IP',
        'tcp',
        'Allen-Bradley EtherNet/IP CIP communication',
        true,
        NOW() - INTERVAL '100 days'
    ),
    -- BRIDGE endpoint for N-1004 (serial /dev/ttyS0)
    -- Port 8500 reserved as the TCP-side of the bridge
    (
        'a1000000-0000-0000-0004-100400000004',
        'a1000000-0000-0000-0003-000000000004',
        'BRIDGE',
        8500,
        'Serial Bridge (/dev/ttyS0)',
        'tcp',
        'Serial-to-TCP bridge for /dev/ttyS0 (RS-232)',
        true,
        NOW() - INTERVAL '100 days'
    )
ON CONFLICT (device_id, port) DO NOTHING;

-- Bridge profile for N-1004 /dev/ttyS0
INSERT INTO bridge_profiles (id, device_id, serial_port, baud_rate, parity, stop_bits, data_bits, tcp_port, status, created_at)
VALUES
    (
        'a1000000-0000-0000-0005-100400000001',
        'a1000000-0000-0000-0003-000000000004',
        '/dev/ttyS0',
        19200,
        'E',
        1,
        8,
        8500,
        'idle',
        NOW() - INTERVAL '100 days'
    )
ON CONFLICT (id) DO NOTHING;
