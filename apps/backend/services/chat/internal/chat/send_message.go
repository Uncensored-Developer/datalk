package chat

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/sqlrunner"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	connectionerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/errors"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/mdobak/go-xerrors"
)

func (s *Service) SendMessage(ctx context.Context, params chattype.SendMessageParams) (*chattype.AssistantTurn, error) {
	startedAt := time.Now()
	userContent := strings.TrimSpace(params.Content)
	if userContent == "" {
		return nil, xerrors.New("message content is required")
	}

	conversation, connection, err := s.getOwnedConversation(ctx, params.UserID, params.ConversationID)
	if err != nil {
		s.logSendMessageFailure("failed to validate conversation access", err, params, nil)
		return nil, err
	}

	if !isSupportedChatDatabase(connection.Database) {
		err := xerrors.Newf("%s: %w", connection.Database, chaterrors.ErrUnsupportedDatabaseKind)
		s.logSendMessageFailure("unsupported chat database", err, params, connection)
		return nil, err
	}

	resolved, err := s.clientResolver.ResolveClient(ctx, params.Provider, params.Model)
	if err != nil {
		s.logSendMessageFailure("failed to resolve llm client", err, params, connection)
		return nil, err
	}
	if resolved == nil || resolved.ProviderConfig == nil || resolved.Client == nil {
		err := xerrors.Newf("model resolver returned incomplete client: %w", chaterrors.ErrModelNotAvailable)
		s.logSendMessageFailure("llm resolver returned incomplete client", err, params, connection)
		return nil, err
	}

	history, err := s.loadRecentHistory(ctx, conversation.ID, 0)
	if err != nil {
		s.logSendMessageFailure("failed to load recent chat history", err, params, connection)
		return nil, err
	}

	lastAssistantSQL, err := s.latestAssistantSQL(ctx, history)
	if err != nil {
		s.logSendMessageFailure("failed to load previous assistant sql", err, params, connection)
		return nil, err
	}

	retrievalQuery := buildRetrievalQuery(userContent, toConversationMessages(history), lastAssistantSQL)
	schemaContext, err := s.schemaRetriever.RetrieveRelevantSchemaContext(ctx, schematypes.RetrieveRelevantSchemaContextParams{
		ConnectionID: conversation.ConnectionID,
		QueryText:    retrievalQuery,
		Limit:        defaultSchemaChunkLimit,
	})
	if err != nil {
		s.logSendMessageFailure("failed to retrieve schema context", err, params, connection)
		return nil, err
	}
	if schemaContext == nil {
		err := xerrors.Newf("schema retrieval returned no context: %w", chaterrors.ErrEmbeddedSnapshotNotReady)
		s.logSendMessageFailure("schema retrieval returned no context", err, params, connection)
		return nil, err
	}

	retrieval := &chattype.MessageRetrieval{
		SnapshotID:  schemaContext.SnapshotID,
		QueryText:   retrievalQuery,
		Chunks:      schemaContext.Chunks,
		RetrievedAt: schemaContext.RetrievedAt,
	}

	generateReq := buildGenerateSQLRequest(conversation, connection.Database, userContent, history, schemaContext, resolved.ProviderModelID)
	if err := enforceGenerateSQLRequestLimits(&generateReq); err != nil {
		s.logSendMessageFailure("generated sql request exceeded limits", err, params, connection,
			slog.Int("schema_chunks", len(generateReq.Schema.Chunks)),
			slog.Int("max_prompt_bytes", generateReq.Options.MaxPromptBytes),
		)
		return nil, err
	}

	llmStarted := time.Now()
	generateResp, err := resolved.Client.GenerateSQL(ctx, generateReq)
	llmLatencyMS := elapsedMilliseconds(llmStarted)
	if err != nil {
		s.logSendMessageFailure("failed to generate sql", err, params, connection, slog.Int("llm_latency_ms", int(llmLatencyMS)))
		return nil, xerrors.Newf("failed to generate sql: %w", err)
	}
	if generateResp == nil {
		err := xerrors.New("provider returned empty sql response")
		s.logSendMessageFailure("provider returned empty sql response", err, params, connection, slog.Int("llm_latency_ms", int(llmLatencyMS)))
		return nil, err
	}

	if err := sqlrunner.NewValidator().Validate(connection.Database, generateResp.SQL); err != nil {
		s.logSendMessageFailure("failed to validate generated sql", err, params, connection, slog.Int("llm_latency_ms", int(llmLatencyMS)))
		return nil, xerrors.Newf("failed to validate sql: %w", err)
	}

	execStarted := time.Now()
	result, err := s.sqlRunner.Run(ctx, *connection, generateResp.SQL, sqlrunner.RunOptions{
		Timeout:  defaultQueryTimeout,
		RowLimit: defaultResultRowLimit,
	})
	executionLatencyMS := elapsedMilliseconds(execStarted)
	if err != nil {
		s.logSendMessageFailure("failed to execute generated sql", err, params, connection, slog.Int("execution_latency_ms", int(executionLatencyMS)))
		return nil, err
	}

	userMessage := &chattype.Message{
		ConversationID: conversation.ID,
		Role:           chattype.MessageRoleUser,
		Content:        userContent,
		Status:         chattype.MessageStatusCompleted,
	}
	assistantMessage := &chattype.Message{
		ConversationID: conversation.ID,
		Role:           chattype.MessageRoleAssistant,
		Content:        strings.TrimSpace(generateResp.Explanation),
		Provider:       &params.Provider,
		Model:          &params.Model,
		Status:         chattype.MessageStatusCompleted,
	}
	if assistantMessage.Content == "" {
		assistantMessage.Content = "Query executed successfully."
	}

	execution := &chattype.MessageExecution{
		ConnectionID:       connection.ID,
		DatabaseKind:       connection.Database,
		GeneratedSQL:       generateResp.SQL,
		NormalizedSQL:      generateResp.SQL,
		Result:             *result,
		ExecutionLatencyMS: executionLatencyMS,
		ExecutedAt:         time.Now().UTC(),
	}
	var llmCall *chattype.MessageLLMCall
	if err := s.storage.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.storage.InsertMessage(txCtx, userMessage); err != nil {
			return xerrors.Newf("failed to persist user message: %w", err)
		}

		retrieval.MessageID = userMessage.ID
		if err := s.storage.InsertRetrieval(txCtx, retrieval); err != nil {
			return xerrors.Newf("failed to persist retrieval: %w", err)
		}

		if err := s.storage.InsertMessage(txCtx, assistantMessage); err != nil {
			return xerrors.Newf("failed to persist assistant message: %w", err)
		}

		llmCall = buildLLMCall(assistantMessage.ID, resolved, generateReq, generateResp, llmLatencyMS)
		if err := s.storage.InsertLLMCall(txCtx, llmCall); err != nil {
			return xerrors.Newf("failed to persist llm call: %w", err)
		}

		execution.MessageID = assistantMessage.ID
		if err := s.storage.InsertExecution(txCtx, execution); err != nil {
			return xerrors.Newf("failed to persist execution: %w", err)
		}

		return nil
	}); err != nil {
		s.logSendMessageFailure("failed to persist assistant turn", err, params, connection)
		return nil, err
	}

	s.Logger().Info(
		"chat message completed",
		slog.Int64("conversation_id", conversation.ID),
		slog.Int("user_id", int(params.UserID)),
		slog.Int("connection_id", int(connection.ID)),
		slog.String("database", string(connection.Database)),
		slog.String("provider", string(params.Provider)),
		slog.String("model", params.Model),
		slog.Int("history_messages", len(history)),
		slog.Int("schema_chunks", len(schemaContext.Chunks)),
		slog.Int("llm_latency_ms", int(llmLatencyMS)),
		slog.Int("execution_latency_ms", int(executionLatencyMS)),
		slog.Int("total_latency_ms", int(elapsedMilliseconds(startedAt))),
		slog.Int("result_rows", int(result.RowCount)),
		slog.Bool("result_truncated", result.Truncated),
	)

	return &chattype.AssistantTurn{
		Conversation:     conversation,
		UserMessage:      userMessage,
		AssistantMessage: assistantMessage,
		Execution:        execution,
		Retrieval:        retrieval,
	}, nil
}

