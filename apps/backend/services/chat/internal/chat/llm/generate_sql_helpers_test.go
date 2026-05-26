package llm

import (
	"testing"

	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSQLSystemPrompt(t *testing.T) {
	t.Parallel()

	prompt := GenerateSQLSystemPrompt(llmtypes.GenerateSQLRequest{
		Conversation: llmtypes.ConversationContext{
			DatabaseKind: connectiontypes.DatabasePostgres,
			History: []llmtypes.ConversationMessage{
				{Role: "user", Content: "previous question"},
			},
		},
		UserPrompt: "count users",
		Schema: schematypes.RetrievedSchemaContext{
			Chunks: []schematypes.RetrievedChunk{
				{ObjectType: "table", ObjectName: "public.users", Content: "columns: id, subscribed_at"},
				{ObjectType: "table", ObjectName: "public.subscriptions", Content: "columns: user_id, started_at"},
			},
		},
		Options: llmtypes.GenerateSQLOptions{MaxSchemaChunks: 1},
	})

	assert.Contains(t, prompt, "Database dialect: postgres")
	assert.Contains(t, prompt, "Generate exactly one SQL statement.")
	assert.Contains(t, prompt, "Generate only SELECT queries or WITH queries that end in SELECT.")
	assert.Contains(t, prompt, "table public.users")
	assert.NotContains(t, prompt, "public.subscriptions")
	assert.NotContains(t, prompt, "previous question")
	assert.NotContains(t, prompt, "count users")
}

func TestGenerateSQLSchemaRequiresEveryProperty(t *testing.T) {
	t.Parallel()

	schema := GenerateSQLSchema()

	assert.Equal(t, []string{"sql", "explanation", "assumptions", "confidence"}, schema["required"])
	assert.False(t, schema["additionalProperties"].(bool))

	properties := schema["properties"].(map[string]any)
	assumptions := properties["assumptions"].(map[string]any)
	confidence := properties["confidence"].(map[string]any)
	assert.Equal(t, []string{"array", "null"}, assumptions["type"])
	assert.Equal(t, []string{"number", "null"}, confidence["type"])
}

func TestGenerateSQLMessages(t *testing.T) {
	t.Parallel()

	messages := GenerateSQLMessages(llmtypes.GenerateSQLRequest{
		UserPrompt: " group that by week ",
		Conversation: llmtypes.ConversationContext{
			History: []llmtypes.ConversationMessage{
				{Role: "user", Content: "first question"},
				{Role: "assistant", Content: "first answer"},
				{Role: "user", Content: " second question "},
				{Role: "", Content: "ignored"},
			},
		},
		Options: llmtypes.GenerateSQLOptions{MaxHistoryMessages: 3},
	})

	require.Len(t, messages, 3)
	assert.Equal(t, PromptMessage{Role: "assistant", Content: "first answer"}, messages[0])
	assert.Equal(t, PromptMessage{Role: "user", Content: "second question"}, messages[1])
	assert.Equal(t, PromptMessage{Role: "user", Content: "group that by week"}, messages[2])
}
