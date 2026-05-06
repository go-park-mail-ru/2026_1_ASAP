SELECT id, status, type, feedback_name, feedback_email, body, user_id, attachment_url, created_at, updated_at
FROM complaint
WHERE user_id = $1
ORDER BY created_at DESC
