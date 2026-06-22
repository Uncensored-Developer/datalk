package db

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/factory"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/models"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/ordering"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pagination"
	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/aarondl/opt/null"
	"github.com/aarondl/opt/omit"
	"github.com/gotidy/ptr"
	"github.com/stephenafamo/bob/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorage_InsertGetAndListConversations(t *testing.T) {
	t.Parallel()

	primaryConn := createConnection(t, "chat-conversations-primary")
	secondaryConn := createConnection(t, "chat-conversations-secondary")

	title1 := "Subscription summary"
	title2 := "Weekly breakdown"
	title3 := "Other connection"

	conversation1 := &chattype.Conversation{
		UserID:       primaryConn.UserID,
		ConnectionID: primaryConn.ID,
		Title:        &title1,
	}
	conversation2 := &chattype.Conversation{
		UserID:       primaryConn.UserID,
		ConnectionID: primaryConn.ID,
		Title:        &title2,
	}
	conversation3 := &chattype.Conversation{
		UserID:       secondaryConn.UserID,
		ConnectionID: secondaryConn.ID,
		Title:        &title3,
	}

	require.NoError(t, s.UpsertConversation(t.Context(), conversation1))
	require.NoError(t, s.UpsertConversation(t.Context(), conversation2))
	require.NoError(t, s.UpsertConversation(t.Context(), conversation3))

	require.False(t, conversation1.CreatedAt.IsZero())
	require.False(t, conversation1.UpdatedAt.IsZero())

	got, err := s.GetConversation(t.Context(), conversation1.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, conversation1.ID, got.ID)
	assert.Equal(t, conversation1.UserID, got.UserID)
	assert.Equal(t, conversation1.ConnectionID, got.ConnectionID)
	require.NotNil(t, got.Title)
	assert.Equal(t, title1, *got.Title)

	updatedTitle := "Revenue Growth"
	conversation1.Title = &updatedTitle
	conversation1.UserID = secondaryConn.UserID
	conversation1.ConnectionID = secondaryConn.ID
	require.NoError(t, s.UpsertConversation(t.Context(), conversation1))
	updated, err := s.GetConversation(t.Context(), conversation1.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.NotNil(t, updated.Title)
	assert.Equal(t, updatedTitle, *updated.Title)
	assert.Equal(t, primaryConn.UserID, updated.UserID)
	assert.Equal(t, primaryConn.ID, updated.ConnectionID)

	listed, err := s.ListConversations(t.Context(), chatstorage.ConversationsFilter{
		UserID: []int32{primaryConn.UserID},
		Ordering: ordering.Orderings[chatstorage.ConversationOrdering]{
			ordering.OrderByAsc(chatstorage.ConversationOrderingID),
		},
	})
	require.NoError(t, err)
	require.Len(t, listed, 2)
	assert.Equal(t, []int64{conversation1.ID, conversation2.ID}, conversationIDs(listed))

	paged, err := s.ListConversations(t.Context(), chatstorage.ConversationsFilter{
		ConnectionID: []int32{primaryConn.ID},
		Pagination: pagination.LimitOffsetPagination{
			Limit:  1,
			Offset: 1,
		},
		Ordering: ordering.Orderings[chatstorage.ConversationOrdering]{
			ordering.OrderByAsc(chatstorage.ConversationOrderingID),
		},
	})
	require.NoError(t, err)
	require.Len(t, paged, 1)
	assert.Equal(t, conversation2.ID, paged[0].ID)
}

