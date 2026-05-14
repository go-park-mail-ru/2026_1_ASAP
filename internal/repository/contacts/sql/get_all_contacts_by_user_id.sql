SELECT c.user_id, c.first_name, c.last_name, c.contact_user_id, u.avatar_url, c.created_at, c.updated_at
FROM contacts c
JOIN users u ON c.contact_user_id = u.id
WHERE c.user_id = $1
ORDER BY first_name ASC
