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
WHERE pack_id = ANY($1)
ORDER BY pack_id ASC, sort_order ASC, id ASC