func TestStorage_InsertGetAndListMessages(t *testing.T) {
	t.Parallel()

	connection := createConnection(t, "chat-messages")
	conversation := insertConversation(t, connection.UserID, connection.ID, "Messages test")

	userMessage := &chattype.Message{
		ConversationID: conversation.ID,
		Role:           chattype.MessageRoleUser,
		Content:        "How many users subscribed this month?",
		Status:         chattype.MessageStatusCompleted,
	}
	assistantProvider := llm.ProviderOpenAI
	assistantModel := "gpt-5.2"
	errorMessage := "insufficient context"
	assistantFailed := &chattype.Message{
		ConversationID: conversation.ID,
		Role:           chattype.MessageRoleAssistant,
		Content:        "I could not generate a safe query.",
		Provider:       &assistantProvider,
		Model:          &assistantModel,
		Status:         chattype.MessageStatusFailed,
		ErrorMessage:   &errorMessage,
	}
	assistantCompleted := &chattype.Message{
		ConversationID:  conversation.ID,
		Role:            chattype.MessageRoleAssistant,
		Content:         "SELECT count(*) FROM users;",
		Provider:        &assistantProvider,
		Model:           &assistantModel,
		Status:          chattype.MessageStatusCompleted,
		NaturalResponse: ptr.Of("There are 42 subscribed users this month."),
	}

	require.NoError(t, s.InsertMessage(t.Context(), userMessage))
	require.NoError(t, s.InsertMessage(t.Context(), assistantFailed))
	require.NoError(t, s.InsertMessage(t.Context(), assistantCompleted))

	got, err := s.GetMessage(t.Context(), assistantFailed.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, assistantFailed.ID, got.ID)
	assert.Equal(t, chattype.MessageRoleAssistant, got.Role)
	assert.Equal(t, chattype.MessageStatusFailed, got.Status)
	require.NotNil(t, got.Provider)
	assert.Equal(t, llm.ProviderOpenAI, *got.Provider)
	require.NotNil(t, got.Model)
	assert.Equal(t, assistantModel, *got.Model)
	require.NotNil(t, got.ErrorMessage)
	assert.Equal(t, errorMessage, *got.ErrorMessage)
	assert.Nil(t, got.NaturalResponse)

	gotCompleted, err := s.GetMessage(t.Context(), assistantCompleted.ID)
	require.NoError(t, err)
	require.NotNil(t, gotCompleted)
	require.NotNil(t, gotCompleted.NaturalResponse)
	assert.Equal(t, "There are 42 subscribed users this month.", *gotCompleted.NaturalResponse)

	filtered, err := s.ListMessages(t.Context(), chatstorage.MessagesFilter{
		ConversationID: []int64{conversation.ID},
		Role:           []chattype.MessageRole{chattype.MessageRoleAssistant},
		Status:         []chattype.MessageStatus{chattype.MessageStatusFailed},
		Ordering: ordering.Orderings[chatstorage.MessageOrdering]{
			ordering.OrderByAsc(chatstorage.MessageOrderingID),
		},
	})
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	assert.Equal(t, assistantFailed.ID, filtered[0].ID)

	paged, err := s.ListMessages(t.Context(), chatstorage.MessagesFilter{
		ConversationID: []int64{conversation.ID},
		Pagination: pagination.LimitOffsetPagination{
			Limit:  2,
			Offset: 1,
		},
		Ordering: ordering.Orderings[chatstorage.MessageOrdering]{
			ordering.OrderByAsc(chatstorage.MessageOrderingID),
		},
	})
	require.NoError(t, err)
	require.Len(t, paged, 2)
	assert.Equal(t, []int64{assistantFailed.ID, assistantCompleted.ID}, messageIDs(paged))
	assert.Nil(t, paged[0].NaturalResponse)
	require.NotNil(t, paged[1].NaturalResponse)
	assert.Equal(t, "There are 42 subscribed users this month.", *paged[1].NaturalResponse)
}

