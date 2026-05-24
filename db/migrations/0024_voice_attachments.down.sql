ALTER TABLE message_attachments
  DROP COLUMN IF EXISTS waveform,
  DROP COLUMN IF EXISTS duration_ms;

-- PostgreSQL does not support removing enum values; voice remains in message_attachment_type.
