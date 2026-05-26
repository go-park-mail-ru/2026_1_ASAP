ALTER TABLE message_attachments
  ADD COLUMN IF NOT EXISTS transcript TEXT;
