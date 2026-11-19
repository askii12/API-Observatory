CREATE TABLE monitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    name VARCHAR(120) NOT NULL,
    url TEXT NOT NULL,

    interval_seconds INTEGER NOT NULL DEFAULT 60
        CHECK (interval_seconds BETWEEN 30 AND 86400),

    timeout_ms INTEGER NOT NULL DEFAULT 5000
        CHECK (timeout_ms BETWEEN 500 AND 30000),

    enabled BOOLEAN NOT NULL DEFAULT TRUE,

    next_run_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_monitors_next_run
    ON monitors (next_run_at)
    WHERE enabled = TRUE;

CREATE INDEX idx_monitors_created_at
    ON monitors (created_at DESC);