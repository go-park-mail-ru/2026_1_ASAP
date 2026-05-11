UPDATE payments
SET
    status = $2,
    amount = $3,
    subscription_days = $4,
    payment_url = $5,
    message = $6,
    updated_at = now()
WHERE payment_id = $1
RETURNING id, payment_id, user_id, status, amount, subscription_days, payment_url, message, created_at, updated_at;
