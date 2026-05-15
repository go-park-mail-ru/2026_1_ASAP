ALTER TABLE messages
DROP CONSTRAINT IF EXISTS chk_messages_content;

ALTER TABLE messages
ADD CONSTRAINT chk_messages_content
CHECK (content IS NOT NULL OR sticker_id IS NOT NULL);
