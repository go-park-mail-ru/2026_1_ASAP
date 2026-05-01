CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS search_vector tsvector
  GENERATED ALWAYS AS (to_tsvector('simple', COALESCE(content, ''))) STORED;

CREATE INDEX IF NOT EXISTS idx_messages_search_vector_gin
  ON messages USING GIN (search_vector)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_messages_chat_id_id_desc
  ON messages (chat_id, id DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_profiles_first_name_trgm
  ON profiles USING GIN (first_name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_profiles_last_name_trgm
  ON profiles USING GIN (last_name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_users_login_trgm
  ON users USING GIN (login gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_chats_title_trgm
  ON chats USING GIN (title gin_trgm_ops)
  WHERE title IS NOT NULL;
