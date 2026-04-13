INSERT INTO users
(login, first_name, last_name, email, password_hash, avatar_url, bio, birth_date, last_seen)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
RETURNING id, created_at, updated_at
