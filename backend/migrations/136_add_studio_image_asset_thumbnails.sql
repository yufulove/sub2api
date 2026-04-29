ALTER TABLE studio_image_assets
    ADD COLUMN IF NOT EXISTS thumbnail_path TEXT NOT NULL DEFAULT '';
