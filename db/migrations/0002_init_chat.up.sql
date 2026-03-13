CREATE TABLE chats (
    id UUID PRIMARY KEY,
    type TEXT NOT NULL,
    title TEXT,
    description TEXT,
    owner_id UUID,
    avatar_url TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);