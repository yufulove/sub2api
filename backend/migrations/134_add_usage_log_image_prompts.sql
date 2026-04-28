ALTER TABLE usage_logs
  ADD COLUMN IF NOT EXISTS image_requested_size VARCHAR(32),
  ADD COLUMN IF NOT EXISTS image_prompt TEXT,
  ADD COLUMN IF NOT EXISTS image_revised_prompt TEXT;
