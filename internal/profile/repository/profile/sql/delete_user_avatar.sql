UPDATE profiles
SET avatar_url=NULL, updated_at = now()
WHERE id=$1
RETURNING user_id, first_name, last_name, avatar_url, bio, birth_date, last_seen
