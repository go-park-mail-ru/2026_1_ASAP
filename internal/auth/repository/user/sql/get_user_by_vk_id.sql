SELECT u.id, u.login, u.email, u.password_hash, u.created_at, u.updated_at
FROM users u
JOIN vk_accounts vk ON vk.user_id = u.id
WHERE vk.vk_user_id = $1
