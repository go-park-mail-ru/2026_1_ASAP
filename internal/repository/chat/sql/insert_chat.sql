INSERT INTO chats
(type, title, description, owner_id, avatar_url)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at, updated_at
