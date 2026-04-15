SELECT m.id, m.chat_id, m.sender_id, m.content, m.sticker_id, m.edited, m.created_at, m.updated_at, m.deleted_at
FROM messages m
JOIN chats c ON m.id = c.last_message_id
JOIN chat_members cm ON c.id = cm.chat_id
WHERE cm.user_id = $1
