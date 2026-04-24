DELETE FROM contacts
WHERE user_id = $1 AND contact_user_id = $2
