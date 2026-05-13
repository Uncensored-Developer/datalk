DROP INDEX IF EXISTS chat_message_llm_calls_provider_config_id_idx;
DROP INDEX IF EXISTS chat_message_llm_calls_message_id_idx;
DROP TABLE IF EXISTS chat_message_llm_calls;

DROP TABLE IF EXISTS llm_provider_models;

DROP INDEX IF EXISTS llm_provider_configs_enabled_idx;
DROP TABLE IF EXISTS llm_provider_configs;

DROP TABLE IF EXISTS chat_message_retrievals;

DROP INDEX IF EXISTS chat_message_executions_connection_executed_idx;
DROP TABLE IF EXISTS chat_message_executions;

DROP INDEX IF EXISTS chat_messages_conversation_created_idx;
DROP TABLE IF EXISTS chat_messages;

DROP INDEX IF EXISTS chat_conversations_connection_created_idx;
DROP INDEX IF EXISTS chat_conversations_user_created_idx;
DROP TABLE IF EXISTS chat_conversations;