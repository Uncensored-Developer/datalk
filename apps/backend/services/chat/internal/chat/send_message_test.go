package chat

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
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
	usertypes "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/gotidy/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type staticUserService struct {
	user *usertypes.User
	err  error
}

func (s staticUserService) GetUser(context.Context, int32) (*usertypes.User, error) {
	return s.user, s.err
}

func correctionEligibleSQLError(message string) error {
	return &sqlrunner.Error{
		Kind:               sqlrunner.ErrorKindQueryExecution,
		CorrectionEligible: true,
		Err:                errors.New(message),
	}
}

func TestService_GetQueryableConnection_AllowsAdminsWithoutConnectionAccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	userID := int32(7)
	connectionID := int32(42)
	connection := &connectiontypes.Connection{ID: connectionID, Database: connectiontypes.DatabasePostgres}
	mockConnections := chattesting.NewConnectionService(t)
	mockConnections.On("GetConnection", ctx, connectionID).Return(connection, nil).Once()

	service := NewService(
		config.Config{},
		nil,
		nil,
		mockConnections,
		staticUserService{user: &usertypes.User{ID: userID, Role: usertypes.RoleAdmin}},
		nil,
		nil,
		nil,
	)

	got, err := service.getQueryableConnection(ctx, userID, connectionID)

	require.NoError(t, err)
	assert.Equal(t, connection, got)
	mockConnections.AssertNotCalled(t, "GetAccess", ctx, userID, connectionID)
}

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
	mockClient.AssertNotCalled(t, "GenerateAnswer", mock.Anything, mock.Anything)
}

