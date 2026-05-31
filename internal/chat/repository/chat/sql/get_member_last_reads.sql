SELECT user_id, last_read_message_id
FROM chat_members
WHERE chat_id = $1
ORDER BY user_id ASC
