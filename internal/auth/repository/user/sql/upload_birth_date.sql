UPDATE users SET birth_date = $2, updated_at = now()
WHERE id = $1
RETURNING id, login, first_name, last_name, avatar_url, bio, birth_date, last_seen
