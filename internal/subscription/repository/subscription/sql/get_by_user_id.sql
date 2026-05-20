SELECT user_id, active, start_at, end_at
FROM subscriptions
WHERE user_id = $1;
