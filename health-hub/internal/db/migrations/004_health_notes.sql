CREATE TABLE IF NOT EXISTS health_notes (
    id        BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    time      TIMESTAMPTZ NOT NULL,
    category  TEXT DEFAULT 'memo',
    text      TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_notes_time ON health_notes (time DESC);
CREATE INDEX IF NOT EXISTS idx_notes_category ON health_notes (category, time DESC);
