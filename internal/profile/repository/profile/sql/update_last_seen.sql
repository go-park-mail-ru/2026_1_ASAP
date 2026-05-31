UPDATE profiles
SET last_seen = NOW()
WHERE user_id = $1
