-- FTS on search_vector; chat_id + membership mandatory; ts_rank for ordering
SELECT
  m.id,
  m.chat_id,
  m.sender_id,
  m.content,
  m.created_at,
  ts_rank(m.search_vector, q.tsq) AS rank
FROM messages m
CROSS JOIN LATERAL (SELECT plainto_tsquery('simple', $2) AS tsq) AS q
WHERE m.chat_id = $1
  AND m.deleted_at IS NULL
  AND m.search_vector @@ q.tsq
  AND EXISTS (
    SELECT 1 FROM chat_members cm
    WHERE cm.chat_id = m.chat_id AND cm.user_id = $3
  )
  AND ($4::BIGINT = 0 OR m.id < $4::BIGINT)
ORDER BY rank DESC, m.id DESC
LIMIT $5::INT;
