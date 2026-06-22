package api

import (
	"context"

	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
)

// Client is the client interface to the chat API contract that handlers and other services can depend on.
//
//go:generate go tool with-modfile mockery --name Client --structname API --outpkg testing --output ./testing --filename generated__chat_api_mocks.go
type Client interface {
	CreateConversation(ctx context.Context, userID int32, params CreateConversationParams) (*chattype.Conversation, error)
	GetConversation(ctx context.Context, userID int32, conversationID int64) (*chattype.Conversation, error)
	InferConversationTitle(ctx context.Context, userID int32, conversationID int64) (*chattype.Conversation, error)
	ListConversations(ctx context.Context, userID int32, filter ListConversationsFilter) ([]*chattype.Conversation, error)
	DeleteConversation(ctx context.Context, userID int32, conversationID int64) error
	ListMessages(ctx context.Context, userID int32, filter ListMessagesFilter) ([]*chattype.MessageDetails, error)
	SendMessage(ctx context.Context, params SendMessageParams) (*chattype.AssistantTurn, error)
	SendMessageWithProgress(ctx context.Context, params SendMessageParams, progress chattype.SendMessageProgressHandler) (*chattype.AssistantTurn, error)
	ListProviderConfigs(ctx context.Context) ([]*llmtypes.ProviderConfig, error)
	SaveProviderConfig(ctx context.Context, params SaveProviderConfigParams) (*llmtypes.ProviderConfig, error)
	TestProviderConfig(ctx context.Context, params TestProviderConfigParams) (*TestProviderConfigResult, error)
	ListAvailableModels(ctx context.Context) ([]llmtypes.Model, error)
}
