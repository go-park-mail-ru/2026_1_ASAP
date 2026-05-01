SELECT
  c.id,
  c.type,
  CASE
    WHEN c.type = 'dialog' THEN
      CASE
        WHEN COALESCE(d.last_name, '') <> '' THEN COALESCE(d.first_name, '') || COALESCE(d.last_name, '')
        ELSE COALESCE(NULLIF(d.first_name, ''), d.login, '')
      END
    ELSE COALESCE(c.title, '')
  END AS title,
  c.avatar_url,
  lm.content AS last_message_preview,
  lm.created_at AS last_message_at
FROM chats c
INNER JOIN chat_members cm ON c.id = cm.chat_id AND cm.user_id = $1
LEFT JOIN messages lm ON lm.id = c.last_message_id AND lm.deleted_at IS NULL
LEFT JOIN LATERAL (
  SELECT
    u.login AS login,
    p.first_name AS first_name,
    COALESCE(p.last_name, '') AS last_name
  FROM chat_members cm2
  JOIN users u ON u.id = cm2.user_id
  LEFT JOIN profiles p ON p.user_id = cm2.user_id
  WHERE cm2.chat_id = c.id
    AND cm2.user_id <> $1
  ORDER BY cm2.user_id
  LIMIT 1
) AS d ON c.type = 'dialog'
WHERE (
    COALESCE(c.title, '') ILIKE $2 ESCAPE '\'
    OR (
      c.type = 'dialog' AND (
        COALESCE(d.login, '') ILIKE $2 ESCAPE '\'
        OR COALESCE(d.first_name, '') ILIKE $2 ESCAPE '\'
        OR COALESCE(d.last_name, '') ILIKE $2 ESCAPE '\'
        OR CONCAT_WS(' ', COALESCE(d.first_name, ''), COALESCE(d.last_name, '')) ILIKE $2 ESCAPE '\'
      )
    )
  )
  AND (COALESCE(CARDINALITY($3::TEXT[]), 0) = 0 OR c.type = ANY($3::TEXT[]))
  AND ($4::BIGINT = 0 OR c.id < $4::BIGINT)
ORDER BY c.id DESC
LIMIT $5::INT;
