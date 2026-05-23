SELECT
    id,
    name,
    slug,
    title,
    thumbnail_url,
    sort_order,
    created_at,
    updated_at
FROM sticker_packs
ORDER BY sort_order ASC, id ASC
