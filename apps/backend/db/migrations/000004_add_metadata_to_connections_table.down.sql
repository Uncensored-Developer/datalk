ALTER TABLE connections DROP COLUMN metadata;

CREATE TABLE connection_namespaces (
    id             SERIAL PRIMARY KEY,
    connection_id  INT NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    namespace_type TEXT NOT NULL CHECK (namespace_type IN ('schema','database','keyspace')),
    is_enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (connection_id, namespace_type, name)
);