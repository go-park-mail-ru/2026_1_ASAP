ALTER TABLE messages
DROP CONSTRAINT IF EXISTS chk_messages_content;

ALTER TABLE messages
ADD CONSTRAINT chk_messages_content
CHECK (deleted_at IS NOT NULL OR content IS NOT NULL OR sticker_id IS NOT NULL);

DROP INDEX IF EXISTS idx_message_attachments_message_id;
DROP TABLE IF EXISTS message_attachments;
DROP TYPE IF EXISTS message_attachment_type;
