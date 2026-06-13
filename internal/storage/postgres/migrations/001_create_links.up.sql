CREATE TABLE IF NOT EXISTS links (
    code   CHAR(10)     PRIMARY KEY,
    original_url TEXT         NOT NULL UNIQUE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);