func TestStorage_InsertAndGetExecution(t *testing.T) {
	t.Parallel()

	connection := createConnection(t, "chat-execution")
	conversation := insertConversation(t, connection.UserID, connection.ID, "Execution test")
	message := insertMessage(t, conversation.ID, chattype.MessageRoleAssistant, "SELECT count(*) FROM users;", chattype.MessageStatusCompleted)

	executedAt := time.Now().UTC().Truncate(time.Second)
	execution := &chattype.MessageExecution{
		MessageID:     message.ID,
		ConnectionID:  connection.ID,
		DatabaseKind:  connectiontypes.DatabasePostgres,
		GeneratedSQL:  "SELECT count(*) FROM users;",
		NormalizedSQL: "select count(*) from users;",
		Result: chattype.QueryResult{
			Columns: []chattype.ResultColumn{
				{Name: "count", DataType: "bigint"},
			},
			Rows: []map[string]any{
				{"count": float64(42)},
			},
			RowCount:  1,
			Truncated: false,
			Kind:      chattype.QueryResultKindScalar,
		},
		ExecutionLatencyMS: 123,
		ExecutedAt:         executedAt,
	}

	require.NoError(t, s.InsertExecution(t.Context(), execution))

	dbExecution, err := models.FindChatMessageExecution(t.Context(), runner.BobConn, message.ID)
	require.NoError(t, err)
	require.NotNil(t, dbExecution)
	assert.JSONEq(t, `{"columns":[{"name":"count","data_type":"bigint"}],"rows":[{"count":42}],"row_count":1,"truncated":false,"kind":"scalar"}`, string(dbExecution.ResultJSON.Val))

	got, err := s.GetExecution(t.Context(), message.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, execution.MessageID, got.MessageID)
	assert.Equal(t, execution.ConnectionID, got.ConnectionID)
	assert.Equal(t, execution.DatabaseKind, got.DatabaseKind)
	assert.Equal(t, execution.GeneratedSQL, got.GeneratedSQL)
	assert.Equal(t, execution.NormalizedSQL, got.NormalizedSQL)
	assert.Equal(t, execution.Result, got.Result)
	assert.Equal(t, execution.ExecutionLatencyMS, got.ExecutionLatencyMS)
	assert.WithinDuration(t, executedAt, got.ExecutedAt, time.Second)
}

func TestStorage_InsertAndGetRetrieval(t *testing.T) {
	t.Parallel()

	connection := createConnection(t, "chat-retrieval")
	snapshot := createSnapshot(t, connection.ID)
	conversation := insertConversation(t, connection.UserID, connection.ID, "Retrieval test")
	message := insertMessage(t, conversation.ID, chattype.MessageRoleUser, "Group that by week.", chattype.MessageStatusCompleted)

	retrievedAt := time.Now().UTC().Truncate(time.Second)
	retrieval := &chattype.MessageRetrieval{
		MessageID:  message.ID,
		SnapshotID: snapshot.ID,
		QueryText:  "Previous question: how many users subscribed this month. Current question: group that by week.",
		Chunks: []schematypes.RetrievedChunk{
			{
				ChunkID:    101,
				ObjectType: "table",
				ObjectName: "public.users",
				Content:    "table public.users columns: id, subscribed_at",
				SchemaJSON: json.RawMessage(`{"name":"users"}`),
				Similarity: 0.91,
			},
		},
		RetrievedAt: retrievedAt,
	}

	require.NoError(t, s.InsertRetrieval(t.Context(), retrieval))

	got, err := s.GetRetrieval(t.Context(), message.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, retrieval.MessageID, got.MessageID)
	assert.Equal(t, retrieval.SnapshotID, got.SnapshotID)
	assert.Equal(t, retrieval.QueryText, got.QueryText)
	require.Len(t, got.Chunks, 1)
	assert.Equal(t, retrieval.Chunks[0].ObjectName, got.Chunks[0].ObjectName)
	assert.Equal(t, retrieval.Chunks[0].Content, got.Chunks[0].Content)
	assert.JSONEq(t, string(retrieval.Chunks[0].SchemaJSON), string(got.Chunks[0].SchemaJSON))
	assert.InDelta(t, retrieval.Chunks[0].Similarity, got.Chunks[0].Similarity, 0.0001)
	assert.WithinDuration(t, retrievedAt, got.RetrievedAt, time.Second)
}

func TestStorage_InTransactionRollsBackOnError(t *testing.T) {
	t.Parallel()

	connection := createConnection(t, "chat-transaction-rollback")
	conversation := insertConversation(t, connection.UserID, connection.ID, "Rollback test")
	rollbackErr := errors.New("force rollback")

	err := s.InTransaction(t.Context(), func(ctx context.Context) error {
		return errors.Join(
			s.InsertMessage(ctx, &chattype.Message{
				ConversationID: conversation.ID,
				Role:           chattype.MessageRoleUser,
				Content:        "this should roll back",
				Status:         chattype.MessageStatusCompleted,
			}),
			rollbackErr,
		)
	})
	require.ErrorIs(t, err, rollbackErr)

	messages, err := s.ListMessages(t.Context(), chatstorage.MessagesFilter{
		ConversationID: []int64{conversation.ID},
	})
	require.NoError(t, err)
	assert.Empty(t, messages)
}

