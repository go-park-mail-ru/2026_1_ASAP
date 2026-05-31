SELECT
  c.contact_user_id,
  c.first_name,
  c.last_name,
  u.login,
  p.avatar_url,
  p.last_seen
FROM contacts c
INNER JOIN users u ON u.id = c.contact_user_id
INNER JOIN profiles p ON p.user_id = c.contact_user_id
WHERE c.user_id = $1
  AND (
    c.first_name ILIKE $2 ESCAPE '\'
    OR COALESCE(c.last_name, '') ILIKE $2 ESCAPE '\'
    OR u.login ILIKE $2 ESCAPE '\'
  )
  AND ($3::BIGINT = 0 OR c.contact_user_id < $3::BIGINT)
ORDER BY c.contact_user_id DESC
LIMIT $4::INT;
