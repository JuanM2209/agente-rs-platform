-- Seed: 002_users
-- Description: Seed data for the users table
-- Idempotent: ON CONFLICT DO NOTHING on email (unique constraint)
--
-- Password for all seed users: DevPass123!
-- bcrypt hash (cost 10): $2a$10$0C2h/7oWHWDUi5F56HdoXeyWjchc0KIrvdpXfXv1Qndc7fnY3aYbG

INSERT INTO users (id, tenant_id, email, password_hash, display_name, role, is_active, created_at, updated_at)
VALUES
    -- Alpha Industries users
    (
        'a1000000-0000-0000-0001-000000000001',
        'a1000000-0000-0000-0000-000000000001',
        'admin@alpha.com',
        '$2a$10$0C2h/7oWHWDUi5F56HdoXeyWjchc0KIrvdpXfXv1Qndc7fnY3aYbG',
        'Alpha Admin',
        'admin',
        true,
        NOW() - INTERVAL '179 days',
        NOW() - INTERVAL '1 day'
    ),
    (
        'a1000000-0000-0000-0001-000000000002',
        'a1000000-0000-0000-0000-000000000001',
        'operator@alpha.com',
        '$2a$10$0C2h/7oWHWDUi5F56HdoXeyWjchc0KIrvdpXfXv1Qndc7fnY3aYbG',
        'Alpha Operator',
        'operator',
        true,
        NOW() - INTERVAL '120 days',
        NOW() - INTERVAL '2 days'
    ),
    (
        'a1000000-0000-0000-0001-000000000003',
        'a1000000-0000-0000-0000-000000000001',
        'viewer@alpha.com',
        '$2a$10$0C2h/7oWHWDUi5F56HdoXeyWjchc0KIrvdpXfXv1Qndc7fnY3aYbG',
        'Alpha Viewer',
        'viewer',
        true,
        NOW() - INTERVAL '60 days',
        NOW() - INTERVAL '5 days'
    ),
    -- Beta Controls users
    (
        'b2000000-0000-0000-0001-000000000001',
        'b2000000-0000-0000-0000-000000000001',
        'support@beta.com',
        '$2a$10$0C2h/7oWHWDUi5F56HdoXeyWjchc0KIrvdpXfXv1Qndc7fnY3aYbG',
        'Beta Support Engineer',
        'support',
        true,
        NOW() - INTERVAL '88 days',
        NOW() - INTERVAL '3 days'
    ),
    (
        'b2000000-0000-0000-0001-000000000002',
        'b2000000-0000-0000-0000-000000000001',
        'admin@beta.com',
        '$2a$10$0C2h/7oWHWDUi5F56HdoXeyWjchc0KIrvdpXfXv1Qndc7fnY3aYbG',
        'Beta Admin',
        'admin',
        true,
        NOW() - INTERVAL '89 days',
        NOW() - INTERVAL '1 day'
    )
ON CONFLICT (email) DO NOTHING;
