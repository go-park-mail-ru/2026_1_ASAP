SELECT user_id
FROM chat_members
WHERE chat_id = $1
ORDER BY user_id ASC
