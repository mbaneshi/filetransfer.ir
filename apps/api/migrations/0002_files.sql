CREATE TABLE IF NOT EXISTS files (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id     uuid NOT NULL REFERENCES users(id),
    object_key   text NOT NULL UNIQUE,
    name         text NOT NULL,
    size_bytes   bigint NOT NULL CHECK (size_bytes > 0),
    content_type text NOT NULL,
    status       text NOT NULL CHECK (status IN ('uploading', 'ready', 'deleted')),
    upload_id    text,
    part_size    bigint NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz
);
CREATE INDEX IF NOT EXISTS files_owner_idx ON files (owner_id, created_at DESC);
CREATE INDEX IF NOT EXISTS files_stale_idx ON files (created_at) WHERE status = 'uploading';

CREATE TABLE IF NOT EXISTS shares (
    token      text PRIMARY KEY,
    file_id    uuid NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
