package api

import (
	"context"
	"log/slog"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
)

//go:generate go tool with-modfile mockery --name Service --outpkg testing --output ./testing --filename generated__chat_service_mocks.go
type Service interface {
	CreateConversation(ctx context.Context, userID int32, params chattype.CreateConversationParams) (*chattype.Conversation, error)
	GetConversation(ctx context.Context, userID int32, conversationID int64) (*chattype.Conversation, error)
	ListConversations(ctx context.Context, userID int32, filter chattype.ListConversationsFilter) ([]*chattype.Conversation, error)
	DeleteConversation(ctx context.Context, userID int32, conversationID int64) error
	ListMessages(ctx context.Context, userID int32, filter chattype.ListMessagesFilter) ([]*chattype.MessageDetails, error)
	SendMessage(ctx context.Context, params chattype.SendMessageParams) (*chattype.AssistantTurn, error)
	SendMessageWithProgress(ctx context.Context, params chattype.SendMessageParams, progress chattype.SendMessageProgressHandler) (*chattype.AssistantTurn, error)
	ListProviderConfigs(ctx context.Context) ([]*llmtypes.ProviderConfig, error)
	SaveProviderConfig(ctx context.Context, params chat.SaveProviderConfigParams) (*llmtypes.ProviderConfig, error)
}

//go:generate go tool with-modfile mockery --name ModelCatalog --outpkg testing --output ./testing --filename generated__model_catalog_mocks.go
type ModelCatalog interface {
	ListAvailableModels(ctx context.Context) ([]llmtypes.Model, error)
}

type Api struct {
	*base.Base
	service      Service
	modelCatalog ModelCatalog
}

func New(logger *slog.Logger, cfg config.Config, service Service, modelCatalog ModelCatalog) *Api {
	return &Api{
		Base:         base.NewBase("chat", logger, cfg),
		service:      service,
		modelCatalog: modelCatalog,
	}
}

func (a *Api) CreateConversation(ctx context.Context, userID int32, params CreateConversationParams) (*chattype.Conversation, error) {
	return a.service.CreateConversation(ctx, userID, params)
}

func (a *Api) GetConversation(ctx context.Context, userID int32, conversationID int64) (*chattype.Conversation, error) {
	return a.service.GetConversation(ctx, userID, conversationID)
}

func (a *Api) ListConversations(ctx context.Context, userID int32, filter ListConversationsFilter) ([]*chattype.Conversation, error) {
	return a.service.ListConversations(ctx, userID, filter)
}

func (a *Api) DeleteConversation(ctx context.Context, userID int32, conversationID int64) error {
	return a.service.DeleteConversation(ctx, userID, conversationID)
}

func (a *Api) ListMessages(ctx context.Context, userID int32, filter ListMessagesFilter) ([]*chattype.MessageDetails, error) {
	return a.service.ListMessages(ctx, userID, filter)
}

func (a *Api) SendMessage(ctx context.Context, params SendMessageParams) (*chattype.AssistantTurn, error) {
	return a.service.SendMessage(ctx, params)
}

func (a *Api) SendMessageWithProgress(ctx context.Context, params SendMessageParams, progress chattype.SendMessageProgressHandler) (*chattype.AssistantTurn, error) {
	return a.service.SendMessageWithProgress(ctx, params, progress)
}

func (a *Api) ListProviderConfigs(ctx context.Context) ([]*llmtypes.ProviderConfig, error) {
	return a.service.ListProviderConfigs(ctx)
}

func (a *Api) SaveProviderConfig(ctx context.Context, params chat.SaveProviderConfigParams) (*llmtypes.ProviderConfig, error) {
	return a.service.SaveProviderConfig(ctx, params)
}

func (a *Api) ListAvailableModels(ctx context.Context) ([]llmtypes.Model, error) {
	return a.modelCatalog.ListAvailableModels(ctx)
}
