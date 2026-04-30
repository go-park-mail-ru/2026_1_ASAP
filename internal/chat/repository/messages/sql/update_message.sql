UPDATE messages
SET content=$1, edited = TRUE, updated_at = NOW()
WHERE id=$2 AND chat_id=$3 AND sender_id=$4 AND deleted_at IS NULL
RETURNING id, chat_id, sender_id, content, created_at, updated_at, edited;
