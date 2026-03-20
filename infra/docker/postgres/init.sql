-- PostgreSQL initialization script
-- Runs automatically when the postgres container starts fresh

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE nucleus_portal TO nucleus;
