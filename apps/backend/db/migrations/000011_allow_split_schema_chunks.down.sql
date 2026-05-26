DROP INDEX IF EXISTS schema_chunks_snapshot_unique;

CREATE UNIQUE INDEX schema_chunks_snapshot_unique
ON schema_chunks (snapshot_id, object_type, object_name);
