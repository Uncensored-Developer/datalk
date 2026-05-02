ALTER TABLE schema_embedding_jobs ADD CONSTRAINT schema_embedding_jobs_snapshot_id_key UNIQUE (snapshot_id);

ALTER TABLE schema_chunks ALTER COLUMN embedding TYPE datalk.VECTOR(768) USING embedding::datalk.VECTOR(768);
