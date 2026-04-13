INSERT INTO contacts
(user_id, contact_user_id, first_name, last_name)
VALUES ($1, $2, $3, $4)
RETURNING user_id, contact_user_id, first_name, last_name, created_at, updated_at
