CREATE TABLE messages (
    id BIGSERIAL PRIMARY KEY,
    chat_id UUID NOT NULL,
    sender_id UUID NOT NULL,
    content TEXT,
    sticker_id UUID,
    edited BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ
);