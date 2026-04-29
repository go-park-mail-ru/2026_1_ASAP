SELECT c.id, c.type, c.title, c.description, c.owner_id, c.avatar_url, c.created_at, c.updated_at
FROM chats c
INNER JOIN chat_members cm ON c.id = cm.chat_id
WHERE cm.user_id = $1
