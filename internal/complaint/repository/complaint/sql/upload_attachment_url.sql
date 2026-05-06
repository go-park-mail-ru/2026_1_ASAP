UPDATE complaint
SET attachment_url = $2, updated_at = now()
WHERE id = $1
RETURNING id, status, type, feedback_name, feedback_email, body, user_id, attachment_url, created_at, updated_at
