SELECT
    id,
    pack_id,
    file_url,
    slug,
    emoji,
    width,
    height,
    sort_order,
    created_at,
    updated_at
FROM stickers
WHERE id = ANY($1)
