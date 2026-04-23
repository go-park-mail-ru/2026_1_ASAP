SELECT id, first_name, last_name, avatar_url, bio, birth_date, last_seen
FROM users WHERE id=$1
