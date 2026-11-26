-- Prepaid plans. A row is one purchased or hand-activated period.
-- Cancelling sets expires_at to the moment of cancellation and records cancelled_at.
CREATE TABLE IF NOT EXISTS subscriptions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid NOT NULL REFERENCES users(id),
    plan         text NOT NULL CHECK (plan IN ('basic', 'max', 'ultra')),
    starts_at    timestamptz NOT NULL,
    expires_at   timestamptz NOT NULL,
    cancelled_at timestamptz,
    note         text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    CHECK (expires_at >= starts_at)
);
CREATE INDEX IF NOT EXISTS subscriptions_user_idx ON subscriptions (user_id, starts_at DESC);

-- One row each time a share link hands out a download. The sum per owner is the traffic used.
CREATE TABLE IF NOT EXISTS downloads (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    file_id     uuid NOT NULL REFERENCES files(id),
    owner_id    uuid NOT NULL REFERENCES users(id),
    share_token text NOT NULL,
    size_bytes  bigint NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS downloads_owner_idx ON downloads (owner_id, created_at);

-- When the plan's retention removes the file. NULL (files from before plans existed) means no retention date.
ALTER TABLE files ADD COLUMN IF NOT EXISTS expires_at timestamptz;
CREATE INDEX IF NOT EXISTS files_expiry_idx ON files (expires_at) WHERE status = 'ready';
