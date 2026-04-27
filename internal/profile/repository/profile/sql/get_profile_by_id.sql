SELECT user_id, first_name, last_name, avatar_url, bio, birth_date, last_seen
FROM profiles WHERE user_id=$1
