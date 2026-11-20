
ALTER TABLE monitors
    ADD COLUMN lease_owner TEXT,
    ADD COLUMN lease_expires_at TIMESTAMPTZ;

CREATE INDEX idx_monitors_due
    ON monitors (next_run_at, lease_expires_at)
    WHERE enabled = TRUE;
