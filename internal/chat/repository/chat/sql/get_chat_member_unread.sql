SELECT COALESCE(cm.last_read_message_id, 0) AS last_read_message_id,
       (SELECT COUNT(*)::bigint
        FROM messages m
        WHERE m.chat_id = cm.chat_id
          AND m.id > COALESCE(cm.last_read_message_id, 0)
          AND m.sender_id <> $2
          AND m.deleted_at IS NULL) AS unread_count
FROM chat_members cm
WHERE cm.chat_id = $1
  AND cm.user_id = $2
