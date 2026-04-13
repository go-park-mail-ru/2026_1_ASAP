SELECT id, login, first_name, last_name, email, password_hash, avatar_url, bio, birth_date, last_seen, created_at, updated_at
FROM users WHERE id=$1
