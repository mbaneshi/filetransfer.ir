CREATE TABLE IF NOT EXISTS users (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    phone      text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    last_login timestamptz NOT NULL DEFAULT now()
);
