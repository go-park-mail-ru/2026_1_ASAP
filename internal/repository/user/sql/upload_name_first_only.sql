UPDATE users SET first_name = $2, updated_at = now()
WHERE id = $1
RETURNING id, login, first_name, last_name, avatar_url, bio, birth_date, last_seen
