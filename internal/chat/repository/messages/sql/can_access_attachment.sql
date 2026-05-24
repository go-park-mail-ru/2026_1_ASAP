SELECT EXISTS (
    SELECT 1
    FROM message_attachments ma
    JOIN messages m ON m.id = ma.message_id AND m.deleted_at IS NULL
    JOIN chat_members cm ON cm.chat_id = m.chat_id AND cm.user_id = $1
    WHERE ma.file_url LIKE '%' || $2 || '%'
       OR $3 LIKE '%' || ma.file_url || '%'
)
