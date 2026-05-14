SELECT id, type, title, description, owner_id, avatar_url, created_at, updated_at
FROM chats
WHERE id = $1
