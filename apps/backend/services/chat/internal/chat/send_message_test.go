package chat

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	chatllm "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm"
	llmtesting "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm/testing"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/sqlrunner"
	sqlrunnertesting "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/sqlrunner/testing"
	chattesting "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/testing"
	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage/testing"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	connectionerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/errors"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/gotidy/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_SendMessage_HappyPath(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	userID := int32(7)
	conversation := &chattype.Conversation{ID: 10, UserID: userID, ConnectionID: 42}
	connection := &connectiontypes.Connection{ID: 42, Database: connectiontypes.DatabasePostgres, DSN: "postgres://warehouse"}
	previousUser := &chattype.Message{ID: 80, ConversationID: conversation.ID, Role: chattype.MessageRoleUser, Content: "how many users signed up", Status: chattype.MessageStatusCompleted}
	previousAssistant := &chattype.Message{ID: 90, ConversationID: conversation.ID, Role: chattype.MessageRoleAssistant, Content: "counted users", Status: chattype.MessageStatusCompleted}
	previousSQL := "select count(*) from users"
	schemaContext := &schematypes.RetrievedSchemaContext{
		ConnectionID:   conversation.ConnectionID,
		SnapshotID:     55,
		EmbeddingModel: "nomic-embed-text",
		QueryText:      "retrieval query",
		Chunks: []schematypes.RetrievedChunk{
			{ChunkID: 1, ObjectType: "table", ObjectName: "users", Content: "table users(id int)", Similarity: 0.91},
		},
		RetrievedAt: time.Now().UTC(),
	}
	queryResult := &chattype.QueryResult{
		Columns:  []chattype.ResultColumn{{Name: "count", DataType: "INT8"}},
		Rows:     []map[string]any{{"count": int64(3)}},
		RowCount: 1,
		Kind:     chattype.QueryResultKindScalar,
	}

	mockStorage := storagetesting.NewStorage(t)
	mockConnections := chattesting.NewConnectionService(t)
	mockSchemas := chattesting.NewSchemaRetriever(t)
	mockModels := llmtesting.NewClientResolver(t)
	mockSQLRunner := sqlrunnertesting.NewSQLRunner(t)
	mockClient := llmtesting.NewClient(t)

	mockStorage.On("GetConversation", ctx, conversation.ID).Return(conversation, nil).Once()
	mockConnections.On("GetConnection", ctx, conversation.ConnectionID).Return(connection, nil).Once()
	mockConnections.On("GetAccess", ctx, userID, conversation.ConnectionID).Return(&connectiontypes.Access{UserID: userID, ConnectionID: conversation.ConnectionID, CanQuery: true}, nil).Once()
	mockStorage.
		On("InsertMessage", ctx, mock.MatchedBy(func(message *chattype.Message) bool {
			if message.Role != chattype.MessageRoleUser {
				return false
			}
			assert.Equal(t, chattype.MessageRoleUser, message.Role)
			assert.Equal(t, "how about this month?", message.Content)
			return message.ConversationID == conversation.ID && message.Status == chattype.MessageStatusCompleted
		})).
		Run(func(args mock.Arguments) {
			args.Get(1).(*chattype.Message).ID = 100
		}).
		Return(nil).
		Once()
	mockStorage.
		On("ListMessages", ctx, mock.MatchedBy(func(filter chatstorage.MessagesFilter) bool {
			return len(filter.ConversationID) == 1 && filter.ConversationID[0] == conversation.ID
		})).
		Return([]*chattype.Message{
			previousAssistant,
			previousUser,
		}, nil).
		Once()
	mockStorage.On("GetExecution", ctx, previousAssistant.ID).Return(&chattype.MessageExecution{MessageID: previousAssistant.ID, GeneratedSQL: previousSQL}, nil).Once()
	mockSchemas.
		On("RetrieveRelevantSchemaContext", ctx, mock.MatchedBy(func(params schematypes.RetrieveRelevantSchemaContextParams) bool {
			return params.ConnectionID == conversation.ConnectionID &&
				params.Limit == defaultSchemaChunkLimit &&
				strings.Contains(params.QueryText, "Previous user question: how many users signed up") &&
				strings.Contains(params.QueryText, "Previous SQL: "+previousSQL) &&
				strings.Contains(params.QueryText, "Current follow-up question: how about this month?")
		})).
		Return(schemaContext, nil).
		Once()
	mockStorage.
		On("InsertRetrieval", ctx, mock.MatchedBy(func(retrieval *chattype.MessageRetrieval) bool {
			return retrieval.MessageID == 100 && retrieval.SnapshotID == schemaContext.SnapshotID && len(retrieval.Chunks) == 1
		})).
		Return(nil).
		Once()
	mockModels.
		On("ResolveClient", ctx, llmtypes.ProviderOpenAI, "gpt-5.2").
		Return(&chatllm.ResolvedClient{
			ResolvedModel: &chatllm.ResolvedModel{
				ProviderConfig:   &llmtypes.ProviderConfig{ID: 300, Provider: llmtypes.ProviderOpenAI},
				ProviderModelID:  "gpt-5.2",
				QualifiedModelID: "openai:gpt-5.2",
			},
			Client: mockClient,
		}, nil).
		Once()
	mockClient.
		On("GenerateSQL", ctx, mock.MatchedBy(func(req llmtypes.GenerateSQLRequest) bool {
			return req.Model == "gpt-5.2" &&
				req.UserPrompt == "how about this month?" &&
				req.Conversation.ConversationID == conversation.ID &&
				req.Conversation.DatabaseKind == connectiontypes.DatabasePostgres &&
				len(req.Conversation.History) == 2
		})).
		Return(&llmtypes.GenerateSQLResponse{
			SQL:         "select count(*) from users",
			Explanation: "Counts users.",
			RawRequest:  json.RawMessage(`{"request":true}`),
			RawResponse: json.RawMessage(`{"response":true}`),
			Usage:       &llmtypes.Usage{InputTokens: ptr.Of(11), OutputTokens: ptr.Of(7)},
		}, nil).
		Once()
	mockSQLRunner.
		On("Run", ctx, *connection, "select count(*) from users", sqlrunner.RunOptions{
			Timeout:  defaultQueryTimeout,
			RowLimit: defaultResultRowLimit,
		}).
		Return(queryResult, nil).
		Once()
	mockStorage.
		On("InTransaction", ctx, mock.Anything).
		Return(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).
		Once()
	mockStorage.
		On("InsertMessage", ctx, mock.MatchedBy(func(message *chattype.Message) bool {
			if message.Role != chattype.MessageRoleAssistant {
				return false
			}
			return message.Role == chattype.MessageRoleAssistant &&
				message.Content == "Counts users." &&
				message.Provider != nil &&
				*message.Provider == llmtypes.ProviderOpenAI
		})).
		Run(func(args mock.Arguments) {
			args.Get(1).(*chattype.Message).ID = 200
		}).
		Return(nil).
		Once()
	mockStorage.
		On("InsertLLMCall", ctx, mock.MatchedBy(func(call *chattype.MessageLLMCall) bool {
			return call.MessageID == 200 &&
				call.ProviderConfigID == 300 &&
				call.Provider == llmtypes.ProviderOpenAI &&
				call.Model == "gpt-5.2" &&
				call.InputTokens != nil &&
				*call.InputTokens == 11 &&
				call.OutputTokens != nil &&
				*call.OutputTokens == 7
		})).
		Return(nil).
		Once()
	mockStorage.
		On("InsertExecution", ctx, mock.MatchedBy(func(execution *chattype.MessageExecution) bool {
			return execution.MessageID == 200 &&
				execution.ConnectionID == connection.ID &&
				execution.DatabaseKind == connection.Database &&
				execution.GeneratedSQL == "select count(*) from users" &&
				execution.Result.RowCount == 1
		})).
		Return(nil).
		Once()

	service := newTestService(
		mockStorage,
		mockConnections,
		mockSchemas,
		mockModels,
		mockSQLRunner,
	)

	turn, err := service.SendMessage(ctx, chattype.SendMessageParams{
		UserID:         userID,
		ConversationID: conversation.ID,
		Content:        " how about this month? ",
		Provider:       llmtypes.ProviderOpenAI,
		Model:          "gpt-5.2",
	})

	require.NoError(t, err)
	require.NotNil(t, turn)
	assert.Equal(t, int64(100), turn.UserMessage.ID)
	assert.Equal(t, int64(200), turn.AssistantMessage.ID)
	assert.Equal(t, int64(100), turn.Retrieval.MessageID)
	assert.Equal(t, int64(200), turn.Execution.MessageID)
}

