CREATE TABLE contacts (
    user_id BIGINT NOT NULL,
    contact_user_id BIGINT NOT NULL,
    contact_name TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (user_id, contact_user_id)
);