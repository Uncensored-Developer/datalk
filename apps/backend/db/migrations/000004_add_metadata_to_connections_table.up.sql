ALTER TABLE connections ADD COLUMN metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

DROP TABLE IF EXISTS connection_namespaces;