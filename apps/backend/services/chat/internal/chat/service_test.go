package chat

import (
	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	chatllm "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/sqlrunner"
	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
)

func newTestService(
	storage chatstorage.Storage,
	connections ConnectionService,
	schemaRetriever SchemaRetriever,
	clientResolver chatllm.ClientResolver,
	sqlRunner sqlrunner.SQLRunner,
) *Service {
	return NewService(config.Config{}, nil, storage, connections, schemaRetriever, clientResolver, sqlRunner)
}
