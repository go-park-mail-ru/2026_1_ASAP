UPDATE message_attachments
SET transcript = $2
WHERE id = $1
  AND type = 'voice'
RETURNING id, message_id, type, sort_order, file_url, file_name, mime_type, file_size,
    contact_user_id, contact_first_name, contact_last_name, contact_avatar_url,
    duration_ms, waveform, transcript, created_at
