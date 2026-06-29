BEGIN;

-- Unarchive personal tenant.
UPDATE tenants SET status = 'active', updated_at = NOW()
WHERE id = '019ef56d-845d-79ad-b551-cf7a785c39b8';

-- Remove user from Master tenant_users (added by migration).
DELETE FROM tenant_users
WHERE tenant_id = '0193a5b0-7000-7000-8000-000000000001'
  AND user_id = '019ef56d-845a-7a38-a235-f223011a6629';

-- Re-add user to personal tenant.
INSERT INTO tenant_users (id, tenant_id, user_id, role, created_at, updated_at)
VALUES (gen_random_uuid(), '019ef56d-845d-79ad-b551-cf7a785c39b8', '019ef56d-845a-7a38-a235-f223011a6629', 'owner', NOW(), NOW())
ON CONFLICT (tenant_id, user_id) DO NOTHING;

-- NOTE: Data rows (agents, sessions, channels, etc.) are NOT moved back.
-- This rollback only restores the tenant + user membership.
-- Full data rollback requires a DB snapshot.

COMMIT;
