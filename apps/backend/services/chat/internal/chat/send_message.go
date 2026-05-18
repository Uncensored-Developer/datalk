package chat

import (
	"context"
	"errors"
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
	userContent := strings.TrimSpace(params.Content)
	if userContent == "" {
		return nil, xerrors.New("message content is required")
	}

	conversation, connection, err := s.getOwnedConversation(ctx, params.UserID, params.ConversationID)
	if err != nil {
		return nil, err
	}

	if !isSupportedChatDatabase(connection.Database) {
		return nil, xerrors.Newf("%s: %w", connection.Database, chaterrors.ErrUnsupportedDatabaseKind)
	}

	resolved, err := s.clientResolver.ResolveClient(ctx, params.Provider, params.Model)
	if err != nil {
		return nil, err
	}
	if resolved == nil || resolved.ProviderConfig == nil || resolved.Client == nil {
		return nil, xerrors.Newf("model resolver returned incomplete client: %w", chaterrors.ErrModelNotAvailable)
	}

	history, err := s.loadRecentHistory(ctx, conversation.ID, 0)
	if err != nil {
		return nil, err
	}

	lastAssistantSQL, err := s.latestAssistantSQL(ctx, history)
	if err != nil {
		return nil, err
	}

	retrievalQuery := buildRetrievalQuery(userContent, toConversationMessages(history), lastAssistantSQL)
	schemaContext, err := s.schemaRetriever.RetrieveRelevantSchemaContext(ctx, schematypes.RetrieveRelevantSchemaContextParams{
		ConnectionID: conversation.ConnectionID,
		QueryText:    retrievalQuery,
		Limit:        defaultSchemaChunkLimit,
	})
	if err != nil {
		return nil, err
	}
	if schemaContext == nil {
		return nil, xerrors.Newf("schema retrieval returned no context: %w", chaterrors.ErrEmbeddedSnapshotNotReady)
	}

	retrieval := &chattype.MessageRetrieval{
		SnapshotID:  schemaContext.SnapshotID,
		QueryText:   retrievalQuery,
		Chunks:      schemaContext.Chunks,
		RetrievedAt: schemaContext.RetrievedAt,
	}

	generateReq := buildGenerateSQLRequest(conversation, connection.Database, userContent, history, schemaContext, resolved.ProviderModelID)

	llmStarted := time.Now()
	generateResp, err := resolved.Client.GenerateSQL(ctx, generateReq)
	llmLatencyMS := elapsedMilliseconds(llmStarted)
	if err != nil {
		return nil, xerrors.Newf("failed to generate sql: %w", err)
	}
	if generateResp == nil {
		return nil, xerrors.New("provider returned empty sql response")
	}

	if err := sqlrunner.NewValidator().Validate(connection.Database, generateResp.SQL); err != nil {
		return nil, xerrors.Newf("failed to validate sql: %w", err)
	}

	execStarted := time.Now()
	result, err := s.sqlRunner.Run(ctx, *connection, generateResp.SQL, sqlrunner.RunOptions{
		Timeout:  defaultQueryTimeout,
		RowLimit: defaultResultRowLimit,
	})
	executionLatencyMS := elapsedMilliseconds(execStarted)
	if err != nil {
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
		return nil, err
	}

	return &chattype.AssistantTurn{
		Conversation:     conversation,
		UserMessage:      userMessage,
		AssistantMessage: assistantMessage,
		Execution:        execution,
		Retrieval:        retrieval,
	}, nil
}

func (s *Service) getOwnedConversation(
	ctx context.Context,
	userID int32,
	conversationID int64,
) (*chattype.Conversation, *connectiontypes.Connection, error) {
	conversation, err := s.storage.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, nil, xerrors.Newf("failed to fetch conversation: %w", err)
	}
	if conversation == nil || conversation.UserID != userID {
		return nil, nil, chaterrors.ErrConversationNotFound
	}

	connection, err := s.connections.GetConnection(ctx, conversation.ConnectionID)
	if err != nil {
		return nil, nil, xerrors.Newf("failed to fetch connection: %w", err)
	}
	if connection == nil {
		return nil, nil, chaterrors.ErrConnectionAccessDenied
	}

	access, err := s.connections.GetAccess(ctx, userID, conversation.ConnectionID)
	if err != nil {
		if errors.Is(err, connectionerrors.ErrAccessNotFound) {
			return nil, nil, chaterrors.ErrConnectionAccessDenied
		}
		return nil, nil, xerrors.Newf("failed to fetch connection access: %w", err)
	}
	if access == nil || !access.CanQuery {
		return nil, nil, chaterrors.ErrConnectionAccessDenied
	}

	return conversation, connection, nil
}

func isSupportedChatDatabase(database connectiontypes.Database) bool {
	switch database {
	case connectiontypes.DatabasePostgres, connectiontypes.DatabaseMySQL:
		return true
	default:
		return false
	}
}
