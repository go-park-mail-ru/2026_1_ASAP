CREATE TABLE attachments (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL,
    file_url TEXT NOT NULL,
    file_type TEXT,
    file_size BIGINT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);