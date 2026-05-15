INSERT INTO complaint
(status, type, feedback_name, feedback_email, body, user_id, attachment_url)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, status, type, feedback_name, feedback_email, body, user_id, attachment_url, created_at, updated_at
