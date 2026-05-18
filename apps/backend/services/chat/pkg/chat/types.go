package chat

import (
	"encoding/json"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
)

type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
)

type MessageStatus string

const (
	MessageStatusPending   MessageStatus = "pending"
	MessageStatusCompleted MessageStatus = "completed"
	MessageStatusFailed    MessageStatus = "failed"
)

type Conversation struct {
	ID           int64
	UserID       int32
	ConnectionID int32
	Title        *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Message struct {
	ID             int64
	ConversationID int64
	Role           MessageRole
	Content        string
	Provider       *llm.Provider
	Model          *string
	Status         MessageStatus
	ErrorMessage   *string
	CreatedAt      time.Time
}

type MessageExecution struct {
	MessageID          int64
	ConnectionID       int32
	DatabaseKind       connectiontypes.Database
	GeneratedSQL       string
	NormalizedSQL      string
	Result             QueryResult
	ExecutionLatencyMS int32
	ExecutedAt         time.Time
}

type QueryResult struct {
	Columns   []ResultColumn   `json:"columns"`
	Rows      []map[string]any `json:"rows"`
	RowCount  int32            `json:"row_count"`
	Truncated bool             `json:"truncated"`
	Kind      QueryResultKind  `json:"kind,omitempty"`
}

type ResultColumn struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
}

type QueryResultKind string

const (
	QueryResultKindScalar     QueryResultKind = "scalar"
	QueryResultKindRecord     QueryResultKind = "record"
	QueryResultKindTable      QueryResultKind = "table"
	QueryResultKindTimeSeries QueryResultKind = "timeseries"
	QueryResultKindEmpty      QueryResultKind = "empty"
)

type MessageRetrieval struct {
	MessageID   int64
	SnapshotID  int32
	QueryText   string
	Chunks      []schematypes.RetrievedChunk
	RetrievedAt time.Time
}

type MessageLLMCall struct {
	ID               int64
	MessageID        int64
	ProviderConfigID int64
	Provider         llm.Provider
	Model            string
	RequestJSON      json.RawMessage
	ResponseJSON     json.RawMessage
	InputTokens      *int32
	OutputTokens     *int32
	LatencyMS        int32
	CreatedAt        time.Time
}

type AssistantTurn struct {
	Conversation     *Conversation
	UserMessage      *Message
	AssistantMessage *Message
	Execution        *MessageExecution
	Retrieval        *MessageRetrieval
}

type CreateConversationParams struct {
	ConnectionID int32
	Title        *string
}

type SendMessageParams struct {
	UserID         int32
	ConversationID int64
	Content        string
	Provider       llm.Provider
	Model          string
}

type ListConversationsFilter struct {
	ConnectionID []int32
	Limit        int
	Offset       int
}

type ListMessagesFilter struct {
	ConversationID int64
	Limit          int
	Offset         int
}
