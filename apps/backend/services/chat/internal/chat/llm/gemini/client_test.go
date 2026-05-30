package gemini

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
	client, err := NewClient(&llmtypes.ProviderConfig{
		APIKeyEnc: "test-key",
	}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, "/v1beta/models", req.URL.Path)
			require.Equal(t, "key=test-key", req.URL.RawQuery)
			return testutil.JSONResponse(http.StatusOK, `{
				"models":[
					{"name":"models/gemini-2.5-pro","displayName":"Gemini 2.5 Pro","inputTokenLimit":1000,"outputTokenLimit":200,"supportedGenerationMethods":["generateContent"]},
					{"name":"models/embed-only","supportedGenerationMethods":["embedContent"]}
				]
			}`), nil
		}),
	})
	require.NoError(t, err)

	models, err := client.ListModels(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, "gemini-2.5-pro", models[0].ID)
	assert.True(t, models[0].IsEnabled)
	assert.True(t, models[0].Capabilities.SupportsStructuredOutput)
	assert.True(t, models[0].Capabilities.SupportsSystemInstructions)
	require.NotNil(t, models[0].Capabilities.MaxContextTokens)
	assert.Equal(t, 1000, *models[0].Capabilities.MaxContextTokens)
}

func TestClient_GenerateSQL(t *testing.T) {
	client, err := NewClient(&llmtypes.ProviderConfig{
		APIKeyEnc: "test-key",
	}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, "/v1beta/models/gemini-2.5-pro:generateContent", req.URL.Path)
			require.Equal(t, "key=test-key", req.URL.RawQuery)
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			assert.Contains(t, string(body), `"responseMimeType":"application/json"`)
			var payload generateRequest
			require.NoError(t, json.Unmarshal(body, &payload))
			require.Len(t, payload.Contents, 3)
			assert.Equal(t, "user", payload.Contents[0].Role)
			assert.Equal(t, "model", payload.Contents[1].Role)
			assert.Equal(t, "user", payload.Contents[2].Role)
			assert.Equal(t, "list users", payload.Contents[2].Parts[0].Text)
			require.NotEmpty(t, payload.SystemInstruction.Parts)
			assertGeminiSQLSchema(t, payload.GenerationConfig.ResponseSchema)

			return testutil.JSONResponse(http.StatusOK, `{
				"candidates":[{"content":{"parts":[{"text":"{\"sql\":\"SELECT * FROM users\",\"explanation\":\"Lists users\"}"}]},"finishReason":"STOP"}],
				"usageMetadata":{"promptTokenCount":14,"candidatesTokenCount":9,"totalTokenCount":23}
			}`), nil
		}),
	})
	require.NoError(t, err)

	resp, err := client.GenerateSQL(context.Background(), llmtypes.GenerateSQLRequest{
		Model:      "gemini-2.5-pro",
		UserPrompt: "list users",
		Conversation: llmtypes.ConversationContext{
			History: []llmtypes.ConversationMessage{
				{Role: "user", Content: "previous question"},
				{Role: "assistant", Content: "previous answer"},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "SELECT * FROM users", resp.SQL)
	assert.Equal(t, "Lists users", resp.Explanation)
	require.NotNil(t, resp.Usage)
	assert.Equal(t, 23, *resp.Usage.TotalTokens)
}

func TestClient_GenerateAnswer(t *testing.T) {
	client, err := NewClient(&llmtypes.ProviderConfig{
		APIKeyEnc: "test-key",
	}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, "/v1beta/models/gemini-2.5-pro:generateContent", req.URL.Path)
			require.Equal(t, "key=test-key", req.URL.RawQuery)
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			var payload generateRequest
			require.NoError(t, json.Unmarshal(body, &payload))
			require.NotEmpty(t, payload.SystemInstruction.Parts)
			systemPrompt := payload.SystemInstruction.Parts[0].Text
			assert.Contains(t, systemPrompt, "SELECT email, count(*) AS transactions FROM transactions GROUP BY email")
			assert.Contains(t, systemPrompt, `"row_count": 1`)
			assert.Contains(t, systemPrompt, `"kind": "record"`)
			assert.Contains(t, systemPrompt, `"truncated": false`)
			require.Len(t, payload.Contents, 1)
			assert.Equal(t, "user", payload.Contents[0].Role)
			assert.Equal(t, "Who transacted the most?", payload.Contents[0].Parts[0].Text)
			assert.Equal(t, "application/json", payload.GenerationConfig.ResponseMIMEType)
			assertGeminiAnswerSchema(t, payload.GenerationConfig.ResponseSchema)

			return testutil.JSONResponse(http.StatusOK, `{
				"candidates":[{"content":{"parts":[{"text":"{\"answer\":\"admin@datalk.app has the most transactions with 200 transactions.\",\"limitations\":[]}"}]},"finishReason":"STOP"}],
				"usageMetadata":{"promptTokenCount":32,"candidatesTokenCount":8,"totalTokenCount":40}
			}`), nil
		}),
	})
	require.NoError(t, err)

	resp, err := client.GenerateAnswer(context.Background(), answerRequest("gemini-2.5-pro"))
	require.NoError(t, err)
	assert.Equal(t, "admin@datalk.app has the most transactions with 200 transactions.", resp.Answer)
	require.NotNil(t, resp.Usage)
	assert.Equal(t, 40, *resp.Usage.TotalTokens)
	assert.NotEmpty(t, resp.RawRequest)
	assert.NotEmpty(t, resp.RawResponse)
}

