CREATE TABLE stickers (
    id UUID PRIMARY KEY,
    pack_id UUID NOT NULL,
    file_url TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);