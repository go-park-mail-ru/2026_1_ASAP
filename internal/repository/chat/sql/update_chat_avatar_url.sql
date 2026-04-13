UPDATE chats SET avatar_url = $2, updated_at = now()
WHERE id = $1
RETURNING id, type, title, description, owner_id, avatar_url, created_at, updated_at
