package chat

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	pkglogger "github.com/Uncensored-Developer/datalk/apps/backend/pkg/logger"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/api"
	internalchat "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat"
	chatllm "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm"
	llmanthropic "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm/anthropic"
	llmgemini "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm/gemini"
	llmollama "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm/ollama"
	llmopenai "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm/openai"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/sqlrunner"
	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage/db"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
)

const defaultLLMHTTPTimeout = 60 * time.Second

type Chat struct {
	API          *api.Api
	ModelCatalog *chatllm.Registry
}

func New(
	cfg config.Config,
	conn *sql.DB,
	connections internalchat.ConnectionService,
	schemaRetriever internalchat.SchemaRetriever,
) Chat {
	logger := pkglogger.SetupLogger(cfg)
	storage := chatstorage.NewStorage(conn)
	modelCatalog := chatllm.NewRegistry(storage, providerFactories(defaultLLMHTTPTimeout))
	chatService := internalchat.NewService(
		storage,
		connections,
		schemaRetriever,
		modelCatalog,
		sqlrunner.NewRunner(),
	)

	return Chat{
		API:          api.New(logger, cfg, chatService, modelCatalog),
		ModelCatalog: modelCatalog,
	}
}

func providerFactories(timeout time.Duration) map[llmtypes.Provider]chatllm.ClientFactory {
	httpClient := &http.Client{Timeout: timeout}

	return map[llmtypes.Provider]chatllm.ClientFactory{
		llmtypes.ProviderOpenAI: func(config *llmtypes.ProviderConfig) (chatllm.Client, error) {
			return llmopenai.NewClient(config, httpClient)
		},
		llmtypes.ProviderAnthropic: func(config *llmtypes.ProviderConfig) (chatllm.Client, error) {
			return llmanthropic.NewClient(config, httpClient)
		},
		llmtypes.ProviderGemini: func(config *llmtypes.ProviderConfig) (chatllm.Client, error) {
			return llmgemini.NewClient(config, httpClient)
		},
		llmtypes.ProviderOllama: func(config *llmtypes.ProviderConfig) (chatllm.Client, error) {
			return llmollama.NewClient(config, httpClient)
		},
	}
}
