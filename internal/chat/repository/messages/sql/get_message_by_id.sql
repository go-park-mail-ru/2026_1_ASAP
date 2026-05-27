SELECT id, chat_id, sender_id, content, sticker_id, edited, created_at, updated_at, deleted_at
FROM messages
WHERE id = $1
  AND chat_id = $2
  AND deleted_at IS NULL
