package chat

import (
	"context"
	"log/slog"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/secrets"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	chatllm "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/sqlrunner"
	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	usertypes "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
)

const (
	defaultHistoryLimit     = 20
	defaultSchemaChunkLimit = 8
	defaultMaxPromptBytes   = 64 * 1024
	defaultResultRowLimit   = 100
	defaultQueryTimeout     = 30 * time.Second
)

//go:generate go tool with-modfile mockery --name ConnectionService --outpkg testing --output ./testing --filename generated__connection_service_mocks.go
type ConnectionService interface {
	GetConnection(ctx context.Context, connectionID int32) (*connectiontypes.Connection, error)
	GetAccess(ctx context.Context, userID int32, connectionID int32) (*connectiontypes.Access, error)
}

type UserService interface {
	GetUser(ctx context.Context, userID int32) (*usertypes.User, error)
}

//go:generate go tool with-modfile mockery --name SchemaRetriever --outpkg testing --output ./testing --filename generated__schema_retriever_mocks.go
type SchemaRetriever interface {
	RetrieveRelevantSchemaContext(ctx context.Context, params schematypes.RetrieveRelevantSchemaContextParams) (*schematypes.RetrievedSchemaContext, error)
}

type Service struct {
	*base.Base

	storage         chatstorage.Storage
	connections     ConnectionService
	users           UserService
	schemaRetriever SchemaRetriever
	clientResolver  chatllm.ClientResolver
	sqlRunner       sqlrunner.SQLRunner
	cipher          secrets.Cipher
}

func NewService(
	cfg config.Config,
	logger *slog.Logger,
	storage chatstorage.Storage,
	connections ConnectionService,
	users UserService,
	schemaRetriever SchemaRetriever,
	clientResolver chatllm.ClientResolver,
	sqlRunner sqlrunner.SQLRunner,
	ciphers ...secrets.Cipher,
) *Service {
	cipher := secrets.Cipher(secrets.PlaintextCipher{})
	if len(ciphers) > 0 && ciphers[0] != nil {
		cipher = ciphers[0]
	}

	return &Service{
		Base:            base.NewBase("chat-core", logger, cfg),
		storage:         storage,
		connections:     connections,
		users:           users,
		schemaRetriever: schemaRetriever,
		clientResolver:  clientResolver,
		sqlRunner:       sqlRunner,
		cipher:          cipher,
	}
}
