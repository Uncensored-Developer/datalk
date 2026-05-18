package chat

import (
	"context"
	"time"

	chatllm "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/sqlrunner"
	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
)

const (
	defaultHistoryLimit     = 20
	defaultSchemaChunkLimit = 8
	defaultResultRowLimit   = 100
	defaultQueryTimeout     = 30 * time.Second
)

//go:generate go tool with-modfile mockery --name ConnectionService --outpkg testing --output ./testing --filename generated__connection_service_mocks.go
type ConnectionService interface {
	GetConnection(ctx context.Context, connectionID int32) (*connectiontypes.Connection, error)
	GetAccess(ctx context.Context, userID int32, connectionID int32) (*connectiontypes.Access, error)
}

//go:generate go tool with-modfile mockery --name SchemaRetriever --outpkg testing --output ./testing --filename generated__schema_retriever_mocks.go
type SchemaRetriever interface {
	RetrieveRelevantSchemaContext(ctx context.Context, params schematypes.RetrieveRelevantSchemaContextParams) (*schematypes.RetrievedSchemaContext, error)
}

type Service struct {
	storage         chatstorage.Storage
	connections     ConnectionService
	schemaRetriever SchemaRetriever
	clientResolver  chatllm.ClientResolver
	sqlRunner       sqlrunner.SQLRunner
}

func NewService(
	storage chatstorage.Storage,
	connections ConnectionService,
	schemaRetriever SchemaRetriever,
	clientResolver chatllm.ClientResolver,
	sqlRunner sqlrunner.SQLRunner,
) *Service {
	return &Service{
		storage:         storage,
		connections:     connections,
		schemaRetriever: schemaRetriever,
		clientResolver:  clientResolver,
		sqlRunner:       sqlRunner,
	}
}
