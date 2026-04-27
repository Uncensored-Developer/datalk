DROP INDEX IF EXISTS schema_snapshots_latest_complete_idx;

ALTER TABLE schema_snapshots DROP COLUMN status;

ALTER TABLE schema_snapshots DROP COLUMN error_message;

CREATE TABLE IF NOT EXISTS schema_embedding_jobs (
    snapshot_id       INT NOT NULL REFERENCES schema_snapshots(id) ON DELETE CASCADE,
    status            TEXT NOT NULL CHECK (status IN ('PENDING','PROCESSING','COMPLETED','FAILED')),
    error_message     TEXT,
    retry_count       INT NOT NULL DEFAULT 0,
    started_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at      TIMESTAMPTZ
);
