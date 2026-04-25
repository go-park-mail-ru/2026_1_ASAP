ALTER TABLE vk_accounts DROP CONSTRAINT IF EXISTS fk_vk_accounts_user_id;
ALTER TABLE vk_accounts DROP CONSTRAINT IF EXISTS uq_vk_accounts_vk_user_id;
