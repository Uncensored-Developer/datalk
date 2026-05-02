ALTER TABLE schema_embedding_jobs DROP CONSTRAINT schema_embedding_jobs_snapshot_id_key;

ALTER TABLE schema_chunks ALTER COLUMN embedding TYPE datalk.VECTOR(1536) USING embedding::datalk.VECTOR(1536);
