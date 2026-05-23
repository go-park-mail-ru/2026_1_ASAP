INSERT INTO message_attachments (
    message_id, type, sort_order, file_url, file_name, mime_type, file_size,
    contact_user_id, contact_first_name, contact_last_name, contact_avatar_url
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, message_id, type, sort_order, file_url, file_name, mime_type, file_size,
    contact_user_id, contact_first_name, contact_last_name, contact_avatar_url, created_at
