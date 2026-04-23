SELECT u.id, u.login, u.first_name, u.last_name, u.email, u.password_hash, u.avatar_url, u.bio, u.birth_date, u.last_seen, u.created_at, u.updated_at
FROM users u
JOIN vk_accounts vk ON vk.user_id = u.id
WHERE vk.vk_user_id = $1;