func TestService_SendMessage_RejectsBeforePersistence(t *testing.T) {
	t.Parallel()

	userID := int32(7)
	conversation := &chattype.Conversation{ID: 10, UserID: userID, ConnectionID: 42}
	defaultParams := chattype.SendMessageParams{
		UserID:         userID,
		ConversationID: conversation.ID,
		Content:        "how many users?",
		Provider:       llmtypes.ProviderOpenAI,
		Model:          "gpt-5.2",
	}

	tests := []struct {
		name      string
		params    chattype.SendMessageParams
		setup     func(t *testing.T, ctx context.Context) (*Service, *storagetesting.Storage)
		expectErr error
	}{
		{
			name:   "conversation owned by another user",
			params: defaultParams,
			setup: func(t *testing.T, ctx context.Context) (*Service, *storagetesting.Storage) {
				t.Helper()

				mockStorage := storagetesting.NewStorage(t)
				mockStorage.
					On("GetConversation", ctx, conversation.ID).
					Return(&chattype.Conversation{ID: conversation.ID, UserID: 99, ConnectionID: conversation.ConnectionID}, nil).
					Once()

				return newTestService(mockStorage, nil, nil, nil, nil), mockStorage
			},
			expectErr: chaterrors.ErrConversationNotFound,
		},
		{
			name:   "query access disabled",
			params: defaultParams,
			setup: func(t *testing.T, ctx context.Context) (*Service, *storagetesting.Storage) {
				t.Helper()

				mockStorage := storagetesting.NewStorage(t)
				mockConnections := chattesting.NewConnectionService(t)
				mockStorage.On("GetConversation", ctx, conversation.ID).Return(conversation, nil).Once()
				mockConnections.
					On("GetConnection", ctx, conversation.ConnectionID).
					Return(&connectiontypes.Connection{ID: conversation.ConnectionID, Database: connectiontypes.DatabasePostgres}, nil).
					Once()
				mockConnections.On("GetAccess", ctx, userID, conversation.ConnectionID).Return(&connectiontypes.Access{CanQuery: false}, nil).Once()

				return newTestService(mockStorage, mockConnections, nil, nil, nil), mockStorage
			},
			expectErr: chaterrors.ErrConnectionAccessDenied,
		},
		{
			name:   "missing access row",
			params: defaultParams,
			setup: func(t *testing.T, ctx context.Context) (*Service, *storagetesting.Storage) {
				t.Helper()

				mockStorage := storagetesting.NewStorage(t)
				mockConnections := chattesting.NewConnectionService(t)
				mockStorage.On("GetConversation", ctx, conversation.ID).Return(conversation, nil).Once()
				mockConnections.
					On("GetConnection", ctx, conversation.ConnectionID).
					Return(&connectiontypes.Connection{ID: conversation.ConnectionID, Database: connectiontypes.DatabasePostgres}, nil).
					Once()
				mockConnections.On("GetAccess", ctx, userID, conversation.ConnectionID).Return(nil, connectionerrors.ErrAccessNotFound).Once()

				return newTestService(mockStorage, mockConnections, nil, nil, nil), mockStorage
			},
			expectErr: chaterrors.ErrConnectionAccessDenied,
		},
		{
			name:   "unsupported database",
			params: defaultParams,
			setup: func(t *testing.T, ctx context.Context) (*Service, *storagetesting.Storage) {
				t.Helper()

				mockStorage := storagetesting.NewStorage(t)
				mockConnections := chattesting.NewConnectionService(t)
				mockStorage.On("GetConversation", ctx, conversation.ID).Return(conversation, nil).Once()
				mockConnections.
					On("GetConnection", ctx, conversation.ConnectionID).
					Return(&connectiontypes.Connection{ID: conversation.ConnectionID, Database: connectiontypes.DatabaseCQL}, nil).
					Once()
				mockConnections.On("GetAccess", ctx, userID, conversation.ConnectionID).Return(&connectiontypes.Access{CanQuery: true}, nil).Once()

				return newTestService(mockStorage, mockConnections, nil, nil, nil), mockStorage
			},
			expectErr: chaterrors.ErrUnsupportedDatabaseKind,
		},
		{
			name: "unavailable model",
			params: chattype.SendMessageParams{
				UserID:         userID,
				ConversationID: conversation.ID,
				Content:        "how many users?",
				Provider:       llmtypes.ProviderOpenAI,
				Model:          "missing",
			},
			setup: func(t *testing.T, ctx context.Context) (*Service, *storagetesting.Storage) {
				t.Helper()

				mockStorage := storagetesting.NewStorage(t)
				mockConnections := chattesting.NewConnectionService(t)
				mockModels := llmtesting.NewClientResolver(t)
				mockStorage.On("GetConversation", ctx, conversation.ID).Return(conversation, nil).Once()
				mockConnections.
					On("GetConnection", ctx, conversation.ConnectionID).
					Return(&connectiontypes.Connection{ID: conversation.ConnectionID, Database: connectiontypes.DatabasePostgres}, nil).
					Once()
				mockConnections.On("GetAccess", ctx, userID, conversation.ConnectionID).Return(&connectiontypes.Access{CanQuery: true}, nil).Once()
				mockModels.
					On("ResolveClient", ctx, llmtypes.ProviderOpenAI, "missing").
					Return((*chatllm.ResolvedClient)(nil), errors.Join(chaterrors.ErrModelNotAvailable)).
					Once()

				return newTestService(mockStorage, mockConnections, nil, mockModels, nil), mockStorage
			},
			expectErr: chaterrors.ErrModelNotAvailable,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			service, mockStorage := test.setup(t, ctx)

			_, err := service.SendMessage(ctx, test.params)

			require.ErrorIs(t, err, test.expectErr)
			mockStorage.AssertNotCalled(t, "InTransaction", mock.Anything, mock.Anything)
			mockStorage.AssertNotCalled(t, "InsertMessage", mock.Anything, mock.Anything)
		})
	}
}

