package ollama

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm/testutil"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ListModels(t *testing.T) {
	client, err := NewClient(&llmtypes.ProviderConfig{}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, "/api/tags", req.URL.Path)
			return testutil.JSONResponse(http.StatusOK, `{"models":[{"model":"qwen2.5-coder:7b"},{"name":"llama3.1"}]}`), nil
		}),
	})
	require.NoError(t, err)

	models, err := client.ListModels(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.Equal(t, "qwen2.5-coder:7b", models[0].ID)
	assert.Equal(t, "llama3.1", models[1].ID)
	assert.True(t, models[0].IsEnabled)
	assert.True(t, models[0].Capabilities.SupportsStructuredOutput)
	assert.True(t, models[0].Capabilities.SupportsSystemInstructions)
}

func TestClient_GenerateSQL(t *testing.T) {
	client, err := NewClient(&llmtypes.ProviderConfig{}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, "/api/chat", req.URL.Path)
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			var payload chatRequest
			require.NoError(t, json.Unmarshal(body, &payload))
			require.Len(t, payload.Messages, 4)
			assert.Equal(t, "system", payload.Messages[0].Role)
			assert.Equal(t, "user", payload.Messages[1].Role)
			assert.Equal(t, "assistant", payload.Messages[2].Role)
			assert.Equal(t, "user", payload.Messages[3].Role)
			assert.Equal(t, "show current time", payload.Messages[3].Content)

			return testutil.JSONResponse(http.StatusOK, `{
				"message":{"role":"assistant","content":"{\"sql\":\"SELECT now()\",\"explanation\":\"Returns the current timestamp\"}"},
				"done_reason":"stop",
				"prompt_eval_count":15,
				"eval_count":10
			}`), nil
		}),
	})
	require.NoError(t, err)

	resp, err := client.GenerateSQL(context.Background(), llmtypes.GenerateSQLRequest{
		Model:      "qwen2.5-coder:7b",
		UserPrompt: "show current time",
		Conversation: llmtypes.ConversationContext{
			History: []llmtypes.ConversationMessage{
				{Role: "user", Content: "previous question"},
				{Role: "assistant", Content: "previous answer"},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "SELECT now()", resp.SQL)
	assert.Equal(t, "Returns the current timestamp", resp.Explanation)
	require.NotNil(t, resp.Usage)
	assert.Equal(t, 25, *resp.Usage.TotalTokens)
}

func TestClient_GenerateAnswer(t *testing.T) {
	client, err := NewClient(&llmtypes.ProviderConfig{}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, "/api/chat", req.URL.Path)
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			var payload chatRequest
			require.NoError(t, json.Unmarshal(body, &payload))
			assert.Equal(t, "qwen2.5-coder:7b", payload.Model)
			require.Len(t, payload.Messages, 2)
			assert.Equal(t, "system", payload.Messages[0].Role)
			assert.Contains(t, payload.Messages[0].Content, "SELECT email, count(*) AS transactions FROM transactions GROUP BY email")
			assert.Contains(t, payload.Messages[0].Content, `"row_count": 1`)
			assert.Contains(t, payload.Messages[0].Content, `"kind": "record"`)
			assert.Contains(t, payload.Messages[0].Content, `"truncated": false`)
			assert.Equal(t, "user", payload.Messages[1].Role)
			assert.Equal(t, "Who transacted the most?", payload.Messages[1].Content)

			return testutil.JSONResponse(http.StatusOK, `{
				"message":{"role":"assistant","content":"{\"answer\":\"admin@datalk.app has the most transactions with 200 transactions.\",\"limitations\":[]}"},
				"done_reason":"stop",
				"prompt_eval_count":30,
				"eval_count":10
			}`), nil
		}),
	})
	require.NoError(t, err)

	resp, err := client.GenerateAnswer(context.Background(), answerRequest("qwen2.5-coder:7b"))
	require.NoError(t, err)
	assert.Equal(t, "admin@datalk.app has the most transactions with 200 transactions.", resp.Answer)
	require.NotNil(t, resp.Usage)
	assert.Equal(t, 40, *resp.Usage.TotalTokens)
	assert.NotEmpty(t, resp.RawRequest)
	assert.NotEmpty(t, resp.RawResponse)
}

func TestClient_GenerateSQL_MalformedResponse(t *testing.T) {
	client, err := NewClient(&llmtypes.ProviderConfig{}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return testutil.JSONResponse(http.StatusOK, `{"message":{"content":""}}`), nil
		}),
	})
	require.NoError(t, err)

	_, err = client.GenerateSQL(context.Background(), llmtypes.GenerateSQLRequest{Model: "qwen2.5-coder:7b"})
	require.EqualError(t, err, "ollama chat response did not include structured output")
}

func answerRequest(model string) llmtypes.GenerateAnswerRequest {
	return llmtypes.GenerateAnswerRequest{
		Model:        model,
		UserPrompt:   "Who transacted the most?",
		GeneratedSQL: "SELECT email, count(*) AS transactions FROM transactions GROUP BY email",
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
			Kind:      "record",
		},
	}
}

func TestClient_GenerateSQL_ProviderError(t *testing.T) {
	client, err := NewClient(&llmtypes.ProviderConfig{}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return testutil.JSONResponse(http.StatusInternalServerError, `{"error":"model not found"}`), nil
		}),
	})
	require.NoError(t, err)

	_, err = client.GenerateSQL(context.Background(), llmtypes.GenerateSQLRequest{Model: "missing"})
	require.EqualError(t, err, "ollama chat api failed with status 500: model not found")
}
