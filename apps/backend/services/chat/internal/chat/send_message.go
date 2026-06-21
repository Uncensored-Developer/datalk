package chat

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	chatllm "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/sqlrunner"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	connectionerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/errors"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/gotidy/ptr"
	"github.com/mdobak/go-xerrors"
)

const maxSQLAttempts = 3

type sqlAttemptResult struct {
	Request   llmtypes.GenerateSQLRequest
	Response  *llmtypes.GenerateSQLResponse
	LatencyMS int32
}

func (s *Service) SendMessage(ctx context.Context, params chattype.SendMessageParams) (*chattype.AssistantTurn, error) {
	return s.SendMessageWithProgress(ctx, params, nil)
}

func (s *Service) SendMessageWithProgress(ctx context.Context, params chattype.SendMessageParams, progress chattype.SendMessageProgressHandler) (*chattype.AssistantTurn, error) {
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
	if err := emitSendMessageProgress(progress, chattype.SendMessageProgress{
		Stage: chattype.SendMessageProgressRetrievingSchema,
	}); err != nil {
		return nil, err
	}
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

	var (
		attempts           []sqlAttemptResult
		correctionAttempts []llmtypes.SQLCorrectionAttempt
		generateResp       *llmtypes.GenerateSQLResponse
		result             *chattype.QueryResult
		finalSQLError      error
		answerReq          *llmtypes.GenerateAnswerRequest
		answerResp         *llmtypes.GenerateAnswerResponse
		answerLatencyMS    int32
		answerErr          error
		executionLatencyMS int32
	)
	for attemptNumber := 1; attemptNumber <= maxSQLAttempts; attemptNumber++ {
		generateReq := buildGenerateSQLRequest(conversation, connection.Database, userContent, history, schemaContext, resolved.ProviderModelID)
		if len(correctionAttempts) > 0 {
			generateReq.Correction = &llmtypes.SQLCorrectionContext{
				AttemptNumber: attemptNumber - 1,
				Attempts:      correctionAttempts,
			}
		}
		if err := enforceGenerateSQLRequestLimits(&generateReq); err != nil {
			s.logSendMessageFailure("generated sql request exceeded limits", err, params, connection,
				slog.Int("schema_chunks", len(generateReq.Schema.Chunks)),
				slog.Int("max_prompt_bytes", generateReq.Options.MaxPromptBytes),
			)
			return nil, err
		}

		if err := emitSendMessageProgress(progress, chattype.SendMessageProgress{
			Stage:   chattype.SendMessageProgressGeneratingSQL,
			Attempt: attemptNumber,
		}); err != nil {
			return nil, err
		}
		llmStarted := time.Now()
		resp, err := resolved.Client.GenerateSQL(ctx, generateReq)
		llmLatencyMS := elapsedMilliseconds(llmStarted)
		if err != nil {
			s.logSendMessageFailure("failed to generate sql", err, params, connection, slog.Int("llm_latency_ms", int(llmLatencyMS)))
			return nil, xerrors.Newf("failed to generate sql: %w", err)
		}
		if resp == nil {
			err := xerrors.New("provider returned empty sql response")
			s.logSendMessageFailure("provider returned empty sql response", err, params, connection, slog.Int("llm_latency_ms", int(llmLatencyMS)))
			return nil, err
		}

		attempts = append(attempts, sqlAttemptResult{
			Request:   generateReq,
			Response:  resp,
			LatencyMS: llmLatencyMS,
		})
		generateResp = resp

		if err := emitSendMessageProgress(progress, chattype.SendMessageProgress{
			Stage:   chattype.SendMessageProgressExecutingSQL,
			Attempt: attemptNumber,
		}); err != nil {
			return nil, err
		}
		execStarted := time.Now()
		result, err = s.sqlRunner.Run(ctx, *connection, resp.SQL, sqlrunner.RunOptions{
			Timeout:  defaultQueryTimeout,
			RowLimit: defaultResultRowLimit,
		})
		executionLatencyMS = elapsedMilliseconds(execStarted)
		if err == nil {
			finalSQLError = nil
			break
		}

		finalSQLError = err
		s.logSendMessageFailure("failed to execute generated sql", err, params, connection,
			slog.Int("attempt", attemptNumber),
			slog.Bool("correction_eligible", sqlrunner.IsCorrectionEligible(err)),
			slog.Int("execution_latency_ms", int(executionLatencyMS)),
		)
		if !sqlrunner.IsCorrectionEligible(err) {
			return nil, err
		}

		correctionAttempts = append(correctionAttempts, llmtypes.SQLCorrectionAttempt{
			SQL:   resp.SQL,
			Error: sanitizeSQLCorrectionError(err),
		})
		if attemptNumber < maxSQLAttempts {
			if err := emitSendMessageProgress(progress, chattype.SendMessageProgress{
				Stage:   chattype.SendMessageProgressRegeneratingSQL,
				Attempt: attemptNumber + 1,
			}); err != nil {
				return nil, err
			}
		}
	}
	if finalSQLError == nil && params.RequireNaturalResponse {
		req := buildGenerateAnswerRequest(conversation, connection.Database, userContent, history, resolved.ProviderModelID, generateResp.SQL, *result)
		answerReq = &req
		if err := emitSendMessageProgress(progress, chattype.SendMessageProgress{
			Stage: chattype.SendMessageProgressGeneratingResponse,
		}); err != nil {
			return nil, err
		}
		answerStarted := time.Now()
		answerResp, answerErr = resolved.Client.GenerateAnswer(ctx, req)
		answerLatencyMS = elapsedMilliseconds(answerStarted)
		if answerErr != nil {
			s.logSendMessageFailure("failed to generate natural language answer", answerErr, params, connection, slog.Int("answer_latency_ms", int(answerLatencyMS)))
		} else if answerResp == nil {
			answerErr = xerrors.New("provider returned empty answer response")
			s.logSendMessageFailure("provider returned empty answer response", answerErr, params, connection, slog.Int("answer_latency_ms", int(answerLatencyMS)))
		}
	}

	userMessage := &chattype.Message{
		ConversationID: conversation.ID,
		Role:           chattype.MessageRoleUser,
		Content:        userContent,
		Status:         chattype.MessageStatusCompleted,
	}
	messageStatus := chattype.MessageStatusCompleted
	if finalSQLError != nil {
		messageStatus = chattype.MessageStatusFailed
	}
	assistantMessage := &chattype.Message{
		ConversationID: conversation.ID,
		Role:           chattype.MessageRoleAssistant,
		Content:        strings.TrimSpace(generateResp.Explanation),
		Provider:       &params.Provider,
		Model:          &params.Model,
		Status:         messageStatus,
	}
	if assistantMessage.Content == "" {
		if finalSQLError != nil {
			assistantMessage.Content = "I couldn't produce a SQL query that executed successfully."
		} else {
			assistantMessage.Content = "Query executed successfully."
		}
	}
	if finalSQLError != nil {
		assistantMessage.ErrorMessage = ptr.Of(sanitizeSQLCorrectionError(finalSQLError))
	}
	if answerResp != nil {
		assistantMessage.NaturalResponse = ptr.Of(strings.TrimSpace(answerResp.Answer))
	}
	if answerErr != nil {
		assistantMessage.ErrorMessage = ptr.Of(sanitizeSQLCorrectionError(answerErr))
	}

	var execution *chattype.MessageExecution
	if finalSQLError == nil {
		execution = &chattype.MessageExecution{
			ConnectionID:       connection.ID,
			DatabaseKind:       connection.Database,
			GeneratedSQL:       generateResp.SQL,
			NormalizedSQL:      generateResp.SQL,
			Result:             *result,
			ExecutionLatencyMS: executionLatencyMS,
			ExecutedAt:         time.Now().UTC(),
		}
	}
	llmCalls := make([]*chattype.MessageLLMCall, 0, len(attempts))
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

		for _, attempt := range attempts {
			llmCall := buildLLMCall(assistantMessage.ID, resolved, attempt.Request, attempt.Response, attempt.LatencyMS)
			if err := s.storage.InsertLLMCall(txCtx, llmCall); err != nil {
				return xerrors.Newf("failed to persist llm call: %w", err)
			}
			llmCalls = append(llmCalls, llmCall)
		}
		if answerReq != nil && answerResp != nil {
			llmCall := buildAnswerLLMCall(assistantMessage.ID, resolved, *answerReq, answerResp, answerLatencyMS)
			if err := s.storage.InsertLLMCall(txCtx, llmCall); err != nil {
				return xerrors.Newf("failed to persist answer llm call: %w", err)
			}
			llmCalls = append(llmCalls, llmCall)
		}

		if execution != nil {
			execution.MessageID = assistantMessage.ID
			if err := s.storage.InsertExecution(txCtx, execution); err != nil {
				return xerrors.Newf("failed to persist execution: %w", err)
			}
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
		slog.Int("sql_attempts", len(attempts)),
		slog.Int("execution_latency_ms", int(executionLatencyMS)),
		slog.Int("total_latency_ms", int(elapsedMilliseconds(startedAt))),
		slog.Bool("sql_failed", finalSQLError != nil),
		slog.Bool("natural_response_requested", params.RequireNaturalResponse),
		slog.Bool("natural_response_generated", answerResp != nil),
		slog.Int("result_rows", resultRowCount(result)),
		slog.Bool("result_truncated", resultTruncated(result)),
	)

	return &chattype.AssistantTurn{
		Conversation:     conversation,
		UserMessage:      userMessage,
		AssistantMessage: assistantMessage,
		Execution:        execution,
		Retrieval:        retrieval,
	}, nil
}