func TestService_SendMessage_DoesNotPersistMessagesWhenLLMGenerationFails(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	userID := int32(7)
	conversation := &chattype.Conversation{ID: 10, UserID: userID, ConnectionID: 42}
	connection := &connectiontypes.Connection{ID: 42, Database: connectiontypes.DatabasePostgres, DSN: "postgres://warehouse"}
	schemaContext := &schematypes.RetrievedSchemaContext{
		ConnectionID:   conversation.ConnectionID,
		SnapshotID:     55,
		EmbeddingModel: "nomic-embed-text",
		QueryText:      "how many users?",
		RetrievedAt:    time.Now().UTC(),
	}
	mockStorage := storagetesting.NewStorage(t)
	mockConnections := chattesting.NewConnectionService(t)
	mockSchemas := chattesting.NewSchemaRetriever(t)
	mockModels := llmtesting.NewClientResolver(t)
	mockClient := llmtesting.NewClient(t)
	mockStorage.On("GetConversation", ctx, conversation.ID).Return(conversation, nil).Once()
	mockConnections.On("GetConnection", ctx, conversation.ConnectionID).Return(connection, nil).Once()
	mockConnections.On("GetAccess", ctx, userID, conversation.ConnectionID).Return(&connectiontypes.Access{CanQuery: true}, nil).Once()
	mockModels.
		On("ResolveClient", ctx, llmtypes.ProviderOpenAI, "gpt-5.2").
		Return(&chatllm.ResolvedClient{
			ResolvedModel: &chatllm.ResolvedModel{
				ProviderConfig:   &llmtypes.ProviderConfig{ID: 300, Provider: llmtypes.ProviderOpenAI},
				ProviderModelID:  "gpt-5.2",
				QualifiedModelID: "openai:gpt-5.2",
			},
			Client: mockClient,
		}, nil).
		Once()
	mockStorage.
		On("ListMessages", ctx, mock.MatchedBy(func(filter chatstorage.MessagesFilter) bool {
			return len(filter.ConversationID) == 1 && filter.ConversationID[0] == conversation.ID
		})).
		Return(nil, nil).
		Once()
	mockSchemas.
		On("RetrieveRelevantSchemaContext", ctx, mock.MatchedBy(func(params schematypes.RetrieveRelevantSchemaContextParams) bool {
			return params.ConnectionID == conversation.ConnectionID &&
				params.Limit == defaultSchemaChunkLimit &&
				strings.Contains(params.QueryText, "how many users?")
		})).
		Return(schemaContext, nil).
		Once()
	mockClient.
		On("GenerateSQL", ctx, mock.Anything).
		Return((*llmtypes.GenerateSQLResponse)(nil), errors.New("provider unavailable")).
		Once()

	service := newTestService(
		mockStorage,
		mockConnections,
		mockSchemas,
		mockModels,
		nil,
	)

	_, err := service.SendMessage(ctx, chattype.SendMessageParams{
		UserID:         userID,
		ConversationID: conversation.ID,
		Content:        "how many users?",
		Provider:       llmtypes.ProviderOpenAI,
		Model:          "gpt-5.2",
	})

	require.ErrorContains(t, err, "failed to generate sql")
	mockStorage.AssertNotCalled(t, "InTransaction", mock.Anything, mock.Anything)
	mockStorage.AssertNotCalled(t, "InsertMessage", mock.Anything, mock.Anything)
}

