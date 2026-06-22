package storage

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/ordering"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pagination"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
)

type ConversationOrdering int

const (
	ConversationOrderingCreatedAt ConversationOrdering = iota
	ConversationOrderingUpdatedAt
	ConversationOrderingID
)

type MessageOrdering int

const (
	MessageOrderingCreatedAt MessageOrdering = iota
	MessageOrderingID
)

type LLMCallOrdering int

const (
	LLMCallOrderingCreatedAt LLMCallOrdering = iota
	LLMCallOrderingID
)

type ConversationsFilter struct {
	ID           []int64
	UserID       []int32
	ConnectionID []int32
	Pagination   pagination.LimitOffsetPagination
	Ordering     ordering.Orderings[ConversationOrdering]
}

type MessagesFilter struct {
	ID             []int64
	ConversationID []int64
	Role           []chattype.MessageRole
	Status         []chattype.MessageStatus
	Pagination     pagination.LimitOffsetPagination
	Ordering       ordering.Orderings[MessageOrdering]
}

type LLMCallsFilter struct {
	ID               []int64
	MessageID        []int64
	ProviderConfigID []int64
	Pagination       pagination.LimitOffsetPagination
	Ordering         ordering.Orderings[LLMCallOrdering]
}

type ProviderConfigsFilter struct {
	ID        []int64
	Provider  []llm.Provider
	IsEnabled *bool
}

type ProviderModelsFilter struct {
	ID               []int64
	ProviderConfigID []int64
	Model            []string
	IsEnabled        *bool
	SupportsSQL      *bool
}

//go:generate go tool with-modfile mockery --name Storage --outpkg testing --output ./testing --filename generated__storage_mocks.go
type Storage interface {
	InTransaction(ctx context.Context, fn func(ctx context.Context) error) error

	UpsertConversation(ctx context.Context, conversation *chattype.Conversation) error
	GetConversation(ctx context.Context, id int64) (*chattype.Conversation, error)
	ListConversations(ctx context.Context, filter ConversationsFilter) ([]*chattype.Conversation, error)
	DeleteConversation(ctx context.Context, id int64) error

	InsertMessage(ctx context.Context, message *chattype.Message) error
	GetMessage(ctx context.Context, id int64) (*chattype.Message, error)
	ListMessages(ctx context.Context, filter MessagesFilter) ([]*chattype.Message, error)

	InsertExecution(ctx context.Context, execution *chattype.MessageExecution) error
	GetExecution(ctx context.Context, messageID int64) (*chattype.MessageExecution, error)

	InsertRetrieval(ctx context.Context, retrieval *chattype.MessageRetrieval) error
	GetRetrieval(ctx context.Context, messageID int64) (*chattype.MessageRetrieval, error)

	InsertLLMCall(ctx context.Context, call *chattype.MessageLLMCall) error
	ListLLMCalls(ctx context.Context, filter LLMCallsFilter) ([]*chattype.MessageLLMCall, error)

	UpsertProviderConfig(ctx context.Context, config *llm.ProviderConfig) error
	GetProviderConfig(ctx context.Context, id int64) (*llm.ProviderConfig, error)
	ListProviderConfigs(ctx context.Context, filter ProviderConfigsFilter) ([]*llm.ProviderConfig, error)

	UpsertProviderModel(ctx context.Context, model *llm.ProviderModel) error
	ListProviderModels(ctx context.Context, filter ProviderModelsFilter) ([]*llm.ProviderModel, error)
}
