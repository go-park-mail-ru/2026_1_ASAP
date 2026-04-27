SELECT id, login, email, password_hash, created_at, updated_at
FROM users WHERE id=$1
