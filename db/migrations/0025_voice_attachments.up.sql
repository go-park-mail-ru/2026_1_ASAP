ALTER TYPE message_attachment_type ADD VALUE IF NOT EXISTS 'voice';

ALTER TABLE message_attachments
  ADD COLUMN IF NOT EXISTS duration_ms INT,
  ADD COLUMN IF NOT EXISTS waveform JSONB;
