INSERT INTO subscriptions (user_id, active, start_at, end_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE SET
    active = EXCLUDED.active,
    start_at = EXCLUDED.start_at,
    end_at = EXCLUDED.end_at
RETURNING user_id, active, start_at, end_at;
