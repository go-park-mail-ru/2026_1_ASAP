SELECT c.id, c.type, c.title, c.description, c.owner_id, c.avatar_url, c.created_at, c.updated_at
FROM chats c
JOIN chat_members cm1 ON c.id = cm1.chat_id
JOIN chat_members cm2 ON c.id = cm2.chat_id
WHERE c.type = 'dialog'
  AND cm1.user_id IN ($1, $2)
  AND cm2.user_id IN ($1, $2)
  AND cm1.user_id != cm2.user_id
LIMIT 1
