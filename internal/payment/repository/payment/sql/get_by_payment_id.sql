SELECT id, payment_id, user_id, status, amount, subscription_days, payment_url, message, created_at, updated_at
FROM payments
WHERE payment_id = $1;