func TestService_SendMessage_DoesNotPersistMessagesWhenSQLExecutionFails(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	userID := int32(7)
	conversation := &chattype.Conversation{ID: 10, UserID: userID, ConnectionID: 42}
	connection := &connectiontypes.Connection{ID: 42, Database: connectiontypes.DatabasePostgres, DSN: "postgres://warehouse"}
	schemaContext := &schematypes.RetrievedSchemaContext{
		ConnectionID:   conversation.ConnectionID,
		SnapshotID:     55,
		EmbeddingModel: "nomic-embed-text",
		QueryText:      "how many users?",
		Chunks: []schematypes.RetrievedChunk{
			{ChunkID: 1, ObjectType: "table", ObjectName: "users", Content: "table users(id int)", Similarity: 0.91},
		},
		RetrievedAt: time.Now().UTC(),
	}

	mockStorage := storagetesting.NewStorage(t)
	mockConnections := chattesting.NewConnectionService(t)
	mockSchemas := chattesting.NewSchemaRetriever(t)
	mockModels := llmtesting.NewClientResolver(t)
	mockClient := llmtesting.NewClient(t)
	mockSQLRunner := sqlrunnertesting.NewSQLRunner(t)

	mockStorage.On("GetConversation", ctx, conversation.ID).Return(conversation, nil).Once()
	mockConnections.On("GetConnection", ctx, conversation.ConnectionID).Return(connection, nil).Once()
	mockConnections.On("GetAccess", ctx, userID, conversation.ConnectionID).Return(&connectiontypes.Access{CanQuery: true}, nil).Once()
	mockModels.
		On("ResolveClient", ctx, llmtypes.ProviderOpenAI, "gpt-5.2").
		Return(&chatllm.ResolvedClient{
			ResolvedModel: &chatllm.ResolvedModel{
				ProviderConfig:   &llmtypes.ProviderConfig{ID: 300, Provider: llmtypes.ProviderOpenAI},
				ProviderModelID:  "gpt-5.2",
				QualifiedModelID: "openai:gpt-5.2",
			},
			Client: mockClient,
		}, nil).
		Once()
	mockStorage.
		On("ListMessages", ctx, mock.MatchedBy(func(filter chatstorage.MessagesFilter) bool {
			return len(filter.ConversationID) == 1 && filter.ConversationID[0] == conversation.ID
		})).
		Return(nil, nil).
		Once()
	mockSchemas.
		On("RetrieveRelevantSchemaContext", ctx, mock.MatchedBy(func(params schematypes.RetrieveRelevantSchemaContextParams) bool {
			return params.ConnectionID == conversation.ConnectionID && params.Limit == defaultSchemaChunkLimit
		})).
		Return(schemaContext, nil).
		Once()
	mockClient.
		On("GenerateSQL", ctx, mock.Anything).
		Return(&llmtypes.GenerateSQLResponse{
			SQL:         "select count(*) from users",
			Explanation: "Counts users.",
		}, nil).
		Once()
	executionErr := errors.New("database unavailable")
	mockSQLRunner.
		On("Run", ctx, *connection, "select count(*) from users", sqlrunner.RunOptions{
			Timeout:  defaultQueryTimeout,
			RowLimit: defaultResultRowLimit,
		}).
		Return((*chattype.QueryResult)(nil), executionErr).
		Once()

	service := newTestService(
		mockStorage,
		mockConnections,
		mockSchemas,
		mockModels,
		mockSQLRunner,
	)

	_, err := service.SendMessage(ctx, chattype.SendMessageParams{
		UserID:         userID,
		ConversationID: conversation.ID,
		Content:        "how many users?",
		Provider:       llmtypes.ProviderOpenAI,
		Model:          "gpt-5.2",
	})

	require.ErrorIs(t, err, executionErr)
	mockStorage.AssertNotCalled(t, "InTransaction", mock.Anything, mock.Anything)
	mockStorage.AssertNotCalled(t, "InsertMessage", mock.Anything, mock.Anything)
}
