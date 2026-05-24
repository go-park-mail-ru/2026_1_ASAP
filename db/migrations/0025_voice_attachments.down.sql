ALTER TABLE message_attachments
  DROP COLUMN IF EXISTS waveform,
  DROP COLUMN IF EXISTS duration_ms;

