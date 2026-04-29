SELECT id, chat_id, sender_id, content, sticker_id, edited, created_at, updated_at, deleted_at
FROM messages
WHERE chat_id = $1 AND id < $2 AND deleted_at IS NULL
ORDER BY id DESC
LIMIT $3
