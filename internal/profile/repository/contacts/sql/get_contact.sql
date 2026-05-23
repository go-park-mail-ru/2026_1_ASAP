SELECT c.user_id, c.first_name, c.last_name, c.contact_user_id, p.avatar_url, c.created_at, c.updated_at
FROM contacts c
JOIN profiles p ON c.contact_user_id = p.user_id
WHERE c.user_id = $1 AND c.contact_user_id = $2
