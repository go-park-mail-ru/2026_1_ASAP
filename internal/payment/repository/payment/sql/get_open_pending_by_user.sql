SELECT id, payment_id, user_id, status, amount, subscription_days, payment_url, message, created_at, updated_at
FROM payments
WHERE user_id = $1
  AND status IN ('pending', 'waiting_for_capture')
ORDER BY created_at DESC
LIMIT 1;