func (s *Service) logSendMessageFailure(
	message string,
	err error,
	params chattype.SendMessageParams,
	connection *connectiontypes.Connection,
	attrs ...slog.Attr,
) {
	args := []any{
		slog.Any("err", err),
		slog.Int64("conversation_id", params.ConversationID),
		slog.Int("user_id", int(params.UserID)),
		slog.String("provider", string(params.Provider)),
		slog.String("model", params.Model),
	}
	if connection != nil {
		args = append(args,
			slog.Int("connection_id", int(connection.ID)),
			slog.String("database", string(connection.Database)),
		)
	}
	for _, attr := range attrs {
		args = append(args, attr)
	}

	s.Logger().Warn(message, args...)
}

func (s *Service) getOwnedConversation(
	ctx context.Context,
	userID int32,
	conversationID int64,
) (*chattype.Conversation, *connectiontypes.Connection, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, nil, err
	}

	connection, err := s.getQueryableConnection(ctx, userID, conversation.ConnectionID)
	if err != nil {
		return nil, nil, err
	}

	return conversation, connection, nil
}

func (s *Service) getQueryableConnection(
	ctx context.Context,
	userID int32,
	connectionID int32,
) (*connectiontypes.Connection, error) {
	connection, err := s.connections.GetConnection(ctx, connectionID)
	if err != nil {
		return nil, xerrors.Newf("failed to fetch connection: %w", err)
	}
	if connection == nil {
		return nil, chaterrors.ErrConnectionAccessDenied
	}

	access, err := s.connections.GetAccess(ctx, userID, connectionID)
	if err != nil {
		if errors.Is(err, connectionerrors.ErrAccessNotFound) {
			return nil, chaterrors.ErrConnectionAccessDenied
		}
		return nil, xerrors.Newf("failed to fetch connection access: %w", err)
	}
	if access == nil || !access.CanQuery {
		return nil, chaterrors.ErrConnectionAccessDenied
	}

	return connection, nil
}

func isSupportedChatDatabase(database connectiontypes.Database) bool {
	switch database {
	case connectiontypes.DatabasePostgres, connectiontypes.DatabaseMySQL:
		return true
	default:
		return false
	}
}
