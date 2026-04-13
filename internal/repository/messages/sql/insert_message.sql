INSERT INTO messages
(chat_id, sender_id, content, sticker_id, edited)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, chat_id, sender_id, content, sticker_id, edited, created_at, updated_at, deleted_at
