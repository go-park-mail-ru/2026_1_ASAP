DROP INDEX IF EXISTS idx_chats_title_trgm;

DROP INDEX IF EXISTS idx_users_login_trgm;
DROP INDEX IF EXISTS idx_profiles_last_name_trgm;
DROP INDEX IF EXISTS idx_profiles_first_name_trgm;

DROP INDEX IF EXISTS idx_messages_chat_id_id_desc;
DROP INDEX IF EXISTS idx_messages_search_vector_gin;

ALTER TABLE messages DROP COLUMN IF EXISTS search_vector;
