UPDATE messages AS m
SET content = $1, edited = TRUE, updated_at = NOW()
FROM chats AS c
WHERE m.id = $2
  AND m.chat_id = $3
  AND m.sender_id = $4
  AND m.deleted_at IS NULL
  AND c.id = m.chat_id
RETURNING m.id,
  m.chat_id,
  m.sender_id,
  m.content,
  m.created_at,
  m.updated_at,
  m.edited,
  (c.last_message_id IS NOT NULL AND c.last_message_id = m.id);
