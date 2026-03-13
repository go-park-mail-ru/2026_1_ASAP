CREATE TABLE reactions (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL,
    user_id UUID NOT NULL,
    emoji TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);