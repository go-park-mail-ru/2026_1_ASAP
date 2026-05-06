SELECT
  p.user_id,
  p.first_name,
  COALESCE(p.last_name, '') AS last_name,
  u.login,
  p.avatar_url,
  p.last_seen
FROM profiles p
INNER JOIN users u ON u.id = p.user_id
WHERE p.user_id <> $1
  AND (
    u.login ILIKE $2 ESCAPE '\'
    OR p.first_name ILIKE $2 ESCAPE '\'
    OR COALESCE(p.last_name, '') ILIKE $2 ESCAPE '\'
  )
  AND ($3::BIGINT = 0 OR p.user_id < $3::BIGINT)
ORDER BY p.user_id DESC
LIMIT $4::INT;
