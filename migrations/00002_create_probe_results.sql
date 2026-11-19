
CREATE TABLE probe_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    monitor_id UUID NOT NULL
        REFERENCES monitors(id)
        ON DELETE CASCADE,

    status VARCHAR(20) NOT NULL
        CHECK (status IN ('success', 'failure')),

    status_code INTEGER
        CHECK (status_code BETWEEN 100 AND 599),

    latency_ms BIGINT NOT NULL
        CHECK (latency_ms >= 0),

    error_message TEXT,

    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_probe_results_monitor_checked
    ON probe_results (monitor_id, checked_at DESC);

CREATE INDEX idx_probe_results_checked_at
    ON probe_results (checked_at DESC);
