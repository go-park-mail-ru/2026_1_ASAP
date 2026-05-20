INSERT INTO payments (payment_id, user_id, status, amount, subscription_days, payment_url, message)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, payment_id, user_id, status, amount, subscription_days, payment_url, message, created_at, updated_at;
