CREATE TABLE profiles (
    user_id    BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    first_name TEXT NOT NULL DEFAULT '',
    last_name  TEXT,
    avatar_url TEXT,
    bio        TEXT,
    birth_date DATE,
    last_seen  TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO profiles (user_id, first_name, last_name, avatar_url, bio, birth_date, last_seen)
SELECT id, first_name, last_name, avatar_url, bio, birth_date, last_seen
FROM users;

ALTER TABLE users
    DROP COLUMN first_name,
    DROP COLUMN last_name,
    DROP COLUMN avatar_url,
    DROP COLUMN bio,
    DROP COLUMN birth_date,
    DROP COLUMN last_seen;