export type UserRole = "owner" | "admin" | "member";

export type User = {
  id: number;
  email: string;
  name: string;
  role: UserRole;
  is_active?: boolean;
  must_change_password: boolean;
};

export type TokenPair = {
  access_token: string;
  refresh_token: string;
  expires_at: string;
};

export type AuthSession = {
  user: User;
  tokens: TokenPair;
  must_change_password: boolean;
};

export type ApiErrorShape = {
  error: string;
};

export type DatabaseKind = "postgres" | "mysql" | "cql";

export type ConnectionMetadata = {
  include_namespaces?: string[] | null;
  exclude_namespaces?: string[] | null;
  include_tables_by_namespace?: Record<string, string[]> | null;
  exclude_tables_by_namespace?: Record<string, string[]> | null;
  include_views?: boolean;
  include_indexes?: boolean;
  include_foreign_keys?: boolean;
  include_comments?: boolean;
};

export type Connection = {
  id: number;
  name: string;
  database: DatabaseKind;
  user_id: number;
  is_enabled: boolean;
  metadata?: ConnectionMetadata | null;
};

export type ConnectionAccessGrant = {
  user_id: number;
  connection_id: number;
  can_query: boolean;
  allow_writes: boolean;
  can_manage: boolean;
};

export type SchemaRefreshResponse = {
  connection_id: number;
  status: "accepted";
};

export type Provider = "openai" | "anthropic" | "gemini" | "ollama";

export type ProviderConfig = {
  id: number;
  provider: Provider;
  display_name: string;
  base_url: string;
  is_enabled: boolean;
  has_api_key: boolean;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type ModelCapabilities = {
  supports_tool_calling: boolean;
  supports_structured_output: boolean;
  supports_streaming: boolean;
  supports_system_instructions: boolean;
  supports_vision: boolean;
  max_context_tokens: number;
  max_output_tokens: number;
};

export type ChatModel = {
  id: string;
  provider: Provider;
  display_name: string;
  description?: string;
  is_enabled: boolean;
  capabilities: ModelCapabilities;
};

export type Conversation = {
  id: number;
  user_id: number;
  connection_id: number;
  title: string;
  created_at: string;
  updated_at: string;
};

export type ChatMessageRole = "user" | "assistant" | "system";
export type ChatMessageStatus = "pending" | "completed" | "failed";

export type ChatMessage = {
  id: number;
  conversation_id: number;
  role: ChatMessageRole;
  content: string;
  provider?: Provider;
  model?: string;
  status: ChatMessageStatus;
  error_message?: string;
  created_at: string;
};

export type QueryResultColumn = {
  name: string;
  data_type: string;
};

export type QueryResult = {
  columns: QueryResultColumn[];
  rows: Record<string, unknown>[];
  row_count: number;
  truncated: boolean;
  kind: string;
};

export type MessageExecution = {
  message_id: number;
  connection_id: number;
  database_kind: DatabaseKind;
  generated_sql: string;
  normalized_sql: string;
  result: QueryResult;
  execution_latency_ms: number;
  executed_at: string;
};

export type MessageRetrieval = {
  message_id: number;
  snapshot_id: number;
  query_text: string;
  retrieved_at: string;
};

export type MessageListItem = {
  message: ChatMessage;
  execution?: MessageExecution;
  retrieval?: MessageRetrieval;
};

export type SendMessageResponse = {
  conversation: Conversation;
  user_message: ChatMessage;
  assistant_message: ChatMessage;
  execution?: MessageExecution;
  retrieval?: MessageRetrieval;
};
