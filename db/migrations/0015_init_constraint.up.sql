ALTER TABLE oauth_users
ADD CONSTRAINT fk_oauth_users_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE oauth_users
ADD CONSTRAINT uq_oauth_users_provider_id UNIQUE (provider, provider_user_id);

