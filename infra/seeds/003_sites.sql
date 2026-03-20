-- Seed: 003_sites
-- Description: Seed data for the sites table
-- Idempotent: INSERT ... ON CONFLICT DO NOTHING (guarded by fixed UUIDs;
-- sites has no other unique constraint, so we rely on PK collision)

INSERT INTO sites (id, tenant_id, name, location, timezone, created_at)
VALUES
    (
        'a1000000-0000-0000-0002-000000000001',
        'a1000000-0000-0000-0000-000000000001',
        'Alpha - Main Plant',
        'Houston, TX',
        'America/Chicago',
        NOW() - INTERVAL '178 days'
    ),
    (
        'a1000000-0000-0000-0002-000000000002',
        'a1000000-0000-0000-0000-000000000001',
        'Alpha - Remote Field Site A',
        'Permian Basin',
        'America/Chicago',
        NOW() - INTERVAL '150 days'
    ),
    (
        'b2000000-0000-0000-0002-000000000001',
        'b2000000-0000-0000-0000-000000000001',
        'Beta - Processing Unit 1',
        'Midland, TX',
        'America/Chicago',
        NOW() - INTERVAL '87 days'
    )
ON CONFLICT (id) DO NOTHING;