func TestService_SendMessage_GeneratesNaturalResponseWhenRequested(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	userID := int32(7)
	conversation := &chattype.Conversation{ID: 10, UserID: userID, ConnectionID: 42}
	connection := &connectiontypes.Connection{ID: 42, Database: connectiontypes.DatabasePostgres, DSN: "postgres://warehouse"}
	schemaContext := &schematypes.RetrievedSchemaContext{
		ConnectionID: conversation.ConnectionID,
		SnapshotID:   55,
		RetrievedAt:  time.Now().UTC(),
	}
	queryResult := &chattype.QueryResult{
		Columns:  []chattype.ResultColumn{{Name: "email", DataType: "TEXT"}, {Name: "transactions", DataType: "INT8"}},
		Rows:     []map[string]any{{"email": "admin@datalk.app", "transactions": int64(200)}},
		RowCount: 1,
		Kind:     chattype.QueryResultKindRecord,
	}

	mockStorage := storagetesting.NewStorage(t)
	mockConnections := chattesting.NewConnectionService(t)
	mockSchemas := chattesting.NewSchemaRetriever(t)
	mockModels := llmtesting.NewClientResolver(t)
	mockSQLRunner := sqlrunnertesting.NewSQLRunner(t)
	mockClient := llmtesting.NewClient(t)

	mockStorage.On("GetConversation", ctx, conversation.ID).Return(conversation, nil).Once()
	mockConnections.On("GetConnection", ctx, conversation.ConnectionID).Return(connection, nil).Once()
	mockConnections.On("GetAccess", ctx, userID, conversation.ConnectionID).Return(&connectiontypes.Access{CanQuery: true}, nil).Once()
	mockModels.On("ResolveClient", ctx, llmtypes.ProviderOpenAI, "gpt-5.2").Return(&chatllm.ResolvedClient{
		ResolvedModel: &chatllm.ResolvedModel{
			ProviderConfig:   &llmtypes.ProviderConfig{ID: 300, Provider: llmtypes.ProviderOpenAI},
			ProviderModelID:  "gpt-5.2",
			QualifiedModelID: "openai:gpt-5.2",
		},
		Client: mockClient,
	}, nil).Once()
	mockStorage.On("ListMessages", ctx, mock.Anything).Return(nil, nil).Once()
	mockSchemas.On("RetrieveRelevantSchemaContext", ctx, mock.Anything).Return(schemaContext, nil).Once()
	mockClient.On("GenerateSQL", ctx, mock.Anything).Return(&llmtypes.GenerateSQLResponse{
		SQL:         "select email, count(*) as transactions from transactions group by email order by transactions desc limit 1",
		Explanation: "Finds the top transacting user.",
		RawRequest:  json.RawMessage(`{"sql_request":true}`),
		RawResponse: json.RawMessage(`{"sql_response":true}`),
		Usage:       &llmtypes.Usage{InputTokens: ptr.Of(10), OutputTokens: ptr.Of(5)},
	}, nil).Once()
	mockSQLRunner.On("Run", ctx, *connection, "select email, count(*) as transactions from transactions group by email order by transactions desc limit 1", sqlrunner.RunOptions{
		Timeout:  defaultQueryTimeout,
		RowLimit: defaultResultRowLimit,
	}).Return(queryResult, nil).Once()
	mockClient.On("GenerateAnswer", ctx, mock.MatchedBy(func(req llmtypes.GenerateAnswerRequest) bool {
		return req.Model == "gpt-5.2" &&
			req.UserPrompt == "Who transacted the most?" &&
			req.GeneratedSQL == "select email, count(*) as transactions from transactions group by email order by transactions desc limit 1" &&
			req.Result.RowCount == 1 &&
			req.Result.Kind == string(chattype.QueryResultKindRecord)
	})).Return(&llmtypes.GenerateAnswerResponse{
		Answer:      "admin@datalk.app has the most transactions with 200 transactions.",
		RawRequest:  json.RawMessage(`{"answer_request":true}`),
		RawResponse: json.RawMessage(`{"answer_response":true}`),
		Usage:       &llmtypes.Usage{InputTokens: ptr.Of(12), OutputTokens: ptr.Of(8)},
	}, nil).Once()

	mockStorage.On("InTransaction", ctx, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	mockStorage.On("InsertMessage", ctx, mock.MatchedBy(func(message *chattype.Message) bool {
		return message.Role == chattype.MessageRoleUser
	})).Run(func(args mock.Arguments) {
		args.Get(1).(*chattype.Message).ID = 100
	}).Return(nil).Once()
	mockStorage.On("InsertRetrieval", ctx, mock.Anything).Return(nil).Once()
	mockStorage.On("InsertMessage", ctx, mock.MatchedBy(func(message *chattype.Message) bool {
		if message.Role != chattype.MessageRoleAssistant {
			return false
		}
		return message.Status == chattype.MessageStatusCompleted &&
			message.Content == "Finds the top transacting user." &&
			message.NaturalResponse != nil &&
			*message.NaturalResponse == "admin@datalk.app has the most transactions with 200 transactions."
	})).Run(func(args mock.Arguments) {
		args.Get(1).(*chattype.Message).ID = 200
	}).Return(nil).Once()
	mockStorage.On("InsertLLMCall", ctx, mock.MatchedBy(func(call *chattype.MessageLLMCall) bool {
		return call.MessageID == 200 && call.InputTokens != nil && *call.InputTokens == 10
	})).Return(nil).Once()
	mockStorage.On("InsertLLMCall", ctx, mock.MatchedBy(func(call *chattype.MessageLLMCall) bool {
		return call.MessageID == 200 && call.InputTokens != nil && *call.InputTokens == 12
	})).Return(nil).Once()
	mockStorage.On("InsertExecution", ctx, mock.MatchedBy(func(execution *chattype.MessageExecution) bool {
		return execution.MessageID == 200 && execution.Result.RowCount == 1
	})).Return(nil).Once()

	service := newTestService(mockStorage, mockConnections, mockSchemas, mockModels, mockSQLRunner)

	turn, err := service.SendMessage(ctx, chattype.SendMessageParams{
		UserID:                 userID,
		ConversationID:         conversation.ID,
		Content:                "Who transacted the most?",
		Provider:               llmtypes.ProviderOpenAI,
		Model:                  "gpt-5.2",
		RequireNaturalResponse: true,
	})

	require.NoError(t, err)
	require.NotNil(t, turn)
	require.NotNil(t, turn.AssistantMessage.NaturalResponse)
	assert.Equal(t, "Finds the top transacting user.", turn.AssistantMessage.Content)
	assert.Equal(t, "admin@datalk.app has the most transactions with 200 transactions.", *turn.AssistantMessage.NaturalResponse)
}

func TestService_SendMessage_PersistsSQLExecutionWhenNaturalResponseFails(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	userID := int32(7)
	conversation := &chattype.Conversation{ID: 10, UserID: userID, ConnectionID: 42}
	connection := &connectiontypes.Connection{ID: 42, Database: connectiontypes.DatabasePostgres, DSN: "postgres://warehouse"}
	schemaContext := &schematypes.RetrievedSchemaContext{ConnectionID: conversation.ConnectionID, SnapshotID: 55, RetrievedAt: time.Now().UTC()}
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
	mockConnections.On("GetAccess", ctx, userID, conversation.ConnectionID).Return(&connectiontypes.Access{CanQuery: true}, nil).Once()
	mockModels.On("ResolveClient", ctx, llmtypes.ProviderOpenAI, "gpt-5.2").Return(&chatllm.ResolvedClient{
		ResolvedModel: &chatllm.ResolvedModel{
			ProviderConfig:   &llmtypes.ProviderConfig{ID: 300, Provider: llmtypes.ProviderOpenAI},
			ProviderModelID:  "gpt-5.2",
			QualifiedModelID: "openai:gpt-5.2",
		},
		Client: mockClient,
	}, nil).Once()
	mockStorage.On("ListMessages", ctx, mock.Anything).Return(nil, nil).Once()
	mockSchemas.On("RetrieveRelevantSchemaContext", ctx, mock.Anything).Return(schemaContext, nil).Once()
	mockClient.On("GenerateSQL", ctx, mock.Anything).Return(&llmtypes.GenerateSQLResponse{
		SQL:         "select count(*) from users",
		Explanation: "Counts users.",
		RawRequest:  json.RawMessage(`{"sql_request":true}`),
		RawResponse: json.RawMessage(`{"sql_response":true}`),
	}, nil).Once()
	mockSQLRunner.On("Run", ctx, *connection, "select count(*) from users", sqlrunner.RunOptions{
		Timeout:  defaultQueryTimeout,
		RowLimit: defaultResultRowLimit,
	}).Return(queryResult, nil).Once()
	answerErr := errors.New("answer model unavailable")
	mockClient.On("GenerateAnswer", ctx, mock.Anything).Return((*llmtypes.GenerateAnswerResponse)(nil), answerErr).Once()

	mockStorage.On("InTransaction", ctx, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	mockStorage.On("InsertMessage", ctx, mock.MatchedBy(func(message *chattype.Message) bool {
		return message.Role == chattype.MessageRoleUser
	})).Run(func(args mock.Arguments) {
		args.Get(1).(*chattype.Message).ID = 100
	}).Return(nil).Once()
	mockStorage.On("InsertRetrieval", ctx, mock.Anything).Return(nil).Once()
	mockStorage.On("InsertMessage", ctx, mock.MatchedBy(func(message *chattype.Message) bool {
		if message.Role != chattype.MessageRoleAssistant {
			return false
		}
		return message.Status == chattype.MessageStatusCompleted &&
			message.Content == "Counts users." &&
			message.NaturalResponse == nil &&
			message.ErrorMessage != nil &&
			strings.Contains(*message.ErrorMessage, "answer model unavailable")
	})).Run(func(args mock.Arguments) {
		args.Get(1).(*chattype.Message).ID = 200
	}).Return(nil).Once()
	mockStorage.On("InsertLLMCall", ctx, mock.MatchedBy(func(call *chattype.MessageLLMCall) bool {
		return call.MessageID == 200
	})).Return(nil).Once()
	mockStorage.On("InsertExecution", ctx, mock.MatchedBy(func(execution *chattype.MessageExecution) bool {
		return execution.MessageID == 200 && execution.GeneratedSQL == "select count(*) from users"
	})).Return(nil).Once()

	service := newTestService(mockStorage, mockConnections, mockSchemas, mockModels, mockSQLRunner)

	turn, err := service.SendMessage(ctx, chattype.SendMessageParams{
		UserID:                 userID,
		ConversationID:         conversation.ID,
		Content:                "how many users?",
		Provider:               llmtypes.ProviderOpenAI,
		Model:                  "gpt-5.2",
		RequireNaturalResponse: true,
	})

	require.NoError(t, err)
	require.NotNil(t, turn)
	assert.Equal(t, chattype.MessageStatusCompleted, turn.AssistantMessage.Status)
	assert.Nil(t, turn.AssistantMessage.NaturalResponse)
	require.NotNil(t, turn.AssistantMessage.ErrorMessage)
	assert.Contains(t, *turn.AssistantMessage.ErrorMessage, "answer model unavailable")
	require.NotNil(t, turn.Execution)
}

func TestService_SendMessage_RetriesCorrectionEligibleSQLError(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	userID := int32(7)
	conversation := &chattype.Conversation{ID: 10, UserID: userID, ConnectionID: 42}
	connection := &connectiontypes.Connection{ID: 42, Database: connectiontypes.DatabasePostgres, DSN: "postgres://warehouse"}
	schemaContext := &schematypes.RetrievedSchemaContext{
		ConnectionID: conversation.ConnectionID,
		SnapshotID:   55,
		Chunks:       []schematypes.RetrievedChunk{{ChunkID: 1, ObjectType: "table", ObjectName: "users", Content: "table users(id int, email text)"}},
		RetrievedAt:  time.Now().UTC(),
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
	mockConnections.On("GetAccess", ctx, userID, conversation.ConnectionID).Return(&connectiontypes.Access{CanQuery: true}, nil).Once()
	mockModels.On("ResolveClient", ctx, llmtypes.ProviderOpenAI, "gpt-5.2").Return(&chatllm.ResolvedClient{
		ResolvedModel: &chatllm.ResolvedModel{
			ProviderConfig:   &llmtypes.ProviderConfig{ID: 300, Provider: llmtypes.ProviderOpenAI},
			ProviderModelID:  "gpt-5.2",
			QualifiedModelID: "openai:gpt-5.2",
		},
		Client: mockClient,
	}, nil).Once()
	mockStorage.On("ListMessages", ctx, mock.Anything).Return(nil, nil).Once()
	mockSchemas.On("RetrieveRelevantSchemaContext", ctx, mock.Anything).Return(schemaContext, nil).Once()

	sqlErr := correctionEligibleSQLError(`failed to execute query: pq: column "user_email" does not exist`)
	mockClient.
		On("GenerateSQL", ctx, mock.MatchedBy(func(req llmtypes.GenerateSQLRequest) bool {
			return req.Correction == nil
		})).
		Return(&llmtypes.GenerateSQLResponse{
			SQL:         "select user_email from users",
			Explanation: "Looks up the user email.",
			RawRequest:  json.RawMessage(`{"attempt":1}`),
			RawResponse: json.RawMessage(`{"sql":1}`),
		}, nil).
		Once()
	mockSQLRunner.
		On("Run", ctx, *connection, "select user_email from users", sqlrunner.RunOptions{Timeout: defaultQueryTimeout, RowLimit: defaultResultRowLimit}).
		Return((*chattype.QueryResult)(nil), sqlErr).
		Once()
	mockClient.
		On("GenerateSQL", ctx, mock.MatchedBy(func(req llmtypes.GenerateSQLRequest) bool {
			return req.Correction != nil &&
				req.Correction.AttemptNumber == 1 &&
				len(req.Correction.Attempts) == 1 &&
				req.Correction.Attempts[0].SQL == "select user_email from users" &&
				strings.Contains(req.Correction.Attempts[0].Error, "user_email")
		})).
		Return(&llmtypes.GenerateSQLResponse{
			SQL:         "select count(*) from users",
			Explanation: "Counts users.",
			RawRequest:  json.RawMessage(`{"attempt":2}`),
			RawResponse: json.RawMessage(`{"sql":2}`),
		}, nil).
		Once()
	mockSQLRunner.
		On("Run", ctx, *connection, "select count(*) from users", sqlrunner.RunOptions{Timeout: defaultQueryTimeout, RowLimit: defaultResultRowLimit}).
		Return(queryResult, nil).
		Once()

	mockStorage.On("InTransaction", ctx, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	mockStorage.On("InsertMessage", ctx, mock.MatchedBy(func(message *chattype.Message) bool {
		return message.Role == chattype.MessageRoleUser
	})).Run(func(args mock.Arguments) {
		args.Get(1).(*chattype.Message).ID = 100
	}).Return(nil).Once()
	mockStorage.On("InsertRetrieval", ctx, mock.MatchedBy(func(retrieval *chattype.MessageRetrieval) bool {
		return retrieval.MessageID == 100
	})).Return(nil).Once()
	mockStorage.On("InsertMessage", ctx, mock.MatchedBy(func(message *chattype.Message) bool {
		if message.Role != chattype.MessageRoleAssistant {
			return false
		}
		return message.Status == chattype.MessageStatusCompleted && message.Content == "Counts users."
	})).Run(func(args mock.Arguments) {
		args.Get(1).(*chattype.Message).ID = 200
	}).Return(nil).Once()
	mockStorage.On("InsertLLMCall", ctx, mock.MatchedBy(func(call *chattype.MessageLLMCall) bool {
		return call.MessageID == 200
	})).Return(nil).Twice()
	mockStorage.On("InsertExecution", ctx, mock.MatchedBy(func(execution *chattype.MessageExecution) bool {
		return execution.MessageID == 200 && execution.GeneratedSQL == "select count(*) from users"
	})).Return(nil).Once()

	service := newTestService(mockStorage, mockConnections, mockSchemas, mockModels, mockSQLRunner)

	turn, err := service.SendMessage(ctx, chattype.SendMessageParams{
		UserID:         userID,
		ConversationID: conversation.ID,
		Content:        "how many users?",
		Provider:       llmtypes.ProviderOpenAI,
		Model:          "gpt-5.2",
	})

	require.NoError(t, err)
	require.NotNil(t, turn)
	require.NotNil(t, turn.Execution)
	assert.Equal(t, chattype.MessageStatusCompleted, turn.AssistantMessage.Status)
	assert.Equal(t, "select count(*) from users", turn.Execution.GeneratedSQL)
}

func TestService_SendMessage_PersistsFailedTurnAfterCorrectionExhaustion(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	userID := int32(7)
	conversation := &chattype.Conversation{ID: 10, UserID: userID, ConnectionID: 42}
	connection := &connectiontypes.Connection{ID: 42, Database: connectiontypes.DatabasePostgres, DSN: "postgres://warehouse"}
	schemaContext := &schematypes.RetrievedSchemaContext{
		ConnectionID: conversation.ConnectionID,
		SnapshotID:   55,
		Chunks:       []schematypes.RetrievedChunk{{ChunkID: 1, ObjectType: "table", ObjectName: "users", Content: "table users(id int, email text)"}},
		RetrievedAt:  time.Now().UTC(),
	}

	mockStorage := storagetesting.NewStorage(t)
	mockConnections := chattesting.NewConnectionService(t)
	mockSchemas := chattesting.NewSchemaRetriever(t)
	mockModels := llmtesting.NewClientResolver(t)
	mockSQLRunner := sqlrunnertesting.NewSQLRunner(t)
	mockClient := llmtesting.NewClient(t)

	mockStorage.On("GetConversation", ctx, conversation.ID).Return(conversation, nil).Once()
	mockConnections.On("GetConnection", ctx, conversation.ConnectionID).Return(connection, nil).Once()
	mockConnections.On("GetAccess", ctx, userID, conversation.ConnectionID).Return(&connectiontypes.Access{CanQuery: true}, nil).Once()
	mockModels.On("ResolveClient", ctx, llmtypes.ProviderOpenAI, "gpt-5.2").Return(&chatllm.ResolvedClient{
		ResolvedModel: &chatllm.ResolvedModel{
			ProviderConfig:   &llmtypes.ProviderConfig{ID: 300, Provider: llmtypes.ProviderOpenAI},
			ProviderModelID:  "gpt-5.2",
			QualifiedModelID: "openai:gpt-5.2",
		},
		Client: mockClient,
	}, nil).Once()
	mockStorage.On("ListMessages", ctx, mock.Anything).Return(nil, nil).Once()
	mockSchemas.On("RetrieveRelevantSchemaContext", ctx, mock.Anything).Return(schemaContext, nil).Once()

	for attempt := 1; attempt <= maxSQLAttempts; attempt++ {
		attempt := attempt
		sqlText := "select missing_" + string(rune('0'+attempt)) + " from users"
		mockClient.
			On("GenerateSQL", ctx, mock.MatchedBy(func(req llmtypes.GenerateSQLRequest) bool {
				if attempt == 1 {
					return req.Correction == nil
				}
				return req.Correction != nil &&
					req.Correction.AttemptNumber == attempt-1 &&
					len(req.Correction.Attempts) == attempt-1
			})).
			Return(&llmtypes.GenerateSQLResponse{
				SQL:         sqlText,
				Explanation: "Attempts to query users.",
				RawRequest:  json.RawMessage(`{"attempt":true}`),
				RawResponse: json.RawMessage(`{"sql":true}`),
			}, nil).
			Once()
		mockSQLRunner.
			On("Run", ctx, *connection, sqlText, sqlrunner.RunOptions{Timeout: defaultQueryTimeout, RowLimit: defaultResultRowLimit}).
			Return((*chattype.QueryResult)(nil), correctionEligibleSQLError("failed to execute query: pq: column does not exist")).
			Once()
	}

	mockStorage.On("InTransaction", ctx, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	mockStorage.On("InsertMessage", ctx, mock.MatchedBy(func(message *chattype.Message) bool {
		return message.Role == chattype.MessageRoleUser
	})).Run(func(args mock.Arguments) {
		args.Get(1).(*chattype.Message).ID = 100
	}).Return(nil).Once()
	mockStorage.On("InsertRetrieval", ctx, mock.MatchedBy(func(retrieval *chattype.MessageRetrieval) bool {
		return retrieval.MessageID == 100
	})).Return(nil).Once()
	mockStorage.On("InsertMessage", ctx, mock.MatchedBy(func(message *chattype.Message) bool {
		if message.Role != chattype.MessageRoleAssistant {
			return false
		}
		return message.Status == chattype.MessageStatusFailed &&
			message.ErrorMessage != nil &&
			strings.Contains(*message.ErrorMessage, "column does not exist")
	})).Run(func(args mock.Arguments) {
		args.Get(1).(*chattype.Message).ID = 200
	}).Return(nil).Once()
	mockStorage.On("InsertLLMCall", ctx, mock.MatchedBy(func(call *chattype.MessageLLMCall) bool {
		return call.MessageID == 200
	})).Return(nil).Times(maxSQLAttempts)

	service := newTestService(mockStorage, mockConnections, mockSchemas, mockModels, mockSQLRunner)

	turn, err := service.SendMessage(ctx, chattype.SendMessageParams{
		UserID:         userID,
		ConversationID: conversation.ID,
		Content:        "how many users?",
		Provider:       llmtypes.ProviderOpenAI,
		Model:          "gpt-5.2",
	})

	require.NoError(t, err)
	require.NotNil(t, turn)
	assert.Equal(t, chattype.MessageStatusFailed, turn.AssistantMessage.Status)
	assert.NotNil(t, turn.AssistantMessage.ErrorMessage)
	assert.Nil(t, turn.Execution)
	mockStorage.AssertNotCalled(t, "InsertExecution", mock.Anything, mock.Anything)
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
