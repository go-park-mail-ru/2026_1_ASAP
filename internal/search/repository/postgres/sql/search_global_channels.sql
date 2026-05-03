SELECT
  c.id,
  COALESCE(c.title, '') AS title,
  c.avatar_url,
  lm.content AS last_message_preview,
  lm.created_at AS last_message_at,
  EXISTS (
    SELECT 1
    FROM chat_members cm
    WHERE cm.chat_id = c.id
      AND cm.user_id = $1
  ) AS is_member
FROM chats c
LEFT JOIN messages lm ON lm.id = c.last_message_id AND lm.deleted_at IS NULL
WHERE c.type = 'channel'
  AND (
    COALESCE(c.title, '') ILIKE $2 ESCAPE '\'
    OR COALESCE(c.description, '') ILIKE $2 ESCAPE '\'
  )
  AND ($3::BIGINT = 0 OR c.id < $3::BIGINT)
ORDER BY c.id DESC
LIMIT $4::INT;
