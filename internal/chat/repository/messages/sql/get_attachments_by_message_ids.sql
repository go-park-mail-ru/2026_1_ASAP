SELECT id, message_id, type, sort_order, file_url, file_name, mime_type, file_size,
    contact_user_id, contact_first_name, contact_last_name, contact_avatar_url,
    duration_ms, waveform, transcript, created_at
FROM message_attachments
WHERE message_id = ANY($1)
ORDER BY message_id, sort_order
