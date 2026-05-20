UPDATE chat_members
SET
    last_read_message_id = GREATEST(COALESCE(last_read_message_id, 0), $3),
    updated_at = now()
WHERE chat_id = $1
  AND user_id = $2
  AND EXISTS (
      SELECT 1
      FROM messages m
      WHERE m.id = $3
        AND m.chat_id = $1
        AND m.deleted_at IS NULL
  )
RETURNING last_read_message_id;
