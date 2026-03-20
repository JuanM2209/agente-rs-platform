-- Seed: 001_tenants
-- Description: Seed data for the tenants table
-- Idempotent: ON CONFLICT DO NOTHING on slug (unique constraint)

INSERT INTO tenants (id, name, slug, created_at, updated_at)
VALUES
    (
        'a1000000-0000-0000-0000-000000000001',
        'Alpha Industries',
        'tenant_alpha',
        NOW() - INTERVAL '180 days',
        NOW() - INTERVAL '1 day'
    ),
    (
        'b2000000-0000-0000-0000-000000000001',
        'Beta Controls',
        'tenant_beta',
        NOW() - INTERVAL '90 days',
        NOW() - INTERVAL '3 days'
    )
ON CONFLICT (slug) DO NOTHING;
