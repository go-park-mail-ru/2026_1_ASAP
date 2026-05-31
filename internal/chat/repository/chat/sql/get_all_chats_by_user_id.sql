SELECT c.id, c.type, c.title, c.description, c.owner_id, c.avatar_url, c.created_at, c.updated_at,
       COALESCE(cm.last_read_message_id, 0) AS last_read_message_id,
       (SELECT COUNT(*)::bigint
        FROM messages m
        WHERE m.chat_id = c.id
          AND m.id > COALESCE(cm.last_read_message_id, 0)
          AND m.sender_id <> $1
          AND m.deleted_at IS NULL) AS unread_count
FROM chats c
INNER JOIN chat_members cm ON c.id = cm.chat_id
WHERE cm.user_id = $1
