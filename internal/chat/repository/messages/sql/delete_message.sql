WITH deleted AS (
    UPDATE messages
    SET
        deleted_at = NOW(),
        updated_at = NOW(),
        edited = FALSE,
        content = NULL,
        sticker_id = NULL
    WHERE id = $1
      AND chat_id = $2
      AND sender_id = $3
      AND deleted_at IS NULL
    RETURNING id, chat_id, sender_id, deleted_at, updated_at
),
new_last AS (
    SELECT m.id
    FROM messages m
    JOIN deleted d ON d.chat_id = m.chat_id
    WHERE m.id <> d.id
      AND m.deleted_at IS NULL
    ORDER BY m.created_at DESC, m.id DESC
    LIMIT 1
),
updated_chat AS (
    UPDATE chats c
    SET
        last_message_id = (SELECT nl.id FROM new_last nl),
        updated_at = NOW()
    FROM deleted d
    WHERE c.id = d.chat_id
      AND c.last_message_id = d.id
    RETURNING c.id
)
SELECT d.id,
       d.chat_id,
       d.sender_id,
       d.deleted_at,
       d.updated_at,
       EXISTS (SELECT 1 FROM updated_chat) AS last_message_edited
FROM deleted d;
