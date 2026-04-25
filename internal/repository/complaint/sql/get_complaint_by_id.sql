SELECT id, status, type, feedback_name, feedback_email, body, user_id, attachment_url, created_at, updated_at
FROM complaint
WHERE id = $1
