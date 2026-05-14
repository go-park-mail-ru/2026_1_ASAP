SELECT id, login, first_name, last_name, email, avatar_url, bio, birth_date, last_seen
FROM users WHERE id=$1
