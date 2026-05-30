package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm/testutil"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ListModels(t *testing.T) {
	client, err := NewClient(&llmtypes.ProviderConfig{
		APIKeyEnc: "test-key",
	}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, http.MethodGet, req.Method)
			require.Equal(t, "/v1/models", req.URL.Path)
			require.Equal(t, "Bearer test-key", req.Header.Get("Authorization"))
			return testutil.JSONResponse(http.StatusOK, `{"data":[{"id":"gpt-5.2"},{"id":"gpt-5-mini"}]}`), nil
		}),
	})
	require.NoError(t, err)

	models, err := client.ListModels(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.Equal(t, "gpt-5.2", models[0].ID)
	assert.Equal(t, "gpt-5-mini", models[1].ID)
	assert.True(t, models[0].IsEnabled)
	assert.True(t, models[0].Capabilities.SupportsStructuredOutput)
	assert.True(t, models[0].Capabilities.SupportsToolCalling)
	assert.True(t, models[0].Capabilities.SupportsSystemInstructions)
}

func TestClient_GenerateSQL(t *testing.T) {
	client, err := NewClient(&llmtypes.ProviderConfig{
		APIKeyEnc: "test-key",
	}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, "/v1/responses", req.URL.Path)
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			var payload generateRequest
			require.NoError(t, json.Unmarshal(body, &payload))
			assert.Equal(t, "gpt-5.2", payload.Model)
			require.Len(t, payload.Input, 4)
			assert.Equal(t, "developer", payload.Input[0].Role)
			assert.Contains(t, payload.Input[0].Content[0].Text, "SQL: SELECT * FROM userss")
			assert.Contains(t, payload.Input[0].Content[0].Text, `Error: relation "userss" does not exist`)
			assert.Equal(t, "user", payload.Input[1].Role)
			assert.Equal(t, "assistant", payload.Input[2].Role)
			assert.Equal(t, "user", payload.Input[3].Role)
			assert.Equal(t, "healthcheck", payload.Input[3].Content[0].Text)

			return testutil.JSONResponse(http.StatusOK, `{
				"status":"completed",
				"output":[{"type":"message","content":[{"type":"output_text","text":"{\"sql\":\"SELECT 1\",\"explanation\":\"Checks connectivity\"}"}]}],
				"usage":{"input_tokens":11,"output_tokens":7,"total_tokens":18}
			}`), nil
		}),
	})
	require.NoError(t, err)

	resp, err := client.GenerateSQL(context.Background(), llmtypes.GenerateSQLRequest{
		Model:      "gpt-5.2",
		UserPrompt: "healthcheck",
		Conversation: llmtypes.ConversationContext{
			History: []llmtypes.ConversationMessage{
				{Role: "user", Content: "previous question"},
				{Role: "assistant", Content: "previous answer"},
			},
		},
		Schema: schematypes.RetrievedSchemaContext{
			Chunks: []schematypes.RetrievedChunk{{ObjectType: "table", ObjectName: "users", Content: "columns: id"}},
		},
		Correction: &llmtypes.SQLCorrectionContext{
			AttemptNumber: 1,
			Attempts: []llmtypes.SQLCorrectionAttempt{
				{SQL: "SELECT * FROM userss", Error: `relation "userss" does not exist`},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "SELECT 1", resp.SQL)
	assert.Equal(t, "Checks connectivity", resp.Explanation)
	require.NotNil(t, resp.Usage)
	assert.Equal(t, 18, *resp.Usage.TotalTokens)
	assert.NotEmpty(t, resp.RawRequest)
	assert.NotEmpty(t, resp.RawResponse)
}

func TestClient_GenerateAnswer(t *testing.T) {
	client, err := NewClient(&llmtypes.ProviderConfig{
		APIKeyEnc: "test-key",
	}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, "/v1/responses", req.URL.Path)
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			var payload generateRequest
			require.NoError(t, json.Unmarshal(body, &payload))
			assert.Equal(t, "gpt-5.2", payload.Model)
			require.Len(t, payload.Input, 2)
			assert.Equal(t, "developer", payload.Input[0].Role)
			systemPrompt := payload.Input[0].Content[0].Text
			assert.Contains(t, systemPrompt, "SELECT email, count(*) AS transactions FROM transactions GROUP BY email")
			assert.Contains(t, systemPrompt, `"row_count": 1`)
			assert.Contains(t, systemPrompt, `"kind": "record"`)
			assert.Contains(t, systemPrompt, `"truncated": false`)
			assert.Equal(t, "Who transacted the most?", payload.Input[1].Content[0].Text)

			return testutil.JSONResponse(http.StatusOK, `{
				"status":"completed",
				"output":[{"type":"message","content":[{"type":"output_text","text":"{\"answer\":\"admin@datalk.app has the most transactions with 200 transactions.\",\"limitations\":[]}"}]}],
				"usage":{"input_tokens":31,"output_tokens":9,"total_tokens":40}
			}`), nil
		}),
	})
	require.NoError(t, err)

	resp, err := client.GenerateAnswer(context.Background(), answerRequest("gpt-5.2"))
	require.NoError(t, err)
	assert.Equal(t, "admin@datalk.app has the most transactions with 200 transactions.", resp.Answer)
	assert.Empty(t, resp.Limitations)
	require.NotNil(t, resp.Usage)
	assert.Equal(t, 40, *resp.Usage.TotalTokens)
	assert.NotEmpty(t, resp.RawRequest)
	assert.NotEmpty(t, resp.RawResponse)
}

func TestClient_GenerateSQL_MalformedPayload(t *testing.T) {
	client, err := NewClient(&llmtypes.ProviderConfig{
		APIKeyEnc: "test-key",
	}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return testutil.JSONResponse(http.StatusOK, `{"status":"completed","output":[{"content":[{"text":"not-json"}]}]}`), nil
		}),
	})
	require.NoError(t, err)

	_, err = client.GenerateSQL(context.Background(), llmtypes.GenerateSQLRequest{Model: "gpt-5.2"})
	require.EqualError(t, err, "failed to decode structured SQL payload: invalid character 'o' in literal null (expecting 'u')")
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

func TestClient_ListModels_ProviderError(t *testing.T) {
	client, err := NewClient(&llmtypes.ProviderConfig{
		APIKeyEnc: "bad-key",
	}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return testutil.JSONResponse(http.StatusUnauthorized, `{"error":{"message":"invalid api key"}}`), nil
		}),
	})
	require.NoError(t, err)

	_, err = client.ListModels(context.Background())
	require.EqualError(t, err, "openai models api failed with status 401: invalid api key")
}
