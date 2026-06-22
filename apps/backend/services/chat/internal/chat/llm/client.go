package llm

import (
	"context"

	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
)

//go:generate go tool with-modfile mockery --name Client --outpkg testing --output ./testing --filename generated__client_mocks.go
type Client interface {
	ListModels(ctx context.Context) ([]llmtypes.Model, error)
	GenerateSQL(ctx context.Context, req llmtypes.GenerateSQLRequest) (*llmtypes.GenerateSQLResponse, error)
	GenerateAnswer(ctx context.Context, req llmtypes.GenerateAnswerRequest) (*llmtypes.GenerateAnswerResponse, error)
	GenerateConversationTitle(ctx context.Context, req llmtypes.GenerateConversationTitleRequest) (*llmtypes.GenerateConversationTitleResponse, error)
}

type ClientFactory func(config *llmtypes.ProviderConfig) (Client, error)
