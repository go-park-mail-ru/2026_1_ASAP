ALTER TABLE users
    ADD COLUMN first_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN last_name  TEXT,
    ADD COLUMN avatar_url TEXT,
    ADD COLUMN bio        TEXT,
    ADD COLUMN birth_date DATE,
    ADD COLUMN last_seen  TIMESTAMPTZ;

UPDATE users u
SET
    first_name = p.first_name,
    last_name  = p.last_name,
    avatar_url = p.avatar_url,
    bio        = p.bio,
    birth_date = p.birth_date,
    last_seen  = p.last_seen
FROM profiles p
WHERE u.id = p.user_id;

DROP TABLE profiles;