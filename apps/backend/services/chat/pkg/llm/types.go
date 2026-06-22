package llm

import (
	"encoding/json"
	"time"

	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
)

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderGemini    Provider = "gemini"
	ProviderOllama    Provider = "ollama"
)

type Model struct {
	ID           string
	Provider     Provider
	DisplayName  string
	Description  *string
	IsEnabled    bool
	Capabilities ModelCapabilities
}

type ModelCapabilities struct {
	SupportsToolCalling        bool
	SupportsStructuredOutput   bool
	SupportsStreaming          bool
	SupportsSystemInstructions bool
	SupportsVision             bool
	MaxContextTokens           *int
	MaxOutputTokens            *int
}

type ProviderConfig struct {
	ID          int64
	Provider    Provider
	DisplayName string
	APIKeyEnc   string
	BaseURL     *string
	IsEnabled   bool
	Metadata    json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ProviderModel struct {
	ID               int64
	ProviderConfigID int64
	Model            string
	DisplayName      string
	ContextWindow    *int32
	SupportsSQL      bool
	IsEnabled        bool
	DiscoveredAt     time.Time
}

type ConversationContext struct {
	ConversationID int64
	ConnectionID   int32
	DatabaseKind   connectiontypes.Database
	History        []ConversationMessage
}

type ConversationMessage struct {
	Role    string
	Content string
}

type GenerateSQLOptions struct {
	MaxHistoryMessages int
	MaxSchemaChunks    int
	MaxPromptBytes     int
	RequireReadOnly    bool
	RequireSingleStmt  bool
	AllowedDatabases   []connectiontypes.Database
}

type GenerateSQLRequest struct {
	Model        string
	Conversation ConversationContext
	UserPrompt   string
	Schema       schematypes.RetrievedSchemaContext
	Options      GenerateSQLOptions
	Correction   *SQLCorrectionContext
}

type SQLCorrectionContext struct {
	AttemptNumber int
	Attempts      []SQLCorrectionAttempt
}

type SQLCorrectionAttempt struct {
	SQL   string
	Error string
}

type GenerateSQLResponse struct {
	SQL          string
	Explanation  string
	Assumptions  []string
	Confidence   *float32
	FinishReason *string
	Usage        *Usage
	RawRequest   json.RawMessage
	RawResponse  json.RawMessage
}

type GenerateAnswerOptions struct {
	MaxHistoryMessages int
	MaxResultRows      int
	MaxResultBytes     int
}

type GenerateAnswerRequest struct {
	Model        string
	Conversation ConversationContext
	UserPrompt   string
	GeneratedSQL string
	DatabaseKind connectiontypes.Database
	Result       QueryResultPreview
	Options      GenerateAnswerOptions
}

type QueryResultPreview struct {
	Columns   []QueryResultColumn `json:"columns"`
	Rows      []map[string]any    `json:"rows"`
	RowCount  int32               `json:"row_count"`
	Truncated bool                `json:"truncated"`
	Kind      string              `json:"kind,omitempty"`
}

type QueryResultColumn struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
}

type GenerateAnswerResponse struct {
	Answer       string
	Limitations  []string
	FinishReason *string
	Usage        *Usage
	RawRequest   json.RawMessage
	RawResponse  json.RawMessage
}

type GenerateConversationTitleRequest struct {
	Model      string
	UserPrompt string
	Assistant  string
	MaxWords   int
	MaxChars   int
}

type GenerateConversationTitleResponse struct {
	Title        string
	FinishReason *string
	Usage        *Usage
	RawRequest   json.RawMessage
	RawResponse  json.RawMessage
}

type Usage struct {
	InputTokens  *int
	OutputTokens *int
	TotalTokens  *int
}
