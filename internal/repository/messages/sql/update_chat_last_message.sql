UPDATE chats
SET last_message_id = $1, updated_at = now()
WHERE id = $2