func TestClient_GenerateSQL_MalformedResponse(t *testing.T) {
	client, err := NewClient(&llmtypes.ProviderConfig{
		APIKeyEnc: "test-key",
	}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return testutil.JSONResponse(http.StatusOK, `{"candidates":[]}`), nil
		}),
	})
	require.NoError(t, err)

	_, err = client.GenerateSQL(context.Background(), llmtypes.GenerateSQLRequest{Model: "gemini-2.5-pro"})
	require.EqualError(t, err, "gemini generate response did not include candidate text")
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

func assertGeminiSQLSchema(t *testing.T, schema map[string]any) {
	t.Helper()

	assert.Equal(t, "OBJECT", schema["type"])
	assert.NotContains(t, schema, "additionalProperties")

	properties := schema["properties"].(map[string]any)
	assumptions := properties["assumptions"].(map[string]any)
	assert.Equal(t, "ARRAY", assumptions["type"])
	assert.True(t, assumptions["nullable"].(bool))
	assert.NotContains(t, assumptions, "additionalProperties")
	assert.Equal(t, "STRING", assumptions["items"].(map[string]any)["type"])

	confidence := properties["confidence"].(map[string]any)
	assert.Equal(t, "NUMBER", confidence["type"])
	assert.True(t, confidence["nullable"].(bool))
}

func assertGeminiAnswerSchema(t *testing.T, schema map[string]any) {
	t.Helper()

	assert.Equal(t, "OBJECT", schema["type"])
	assert.NotContains(t, schema, "additionalProperties")

	properties := schema["properties"].(map[string]any)
	limitations := properties["limitations"].(map[string]any)
	assert.Equal(t, "ARRAY", limitations["type"])
	assert.True(t, limitations["nullable"].(bool))
	assert.NotContains(t, limitations, "additionalProperties")
	assert.Equal(t, "STRING", limitations["items"].(map[string]any)["type"])
}

func TestClient_ListModels_ProviderError(t *testing.T) {
	client, err := NewClient(&llmtypes.ProviderConfig{
		APIKeyEnc: "bad-key",
	}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return testutil.JSONResponse(http.StatusUnauthorized, `{"error":{"message":"API key not valid"}}`), nil
		}),
	})
	require.NoError(t, err)

	_, err = client.ListModels(context.Background())
	require.EqualError(t, err, "gemini models api failed with status 401: API key not valid")
}
