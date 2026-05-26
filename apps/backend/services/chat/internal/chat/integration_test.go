package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/ordering"
	chatllm "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/sqlrunner"
	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
	chatdb "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage/db"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_SendMessageIntegration_PostgresFollowUpAndModelSwitching(t *testing.T) {
	runner, cfg := requireIntegrationRunner(t, "chat_send_message")
	ctx := t.Context()

	require.NoError(t, seedSubscriptionTable(ctx, runner.Conn))
	userID, connectionID := seedChatUserAndConnection(t, runner.Conn, integrationPostgresDSN(cfg, runner.Schema))
	providerConfigIDs := seedProviderConfigs(t, runner.Conn)

	storage := chatdb.NewStorage(runner.Conn)
	conversation := &chattype.Conversation{
		UserID:       userID,
		ConnectionID: connectionID,
	}
	require.NoError(t, storage.InsertConversation(ctx, conversation))

	connectionService := &integrationConnectionService{
		connection: &connectiontypes.Connection{
			ID:       connectionID,
			Database: connectiontypes.DatabasePostgres,
			DSN:      integrationPostgresDSN(cfg, runner.Schema),
			UserID:   userID,
		},
		access: &connectiontypes.Access{UserID: userID, ConnectionID: connectionID, CanQuery: true},
	}
	schemaRetriever := &integrationSchemaRetriever{
		context: &schematypes.RetrievedSchemaContext{
			ConnectionID:   connectionID,
			SnapshotID:     seedChatSnapshot(t, runner.Conn, connectionID),
			EmbeddingModel: "nomic-embed-text",
			Chunks: []schematypes.RetrievedChunk{
				{
					ChunkID:    1,
					ObjectType: "table",
					ObjectName: "subscriptions",
					Content:    "subscriptions(id int, subscribed_at timestamptz)",
					SchemaJSON: json.RawMessage(`{"table":"subscriptions"}`),
					Similarity: 0.99,
				},
			},
			RetrievedAt: time.Now().UTC(),
		},
	}
	client := &integrationSQLClient{
		responses: map[string]string{
			"gpt-5.2": "SELECT count(*)::int AS total FROM subscriptions WHERE subscribed_at >= DATE '2026-05-01' AND subscribed_at < DATE '2026-06-01'",
			"claude":  "SELECT date_trunc('week', subscribed_at)::date::text AS week, count(*)::int AS total FROM subscriptions WHERE subscribed_at >= DATE '2026-05-01' AND subscribed_at < DATE '2026-06-01' GROUP BY 1 ORDER BY 1",
		},
	}
	resolver := &integrationClientResolver{client: client, providerConfigIDs: providerConfigIDs}
	service := NewService(
		config.Config{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		storage,
		connectionService,
		nil,
		schemaRetriever,
		resolver,
		sqlrunner.NewRunner(),
	)

	firstTurn, err := service.SendMessage(ctx, chattype.SendMessageParams{
		UserID:         userID,
		ConversationID: conversation.ID,
		Content:        "how many users subscribed in May 2026?",
		Provider:       llmtypes.ProviderOpenAI,
		Model:          "gpt-5.2",
	})
	require.NoError(t, err)
	require.NotNil(t, firstTurn)
	require.Len(t, firstTurn.Execution.Result.Rows, 1)
	assert.EqualValues(t, int64(2), firstTurn.Execution.Result.Rows[0]["total"])
	assert.Equal(t, llmtypes.ProviderOpenAI, *firstTurn.AssistantMessage.Provider)
	assert.Equal(t, "gpt-5.2", *firstTurn.AssistantMessage.Model)

	secondTurn, err := service.SendMessage(ctx, chattype.SendMessageParams{
		UserID:         userID,
		ConversationID: conversation.ID,
		Content:        "group that by week",
		Provider:       llmtypes.ProviderAnthropic,
		Model:          "claude",
	})
	require.NoError(t, err)
	require.NotNil(t, secondTurn)
	require.Len(t, secondTurn.Execution.Result.Rows, 2)
	assert.Equal(t, llmtypes.ProviderAnthropic, *secondTurn.AssistantMessage.Provider)
	assert.Equal(t, "claude", *secondTurn.AssistantMessage.Model)

	require.Len(t, schemaRetriever.queries, 2)
	assert.Contains(t, schemaRetriever.queries[1], "Current follow-up question: group that by week")
	assert.Contains(t, schemaRetriever.queries[1], "Previous SQL: SELECT count(*)::int AS total FROM subscriptions")

	require.Len(t, client.requests, 2)
	assert.Equal(t, "gpt-5.2", client.requests[0].Model)
	assert.Equal(t, "claude", client.requests[1].Model)
	assert.Len(t, client.requests[1].Conversation.History, 2)

	messages, err := storage.ListMessages(ctx, chatstorage.MessagesFilter{
		ConversationID: []int64{conversation.ID},
		Ordering: ordering.Orderings[chatstorage.MessageOrdering]{
			ordering.OrderByAsc(chatstorage.MessageOrderingID),
		},
	})
	require.NoError(t, err)
	require.Len(t, messages, 4)
	assert.Equal(t, []chattype.MessageRole{
		chattype.MessageRoleUser,
		chattype.MessageRoleAssistant,
		chattype.MessageRoleUser,
		chattype.MessageRoleAssistant,
	}, []chattype.MessageRole{messages[0].Role, messages[1].Role, messages[2].Role, messages[3].Role})
}

type integrationConnectionService struct {
	connection *connectiontypes.Connection
	access     *connectiontypes.Access
}

func (s *integrationConnectionService) GetConnection(context.Context, int32) (*connectiontypes.Connection, error) {
	return s.connection, nil
}

func (s *integrationConnectionService) GetAccess(context.Context, int32, int32) (*connectiontypes.Access, error) {
	return s.access, nil
}

type integrationSchemaRetriever struct {
	context *schematypes.RetrievedSchemaContext
	queries []string
}

func (r *integrationSchemaRetriever) RetrieveRelevantSchemaContext(_ context.Context, params schematypes.RetrieveRelevantSchemaContextParams) (*schematypes.RetrievedSchemaContext, error) {
	r.queries = append(r.queries, params.QueryText)
	next := *r.context
	next.QueryText = params.QueryText
	next.RetrievedAt = time.Now().UTC()
	return &next, nil
}

type integrationClientResolver struct {
	client            *integrationSQLClient
	providerConfigIDs map[llmtypes.Provider]int64
}

func (r *integrationClientResolver) ResolveClient(_ context.Context, provider llmtypes.Provider, modelID string) (*chatllm.ResolvedClient, error) {
	return &chatllm.ResolvedClient{
		ResolvedModel: &chatllm.ResolvedModel{
			ProviderConfig:   &llmtypes.ProviderConfig{ID: r.providerConfigIDs[provider], Provider: provider, DisplayName: string(provider), APIKeyEnc: "test", IsEnabled: true},
			ProviderModelID:  modelID,
			QualifiedModelID: chatllm.QualifiedModelID(provider, modelID),
			Model:            llmtypes.Model{ID: chatllm.QualifiedModelID(provider, modelID), Provider: provider, DisplayName: modelID, IsEnabled: true},
		},
		Client: r.client,
	}, nil
}

type integrationSQLClient struct {
	responses map[string]string
	requests  []llmtypes.GenerateSQLRequest
}

func (c *integrationSQLClient) ListModels(context.Context) ([]llmtypes.Model, error) {
	return nil, nil
}

func (c *integrationSQLClient) GenerateSQL(_ context.Context, req llmtypes.GenerateSQLRequest) (*llmtypes.GenerateSQLResponse, error) {
	c.requests = append(c.requests, req)
	sqlText := c.responses[req.Model]
	return &llmtypes.GenerateSQLResponse{
		SQL:         sqlText,
		Explanation: "integration query executed",
		RawRequest:  json.RawMessage(`{"integration":true}`),
		RawResponse: json.RawMessage(`{"sql":true}`),
	}, nil
}

func integrationPostgresDSN(cfg config.Config, schema string) string {
	return fmt.Sprintf(
		"user=%s password=%s host=%s port=%d dbname=%s sslmode=%s search_path=test%s",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBSSLMode,
		schema,
	)
}

func seedSubscriptionTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE subscriptions (
			id SERIAL PRIMARY KEY,
			subscribed_at TIMESTAMPTZ NOT NULL
		);
		INSERT INTO subscriptions (subscribed_at) VALUES
			('2026-05-03T10:00:00Z'),
			('2026-05-12T10:00:00Z'),
			('2026-04-22T10:00:00Z');
	`)
	return err
}

func seedChatUserAndConnection(t *testing.T, db *sql.DB, dsn string) (int32, int32) {
	t.Helper()

	var userID int32
	err := db.QueryRowContext(
		t.Context(),
		`INSERT INTO users (email, name, password_hash, role) VALUES ($1, 'Integration User', 'hash', 'member') RETURNING id`,
		fmt.Sprintf("chat-%d@example.com", time.Now().UnixNano()),
	).Scan(&userID)
	require.NoError(t, err)

	var connectionID int32
	err = db.QueryRowContext(
		t.Context(),
		`INSERT INTO connections (name, kind, dsn, user_id) VALUES ($1, 'postgres', $2, $3) RETURNING id`,
		fmt.Sprintf("chat-integration-%d", time.Now().UnixNano()),
		dsn,
		userID,
	).Scan(&connectionID)
	require.NoError(t, err)

	return userID, connectionID
}

func seedChatSnapshot(t *testing.T, db *sql.DB, connectionID int32) int32 {
	t.Helper()

	var snapshotID int32
	err := db.QueryRowContext(
		t.Context(),
		`INSERT INTO schema_snapshots (connection_id, schema_hash, slice_json) VALUES ($1, $2, '{}'::jsonb) RETURNING id`,
		connectionID,
		fmt.Sprintf("chat-snapshot-%d", time.Now().UnixNano()),
	).Scan(&snapshotID)
	require.NoError(t, err)

	return snapshotID
}

func seedProviderConfigs(t *testing.T, db *sql.DB) map[llmtypes.Provider]int64 {
	t.Helper()

	configs := map[llmtypes.Provider]int64{}
	for _, provider := range []llmtypes.Provider{llmtypes.ProviderOpenAI, llmtypes.ProviderAnthropic} {
		var id int64
		err := db.QueryRowContext(
			t.Context(),
			`INSERT INTO llm_provider_configs (provider, display_name, api_key_enc, is_enabled) VALUES ($1, $2, 'enc-test', TRUE) RETURNING id`,
			string(provider),
			string(provider),
		).Scan(&id)
		require.NoError(t, err)
		configs[provider] = id
	}

	return configs
}
