CREATE TABLE schema_snapshots (
    id              SERIAL PRIMARY KEY,
    connection_id   INT NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
    namespace_id    INT NOT NULL REFERENCES connection_namespaces(id) ON DELETE CASCADE,
    schema_hash     TEXT NOT NULL,
    slice_json      JSONB NOT NULL,           -- normalized schema JSON
    status          TEXT NOT NULL CHECK (status IN ('started','completed','failed')),
    error_message   TEXT,
    introspected_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (connection_id, schema_hash)
);

CREATE INDEX schema_snapshots_latest_complete_idx
ON schema_snapshots (connection_id, namespace_id, introspected_at DESC)
WHERE status = 'completed';

CREATE TABLE IF NOT EXISTS schema_chunks (
    id              BIGSERIAL PRIMARY KEY,
    snapshot_id     INT NOT NULL REFERENCES schema_snapshots(id) ON DELETE CASCADE,
    connection_id   INT NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
    namespace_id    INT NOT NULL REFERENCES connection_namespaces(id) ON DELETE CASCADE,
    object_type     TEXT        NOT NULL,          -- 'table', 'column', 'fk', 'view', etc.
    object_name     TEXT        NOT NULL,          -- e.g. 'orders' or 'orders.customer_id'
    schema_json     JSONB       NOT NULL,          -- canonical structured schema snippet
    content         TEXT        NOT NULL,
    embedding       datalk.VECTOR(1536),
    metadata        JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX schema_chunks_snapshot_unique
ON schema_chunks (snapshot_id, object_type, object_name);

CREATE INDEX schema_chunks_filter_idx
ON schema_chunks (connection_id, namespace_id, object_type);