func TestStorage_InsertAndListLLMCalls(t *testing.T) {
	t.Parallel()

	connection := createConnection(t, "chat-llm-calls")
	conversation := insertConversation(t, connection.UserID, connection.ID, "LLM calls test")
	message := insertMessage(t, conversation.ID, chattype.MessageRoleAssistant, "SELECT count(*) FROM users;", chattype.MessageStatusCompleted)

	baseURL := "https://api.openai.test"
	providerConfig1 := &llm.ProviderConfig{
		Provider:    llm.ProviderOpenAI,
		DisplayName: "OpenAI",
		APIKeyEnc:   "enc-openai",
		BaseURL:     &baseURL,
		IsEnabled:   true,
		Metadata:    json.RawMessage(`{"tier":"prod"}`),
	}
	providerConfig2 := &llm.ProviderConfig{
		Provider:    llm.ProviderAnthropic,
		DisplayName: "Anthropic",
		APIKeyEnc:   "enc-anthropic",
		IsEnabled:   true,
		Metadata:    json.RawMessage(`{"tier":"backup"}`),
	}

	require.NoError(t, s.UpsertProviderConfig(t.Context(), providerConfig1))
	require.NoError(t, s.UpsertProviderConfig(t.Context(), providerConfig2))

	call1 := &chattype.MessageLLMCall{
		MessageID:        message.ID,
		ProviderConfigID: providerConfig1.ID,
		Provider:         llm.ProviderOpenAI,
		Model:            "gpt-5.2",
		RequestJSON:      json.RawMessage(`{"prompt":"generate sql"}`),
		ResponseJSON:     json.RawMessage(`{"sql":"select count(*) from users"}`),
		LatencyMS:        180,
	}
	inputTokens := int32(120)
	outputTokens := int32(40)
	call2 := &chattype.MessageLLMCall{
		MessageID:        message.ID,
		ProviderConfigID: providerConfig2.ID,
		Provider:         llm.ProviderAnthropic,
		Model:            "claude-sonnet",
		RequestJSON:      json.RawMessage(`{"prompt":"explain query"}`),
		ResponseJSON:     json.RawMessage(`{"sql":"select date_trunc('week', subscribed_at) from users"}`),
		InputTokens:      &inputTokens,
		OutputTokens:     &outputTokens,
		LatencyMS:        240,
	}

	require.NoError(t, s.InsertLLMCall(t.Context(), call1))
	require.NoError(t, s.InsertLLMCall(t.Context(), call2))

	assert.Nil(t, call1.InputTokens)
	assert.Nil(t, call1.OutputTokens)

	listed, err := s.ListLLMCalls(t.Context(), chatstorage.LLMCallsFilter{
		MessageID: []int64{message.ID},
		Ordering: ordering.Orderings[chatstorage.LLMCallOrdering]{
			ordering.OrderByAsc(chatstorage.LLMCallOrderingID),
		},
	})
	require.NoError(t, err)
	require.Len(t, listed, 2)
	assert.Equal(t, []int64{call1.ID, call2.ID}, llmCallIDs(listed))

	filtered, err := s.ListLLMCalls(t.Context(), chatstorage.LLMCallsFilter{
		ProviderConfigID: []int64{providerConfig1.ID},
	})
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	assert.Equal(t, call1.ID, filtered[0].ID)
	assert.Nil(t, filtered[0].InputTokens)
	assert.Nil(t, filtered[0].OutputTokens)
	assert.JSONEq(t, string(call1.RequestJSON), string(filtered[0].RequestJSON))
	assert.JSONEq(t, string(call1.ResponseJSON), string(filtered[0].ResponseJSON))
}

