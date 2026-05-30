package llm

import (
	"testing"

	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAnswerSchemaRequiresEveryProperty(t *testing.T) {
	t.Parallel()

	schema := GenerateAnswerSchema()

	assert.Equal(t, []string{"answer", "limitations"}, schema["required"])
	assert.False(t, schema["additionalProperties"].(bool))

	properties := schema["properties"].(map[string]any)
	limitations := properties["limitations"].(map[string]any)
	assert.Equal(t, []string{"array", "null"}, limitations["type"])
}

func TestGenerateAnswerSystemPromptIncludesSQLAndResultPreview(t *testing.T) {
	t.Parallel()

	prompt := GenerateAnswerSystemPrompt(llmtypes.GenerateAnswerRequest{
		DatabaseKind: connectiontypes.DatabasePostgres,
		GeneratedSQL: "SELECT email, COUNT(*) AS transactions FROM transactions GROUP BY email",
		Result: llmtypes.QueryResultPreview{
			Columns: []llmtypes.QueryResultColumn{
				{Name: "email", DataType: "text"},
				{Name: "transactions", DataType: "int8"},
			},
			Rows: []map[string]any{
				{"email": "admin@datalk.app", "transactions": float64(200)},
			},
			RowCount:  1,
			Truncated: false,
			Kind:      string(chattype.QueryResultKindRecord),
		},
	})

	assert.Contains(t, prompt, "Database dialect: postgres")
	assert.Contains(t, prompt, "Executed SQL:")
	assert.Contains(t, prompt, "SELECT email")
	assert.Contains(t, prompt, `"email": "admin@datalk.app"`)
	assert.Contains(t, prompt, `"row_count": 1`)
	assert.Contains(t, prompt, "Do not include a markdown table")
}

func TestGenerateAnswerMessages(t *testing.T) {
	t.Parallel()

	messages := GenerateAnswerMessages(llmtypes.GenerateAnswerRequest{
		UserPrompt: " who transacted the most? ",
		Conversation: llmtypes.ConversationContext{
			History: []llmtypes.ConversationMessage{
				{Role: "user", Content: "first question"},
				{Role: "assistant", Content: "first answer"},
				{Role: "user", Content: " second question "},
			},
		},
		Options: llmtypes.GenerateAnswerOptions{MaxHistoryMessages: 2},
	})

	require.Len(t, messages, 3)
	assert.Equal(t, PromptMessage{Role: "assistant", Content: "first answer"}, messages[0])
	assert.Equal(t, PromptMessage{Role: "user", Content: "second question"}, messages[1])
	assert.Equal(t, PromptMessage{Role: "user", Content: "who transacted the most?"}, messages[2])
}

func TestBuildQueryResultPreviewAppliesRowAndByteLimits(t *testing.T) {
	t.Parallel()

	preview := BuildQueryResultPreview(chattype.QueryResult{
		Columns: []chattype.ResultColumn{
			{Name: "email", DataType: "text"},
			{Name: "transactions", DataType: "int8"},
		},
		Rows: []map[string]any{
			{"email": "admin@datalk.app", "transactions": int64(200)},
			{"email": "member@datalk.app", "transactions": int64(100)},
			{"email": "viewer@datalk.app", "transactions": int64(50)},
		},
		RowCount: 3,
		Kind:     chattype.QueryResultKindTable,
	}, 2, 0)

	require.Len(t, preview.Columns, 2)
	require.Len(t, preview.Rows, 2)
	assert.Equal(t, int32(3), preview.RowCount)
	assert.True(t, preview.Truncated)
	assert.Equal(t, string(chattype.QueryResultKindTable), preview.Kind)

	byteLimited := BuildQueryResultPreview(chattype.QueryResult{
		Rows: []map[string]any{
			{"long_value": "this row is intentionally too long for the small preview budget"},
		},
		RowCount: 1,
	}, 0, 8)

	assert.Empty(t, byteLimited.Rows)
	assert.True(t, byteLimited.Truncated)
}

func TestParseGenerateAnswerResponse(t *testing.T) {
	t.Parallel()

	finishReason := "stop"
	response, err := ParseGenerateAnswerResponse(
		[]byte(`{"request":true}`),
		[]byte(`{"response":true}`),
		`{"answer":" admin@datalk.app has the most transactions. ","limitations":null}`,
		&llmtypes.Usage{},
		&finishReason,
	)
	require.NoError(t, err)

	assert.Equal(t, "admin@datalk.app has the most transactions.", response.Answer)
	assert.Nil(t, response.Limitations)
	assert.Equal(t, "stop", *response.FinishReason)
	assert.JSONEq(t, `{"request":true}`, string(response.RawRequest))
	assert.JSONEq(t, `{"response":true}`, string(response.RawResponse))
}
