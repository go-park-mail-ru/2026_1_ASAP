ALTER TABLE vk_accounts
ADD CONSTRAINT fk_vk_accounts_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE vk_accounts
ADD CONSTRAINT uq_vk_accounts_vk_user_id UNIQUE (vk_user_id);
