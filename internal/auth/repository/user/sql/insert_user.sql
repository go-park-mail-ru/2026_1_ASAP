INSERT INTO users (login, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id, created_at, updated_at