BEGIN;

-- Guard: skip nếu tenant đã bị archive trước đó (idempotency).
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM tenants WHERE id = '019ef56d-845d-79ad-b551-cf7a785c39b8' AND status = 'active') THEN
    RAISE NOTICE 'tenant already archived, skipping migration';
    RETURN;
  END IF;
END $$;

-- Resolve llm_providers naming conflict — rename personal copy of 'lmstudio'.
UPDATE llm_providers SET name = 'lmstudio-personal'
WHERE tenant_id = '019ef56d-845d-79ad-b551-cf7a785c39b8' AND name = 'lmstudio';

-- Move data từ personal tenant sang Master tenant.
-- UPDATE tenant_id trực tiếp (không DELETE tenant → tất cả FK thỏa mãn).
UPDATE sessions               SET tenant_id = '0193a5b0-7000-7000-8000-000000000001' WHERE tenant_id = '019ef56d-845d-79ad-b551-cf7a785c39b8';
UPDATE channel_instances      SET tenant_id = '0193a5b0-7000-7000-8000-000000000001' WHERE tenant_id = '019ef56d-845d-79ad-b551-cf7a785c39b8';
UPDATE mcp_servers            SET tenant_id = '0193a5b0-7000-7000-8000-000000000001' WHERE tenant_id = '019ef56d-845d-79ad-b551-cf7a785c39b8';
UPDATE llm_providers          SET tenant_id = '0193a5b0-7000-7000-8000-000000000001' WHERE tenant_id = '019ef56d-845d-79ad-b551-cf7a785c39b8';
UPDATE agent_context_files    SET tenant_id = '0193a5b0-7000-7000-8000-000000000001' WHERE tenant_id = '019ef56d-845d-79ad-b551-cf7a785c39b8';
UPDATE user_context_files     SET tenant_id = '0193a5b0-7000-7000-8000-000000000001' WHERE tenant_id = '019ef56d-845d-79ad-b551-cf7a785c39b8';
UPDATE user_agent_profiles    SET tenant_id = '0193a5b0-7000-7000-8000-000000000001' WHERE tenant_id = '019ef56d-845d-79ad-b551-cf7a785c39b8';
UPDATE mcp_agent_grants       SET tenant_id = '0193a5b0-7000-7000-8000-000000000001' WHERE tenant_id = '019ef56d-845d-79ad-b551-cf7a785c39b8';
UPDATE spans                  SET tenant_id = '0193a5b0-7000-7000-8000-000000000001' WHERE tenant_id = '019ef56d-845d-79ad-b551-cf7a785c39b8';
UPDATE traces                 SET tenant_id = '0193a5b0-7000-7000-8000-000000000001' WHERE tenant_id = '019ef56d-845d-79ad-b551-cf7a785c39b8';

-- Move agents (secure_cli_agent_credentials đã verified: 0 rows → không cần xử lý composite FK riêng).
UPDATE agents SET tenant_id = '0193a5b0-7000-7000-8000-000000000001' WHERE tenant_id = '019ef56d-845d-79ad-b551-cf7a785c39b8';

-- Clean up tenant_users: xóa membership cũ trong personal tenant, thêm vào Master.
DELETE FROM tenant_users WHERE tenant_id = '019ef56d-845d-79ad-b551-cf7a785c39b8';
INSERT INTO tenant_users (id, tenant_id, user_id, role, created_at, updated_at)
VALUES (gen_random_uuid(), '0193a5b0-7000-7000-8000-000000000001', '019ef56d-845a-7a38-a235-f223011a6629', 'owner', NOW(), NOW())
ON CONFLICT (tenant_id, user_id) DO NOTHING;

-- Archive personal tenant (không DELETE để tránh FK cascade).
UPDATE tenants SET status = 'archived', updated_at = NOW()
WHERE id = '019ef56d-845d-79ad-b551-cf7a785c39b8';

COMMIT;
