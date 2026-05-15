UPDATE profiles SET avatar_url = $2, updated_at = now()
WHERE user_id = $1
RETURNING user_id, first_name, last_name, avatar_url, bio, birth_date, last_seen
