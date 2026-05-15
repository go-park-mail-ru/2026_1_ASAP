SELECT m.id, m.chat_id, m.sender_id, m.content, m.sticker_id, m.edited, m.created_at, m.updated_at, m.deleted_at
FROM messages m
JOIN chats c ON m.id = c.last_message_id
WHERE c.id = $1 AND m.deleted_at IS NULL

