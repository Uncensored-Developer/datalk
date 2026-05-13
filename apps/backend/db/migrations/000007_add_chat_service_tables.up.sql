CREATE TABLE chat_conversations (
    id            BIGSERIAL PRIMARY KEY,
    user_id       INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    connection_id INT NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
    title         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX chat_conversations_user_created_idx ON chat_conversations (user_id, created_at DESC);

CREATE INDEX chat_conversations_connection_created_idx ON chat_conversations (connection_id, created_at DESC);



CREATE TABLE chat_messages (
    id               BIGSERIAL PRIMARY KEY,
    conversation_id  BIGINT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    role             TEXT NOT NULL CHECK (role IN ('user','assistant')),
    content          TEXT NOT NULL,
    provider         TEXT CHECK (provider IN ('openai','anthropic','gemini','ollama')),
    model            TEXT,
    status           TEXT NOT NULL DEFAULT 'completed' CHECK (status IN ('pending','completed','failed')),
    error_message    TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX chat_messages_conversation_created_idx ON chat_messages (conversation_id, created_at ASC);



CREATE TABLE chat_message_executions (
    message_id              BIGINT PRIMARY KEY REFERENCES chat_messages(id) ON DELETE CASCADE,
    connection_id           INT NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
    database_kind           TEXT NOT NULL CHECK (database_kind IN ('postgres','mysql','cql')),
    generated_sql           TEXT NOT NULL,
    normalized_sql          TEXT NOT NULL,
    result_json             JSONB NOT NULL DEFAULT '{}'::jsonb,
    execution_latency_ms    INT NOT NULL DEFAULT 0,
    executed_at             TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX chat_message_executions_connection_executed_idx ON chat_message_executions (connection_id, executed_at DESC);



CREATE TABLE chat_message_retrievals (
    message_id       BIGINT PRIMARY KEY REFERENCES chat_messages(id) ON DELETE CASCADE,
    snapshot_id      INT NOT NULL REFERENCES schema_snapshots(id) ON DELETE CASCADE,
    query_text       TEXT NOT NULL,
    chunks           JSONB NOT NULL DEFAULT '[]'::jsonb,
    retrieved_at     TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);



CREATE TABLE llm_provider_configs (
    id            BIGSERIAL PRIMARY KEY,
    provider      TEXT NOT NULL CHECK (provider IN ('openai','anthropic','gemini','ollama')),
    display_name  TEXT NOT NULL,
    api_key_enc   TEXT NOT NULL,
    base_url      TEXT,
    is_enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX llm_provider_configs_enabled_idx ON llm_provider_configs (is_enabled, provider);



CREATE TABLE llm_provider_models (
    id               BIGSERIAL PRIMARY KEY,
    provider_config_id BIGINT NOT NULL REFERENCES llm_provider_configs(id) ON DELETE CASCADE,
    model            TEXT NOT NULL,
    display_name     TEXT NOT NULL,
    context_window   INT,
    supports_sql     BOOLEAN NOT NULL DEFAULT TRUE,
    is_enabled       BOOLEAN NOT NULL DEFAULT TRUE,
    discovered_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (provider_config_id, model)
);



CREATE TABLE chat_message_llm_calls (
    id                  BIGSERIAL PRIMARY KEY,
    message_id          BIGINT NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
    provider_config_id  BIGINT NOT NULL REFERENCES llm_provider_configs(id) ON DELETE RESTRICT,
    provider            TEXT NOT NULL CHECK (provider IN ('openai','anthropic','gemini','ollama')),
    model               TEXT NOT NULL,
    request_json        JSONB NOT NULL DEFAULT '{}'::jsonb,
    response_json       JSONB NOT NULL DEFAULT '{}'::jsonb,
    input_tokens        INT,
    output_tokens       INT,
    latency_ms          INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX chat_message_llm_calls_message_id_idx ON chat_message_llm_calls (message_id, created_at DESC);

CREATE INDEX chat_message_llm_calls_provider_config_id_idx ON chat_message_llm_calls (provider_config_id, created_at DESC);