func emitSendMessageProgress(progress chattype.SendMessageProgressHandler, event chattype.SendMessageProgress) error {
	if progress == nil {
		return nil
	}
	return progress(event)
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

	if s.users != nil {
		user, err := s.users.GetUser(ctx, userID)
		if err != nil {
			return nil, xerrors.Newf("failed to fetch user: %w", err)
		}
		if user.IsAdmin() {
			return connection, nil
		}
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

func buildGenerateAnswerRequest(
	conversation *chattype.Conversation,
	databaseKind connectiontypes.Database,
	userPrompt string,
	history []*chattype.Message,
	modelID string,
	generatedSQL string,
	result chattype.QueryResult,
) llmtypes.GenerateAnswerRequest {
	return llmtypes.GenerateAnswerRequest{
		Model: modelID,
		Conversation: llmtypes.ConversationContext{
			ConversationID: conversation.ID,
			ConnectionID:   conversation.ConnectionID,
			DatabaseKind:   databaseKind,
			History:        toConversationMessages(history),
		},
		UserPrompt:   userPrompt,
		GeneratedSQL: generatedSQL,
		DatabaseKind: databaseKind,
		Result:       chatllm.BuildQueryResultPreview(result, defaultAnswerResultRows, defaultAnswerResultBytes),
		Options: llmtypes.GenerateAnswerOptions{
			MaxHistoryMessages: defaultHistoryLimit,
			MaxResultRows:      defaultAnswerResultRows,
			MaxResultBytes:     defaultAnswerResultBytes,
		},
	}
}

func sanitizeSQLCorrectionError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return ""
	}

	message = redactConnectionStrings(message)
	const maxErrorLength = 2000
	if len(message) > maxErrorLength {
		return message[:maxErrorLength] + "..."
	}
	return message
}

func redactConnectionStrings(message string) string {
	fields := strings.Fields(message)
	for index, field := range fields {
		normalized := strings.ToLower(field)
		if strings.Contains(normalized, "postgres://") ||
			strings.Contains(normalized, "postgresql://") ||
			strings.Contains(normalized, "mysql://") {
			fields[index] = redactedValue
		}
	}
	return strings.Join(fields, " ")
}

func resultRowCount(result *chattype.QueryResult) int {
	if result == nil {
		return 0
	}
	return int(result.RowCount)
}

func resultTruncated(result *chattype.QueryResult) bool {
	return result != nil && result.Truncated
}
