CREATE TABLE IF NOT EXISTS studio_image_assets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id BIGINT REFERENCES groups(id) ON DELETE SET NULL,
    request_id TEXT NOT NULL,
    client_id TEXT NOT NULL DEFAULT '',
    image_index INTEGER NOT NULL DEFAULT 0,
    model TEXT NOT NULL DEFAULT '',
    size TEXT NOT NULL DEFAULT '',
    prompt TEXT NOT NULL DEFAULT '',
    revised_prompt TEXT NOT NULL DEFAULT '',
    storage_path TEXT NOT NULL,
    content_type VARCHAR(64) NOT NULL DEFAULT 'image/png',
    byte_size BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, request_id, image_index)
);

CREATE INDEX IF NOT EXISTS idx_studio_image_assets_user_created
    ON studio_image_assets (user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_studio_image_assets_request
    ON studio_image_assets (request_id);
