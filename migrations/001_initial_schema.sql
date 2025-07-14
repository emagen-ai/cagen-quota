-- Initial schema for Cagen Quota Service
-- This file contains the complete database schema

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Quotas table
CREATE TABLE IF NOT EXISTS quotas (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(20) NOT NULL CHECK (type IN ('organization', 'team')),
    
    -- Capacity in MB
    total_mb BIGINT NOT NULL DEFAULT 0 CHECK (total_mb >= 0),
    used_mb BIGINT NOT NULL DEFAULT 0 CHECK (used_mb >= 0),
    allocated_mb BIGINT NOT NULL DEFAULT 0 CHECK (allocated_mb >= 0),
    available_mb BIGINT GENERATED ALWAYS AS (total_mb - used_mb - allocated_mb) STORED,
    
    -- Hierarchy
    parent_quota_id VARCHAR(50) REFERENCES quotas(id),
    level INTEGER NOT NULL DEFAULT 0 CHECK (level >= 0),
    path TEXT NOT NULL,
    
    -- Ownership
    owner_id VARCHAR(255) NOT NULL,
    organization_id VARCHAR(255) NOT NULL,
    team_id VARCHAR(255),
    
    -- Status and timestamps
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'deleted')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT quota_balance_check CHECK (used_mb + allocated_mb <= total_mb)
);

-- Quota usage table
CREATE TABLE IF NOT EXISTS quota_usage (
    id VARCHAR(50) PRIMARY KEY,
    quota_id VARCHAR(50) NOT NULL REFERENCES quotas(id),
    user_id VARCHAR(255) NOT NULL,
    resource_id VARCHAR(255),
    usage_mb BIGINT NOT NULL CHECK (usage_mb > 0),
    operation VARCHAR(20) NOT NULL CHECK (operation IN ('allocate', 'deallocate')),
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Quota audit logs table
CREATE TABLE IF NOT EXISTS quota_audit_logs (
    id VARCHAR(50) PRIMARY KEY,
    quota_id VARCHAR(50) NOT NULL REFERENCES quotas(id),
    action_type VARCHAR(50) NOT NULL,
    actor_user_id VARCHAR(255) NOT NULL,
    target_user_id VARCHAR(255),
    details JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_quotas_parent ON quotas(parent_quota_id);
CREATE INDEX IF NOT EXISTS idx_quotas_organization ON quotas(organization_id);
CREATE INDEX IF NOT EXISTS idx_quotas_team ON quotas(team_id);
CREATE INDEX IF NOT EXISTS idx_quotas_owner ON quotas(owner_id);
CREATE INDEX IF NOT EXISTS idx_quotas_type_status ON quotas(type, status);
CREATE INDEX IF NOT EXISTS idx_quotas_path ON quotas USING GIN (to_tsvector('simple', path));

CREATE INDEX IF NOT EXISTS idx_quota_usage_quota ON quota_usage(quota_id);
CREATE INDEX IF NOT EXISTS idx_quota_usage_user ON quota_usage(user_id);
CREATE INDEX IF NOT EXISTS idx_quota_usage_resource ON quota_usage(resource_id);
CREATE INDEX IF NOT EXISTS idx_quota_usage_created ON quota_usage(created_at);

CREATE INDEX IF NOT EXISTS idx_quota_audit_quota ON quota_audit_logs(quota_id);
CREATE INDEX IF NOT EXISTS idx_quota_audit_actor ON quota_audit_logs(actor_user_id);
CREATE INDEX IF NOT EXISTS idx_quota_audit_created ON quota_audit_logs(created_at);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to automatically update updated_at
DROP TRIGGER IF EXISTS update_quotas_updated_at ON quotas;
CREATE TRIGGER update_quotas_updated_at
    BEFORE UPDATE ON quotas
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();