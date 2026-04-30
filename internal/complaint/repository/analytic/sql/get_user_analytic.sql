SELECT
    COUNT(*) FILTER (WHERE status = 'opened' OR status = 'new') AS count_status_opened,
    COUNT(*) FILTER (WHERE status = 'in_progress') AS count_status_in_work,
    COUNT(*) FILTER (WHERE status = 'closed') AS count_status_closed,
    COUNT(*) FILTER (WHERE type = 'bug') AS count_type_bug,
    COUNT(*) FILTER (WHERE type = 'upgrade') AS count_type_upgrade,
    COUNT(*) FILTER (WHERE type = 'product') AS count_type_product
FROM complaint
WHERE user_id = $1;
