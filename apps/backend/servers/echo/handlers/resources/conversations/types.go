package conversations

import (
	"encoding/json"
	"time"

	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
)

type createConversationRequest struct {
	ConnectionID int32   `json:"connection_id"`
	Title        *string `json:"title"`
}

type sendMessageRequest struct {
	Content  string            `json:"content"`
	Provider llmtypes.Provider `json:"provider"`
	Model    string            `json:"model"`
}

type conversationResponse struct {
	ID           int64     `json:"id"`
	UserID       int32     `json:"user_id"`
	ConnectionID int32     `json:"connection_id"`
	Title        *string   `json:"title,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type messageResponse struct {
	ID             int64                  `json:"id"`
	ConversationID int64                  `json:"conversation_id"`
	Role           chattype.MessageRole   `json:"role"`
	Content        string                 `json:"content"`
	Provider       *llmtypes.Provider     `json:"provider,omitempty"`
	Model          *string                `json:"model,omitempty"`
	Status         chattype.MessageStatus `json:"status"`
	ErrorMessage   *string                `json:"error_message,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

type messageDetailsResponse struct {
	Message   messageResponse    `json:"message"`
	Execution *executionResponse `json:"execution,omitempty"`
	Retrieval *retrievalResponse `json:"retrieval,omitempty"`
}

type executionResponse struct {
	MessageID          int64                `json:"message_id"`
	ConnectionID       int32                `json:"connection_id"`
	DatabaseKind       string               `json:"database_kind"`
	GeneratedSQL       string               `json:"generated_sql"`
	NormalizedSQL      string               `json:"normalized_sql"`
	Result             chattype.QueryResult `json:"result"`
	ExecutionLatencyMS int32                `json:"execution_latency_ms"`
	ExecutedAt         time.Time            `json:"executed_at"`
}

type retrievalResponse struct {
	MessageID   int64                    `json:"message_id"`
	SnapshotID  int32                    `json:"snapshot_id"`
	QueryText   string                   `json:"query_text"`
	Chunks      []retrievedChunkResponse `json:"chunks"`
	RetrievedAt time.Time                `json:"retrieved_at"`
}

type retrievedChunkResponse struct {
	ChunkID    int64           `json:"chunk_id"`
	ObjectType string          `json:"object_type"`
	ObjectName string          `json:"object_name"`
	Content    string          `json:"content"`
	SchemaJSON json.RawMessage `json:"schema_json"`
	Similarity float32         `json:"similarity"`
}

type assistantTurnResponse struct {
	Conversation     conversationResponse `json:"conversation"`
	UserMessage      messageResponse      `json:"user_message"`
	AssistantMessage messageResponse      `json:"assistant_message"`
	Execution        *executionResponse   `json:"execution,omitempty"`
	Retrieval        *retrievalResponse   `json:"retrieval,omitempty"`
}