func TestStorage_InsertGetAndListProviderConfigs(t *testing.T) {
	t.Parallel()

	baseURL := "http://localhost:11434"
	config1 := &llm.ProviderConfig{
		Provider:    llm.ProviderOpenAI,
		DisplayName: "OpenAI Primary",
		APIKeyEnc:   "enc-openai-primary",
		IsEnabled:   true,
		Metadata:    json.RawMessage(`{"region":"us"}`),
	}
	config2 := &llm.ProviderConfig{
		Provider:    llm.ProviderOllama,
		DisplayName: "Local Ollama",
		APIKeyEnc:   "enc-ollama",
		BaseURL:     &baseURL,
		IsEnabled:   false,
		Metadata:    json.RawMessage(`{"local":true}`),
	}

	require.NoError(t, s.UpsertProviderConfig(t.Context(), config1))
	require.NoError(t, s.UpsertProviderConfig(t.Context(), config2))

	got, err := s.GetProviderConfig(t.Context(), config1.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, config1.ID, got.ID)
	assert.Equal(t, llm.ProviderOpenAI, got.Provider)
	assert.Equal(t, config1.DisplayName, got.DisplayName)
	assert.Equal(t, config1.APIKeyEnc, got.APIKeyEnc)
	assert.Nil(t, got.BaseURL)
	assert.True(t, got.IsEnabled)
	assert.JSONEq(t, string(config1.Metadata), string(got.Metadata))

	listed, err := s.ListProviderConfigs(t.Context(), chatstorage.ProviderConfigsFilter{
		ID:        []int64{config1.ID, config2.ID},
		Provider:  []llm.Provider{llm.ProviderOpenAI},
		IsEnabled: ptr.Of(true),
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, config1.ID, listed[0].ID)
	assert.Nil(t, listed[0].BaseURL)

	gotSecond, err := s.GetProviderConfig(t.Context(), config2.ID)
	require.NoError(t, err)
	require.NotNil(t, gotSecond)
	require.NotNil(t, gotSecond.BaseURL)
	assert.Equal(t, baseURL, *gotSecond.BaseURL)
	assert.JSONEq(t, string(config2.Metadata), string(gotSecond.Metadata))

	updatedBaseURL := "https://api.openai.test"
	updated := &llm.ProviderConfig{
		Provider:    llm.ProviderOpenAI,
		DisplayName: "OpenAI Updated",
		APIKeyEnc:   "enc-openai-updated",
		BaseURL:     &updatedBaseURL,
		IsEnabled:   false,
		Metadata:    json.RawMessage(`{"region":"eu"}`),
	}
	require.NoError(t, s.UpsertProviderConfig(t.Context(), updated))
	assert.Equal(t, config1.ID, updated.ID)
	assert.Equal(t, config1.CreatedAt, updated.CreatedAt)
	assert.GreaterOrEqual(t, updated.UpdatedAt.UnixNano(), config1.UpdatedAt.UnixNano())

	openAIConfigs, err := s.ListProviderConfigs(t.Context(), chatstorage.ProviderConfigsFilter{
		Provider: []llm.Provider{llm.ProviderOpenAI},
	})
	require.NoError(t, err)
	require.Len(t, openAIConfigs, 1)
	assert.Equal(t, "OpenAI Updated", openAIConfigs[0].DisplayName)
	assert.Equal(t, "enc-openai-updated", openAIConfigs[0].APIKeyEnc)
	require.NotNil(t, openAIConfigs[0].BaseURL)
	assert.Equal(t, updatedBaseURL, *openAIConfigs[0].BaseURL)
	assert.False(t, openAIConfigs[0].IsEnabled)
	assert.JSONEq(t, `{"region":"eu"}`, string(openAIConfigs[0].Metadata))
}

func TestStorage_UpsertAndListProviderModels(t *testing.T) {
	t.Parallel()

	config := &llm.ProviderConfig{
		Provider:    llm.ProviderOpenAI,
		DisplayName: "Provider for models",
		APIKeyEnc:   "enc-models",
		IsEnabled:   true,
	}
	require.NoError(t, s.UpsertProviderConfig(t.Context(), config))

	model := &llm.ProviderModel{
		ProviderConfigID: config.ID,
		Model:            "gpt-5.2",
		DisplayName:      "GPT-5.2",
		SupportsSQL:      true,
		IsEnabled:        true,
	}
	require.NoError(t, s.UpsertProviderModel(t.Context(), model))
	require.NotNil(t, model.ContextWindow)
	assert.Equal(t, int32(0), *model.ContextWindow)

	contextWindow := int32(16384)
	model.DisplayName = "GPT-5.2 Updated"
	model.ContextWindow = &contextWindow
	model.SupportsSQL = false
	model.IsEnabled = false
	require.NoError(t, s.UpsertProviderModel(t.Context(), model))

	listed, err := s.ListProviderModels(t.Context(), chatstorage.ProviderModelsFilter{
		ProviderConfigID: []int64{config.ID},
		Model:            []string{"gpt-5.2"},
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, model.ID, listed[0].ID)
	assert.Equal(t, "GPT-5.2 Updated", listed[0].DisplayName)
	require.NotNil(t, listed[0].ContextWindow)
	assert.Equal(t, contextWindow, *listed[0].ContextWindow)
	assert.False(t, listed[0].SupportsSQL)
	assert.False(t, listed[0].IsEnabled)

	supportsSQL := false
	isEnabled := false
	filtered, err := s.ListProviderModels(t.Context(), chatstorage.ProviderModelsFilter{
		ProviderConfigID: []int64{config.ID},
		SupportsSQL:      &supportsSQL,
		IsEnabled:        &isEnabled,
	})
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	assert.Equal(t, model.ID, filtered[0].ID)
}

func createConnection(t *testing.T, name string) *models.Connection {
	t.Helper()

	userTmpl := factory.UserTemplate{}
	createdUser := userTmpl.CreateOrFail(t.Context(), t, runner.BobConn)

	connTmpl := factory.ConnectionTemplate{}
	connTmpl.Apply(t.Context(),
		factory.ConnectionMods.Name(name),
		factory.ConnectionMods.Kind(string(connectiontypes.DatabasePostgres)),
		factory.ConnectionMods.DSN(null.From("postgres://"+name)),
		factory.ConnectionMods.UserID(createdUser.ID),
	)

	return connTmpl.CreateOrFail(t.Context(), t, runner.BobConn)
}

func createSnapshot(t *testing.T, connectionID int32) *models.SchemaSnapshot {
	t.Helper()

	snapshot, err := models.SchemaSnapshots.Insert(&models.SchemaSnapshotSetter{
		ConnectionID:   omit.From(connectionID),
		SchemaHash:     omit.From("hash-" + time.Now().UTC().Format("150405.000000000")),
		SliceJSON:      omit.From(types.NewJSON(json.RawMessage(`{"tables":[]}`))),
		IntrospectedAt: omit.From(time.Now().UTC()),
	}).One(t.Context(), runner.BobConn)
	require.NoError(t, err)

	return snapshot
}

func insertConversation(t *testing.T, userID, connectionID int32, title string) *chattype.Conversation {
	t.Helper()

	conversation := &chattype.Conversation{
		UserID:       userID,
		ConnectionID: connectionID,
		Title:        ptr.Of(title),
	}

	require.NoError(t, s.UpsertConversation(t.Context(), conversation))
	return conversation
}

func insertMessage(
	t *testing.T,
	conversationID int64,
	role chattype.MessageRole,
	content string,
	status chattype.MessageStatus,
) *chattype.Message {
	t.Helper()

	message := &chattype.Message{
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		Status:         status,
	}

	require.NoError(t, s.InsertMessage(t.Context(), message))
	return message
}

func conversationIDs(conversations []*chattype.Conversation) []int64 {
	ids := make([]int64, len(conversations))
	for index, conversation := range conversations {
		ids[index] = conversation.ID
	}

	return ids
}

func messageIDs(messages []*chattype.Message) []int64 {
	ids := make([]int64, len(messages))
	for index, message := range messages {
		ids[index] = message.ID
	}

	return ids
}

func llmCallIDs(calls []*chattype.MessageLLMCall) []int64 {
	ids := make([]int64, len(calls))
	for index, call := range calls {
		ids[index] = call.ID
	}

	return ids
}
