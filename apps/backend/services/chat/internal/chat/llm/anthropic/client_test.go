package anthropic

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
			require.Equal(t, http.MethodGet, req.Method)
			require.Equal(t, "/v1/models", req.URL.Path)
			require.Equal(t, "test-key", req.Header.Get("x-api-key"))
			require.Equal(t, anthropicVersion, req.Header.Get("anthropic-version"))
			return testutil.JSONResponse(http.StatusOK, `{"data":[{"id":"claude-sonnet-4-5","display_name":"Claude Sonnet 4.5"}]}`), nil
		}),
	})
	require.NoError(t, err)

	models, err := client.ListModels(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, "claude-sonnet-4-5", models[0].ID)
	assert.Equal(t, "Claude Sonnet 4.5", models[0].DisplayName)
	assert.True(t, models[0].IsEnabled)
	assert.True(t, models[0].Capabilities.SupportsToolCalling)
	assert.True(t, models[0].Capabilities.SupportsStructuredOutput)
	assert.True(t, models[0].Capabilities.SupportsSystemInstructions)
}

func TestClient_GenerateSQL(t *testing.T) {
	client, err := NewClient(&llmtypes.ProviderConfig{
		APIKeyEnc: "test-key",
	}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, "/v1/messages", req.URL.Path)
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			assert.Contains(t, string(body), `"tool_choice":{"type":"tool","name":"propose_sql_query"}`)
			var payload generateRequest
			require.NoError(t, json.Unmarshal(body, &payload))
			require.Len(t, payload.Messages, 3)
			assert.Equal(t, "user", payload.Messages[0].Role)
			assert.Equal(t, "assistant", payload.Messages[1].Role)
			assert.Equal(t, "user", payload.Messages[2].Role)
			assert.Equal(t, "count users", payload.Messages[2].Content)

			return testutil.JSONResponse(http.StatusOK, `{
				"content":[{"type":"tool_use","name":"propose_sql_query","input":{"sql":"SELECT count(*) FROM users","explanation":"Counts users"}}],
				"stop_reason":"tool_use",
				"usage":{"input_tokens":12,"output_tokens":8}
			}`), nil
		}),
	})
	require.NoError(t, err)

	resp, err := client.GenerateSQL(context.Background(), llmtypes.GenerateSQLRequest{
		Model:      "claude-sonnet-4-5",
		UserPrompt: "count users",
		Conversation: llmtypes.ConversationContext{
			History: []llmtypes.ConversationMessage{
				{Role: "user", Content: "previous question"},
				{Role: "assistant", Content: "previous answer"},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "SELECT count(*) FROM users", resp.SQL)
	assert.Equal(t, "Counts users", resp.Explanation)
	require.NotNil(t, resp.Usage)
	assert.Equal(t, 20, *resp.Usage.TotalTokens)
}

func TestClient_GenerateSQL_MalformedResponse(t *testing.T) {
	client, err := NewClient(&llmtypes.ProviderConfig{
		APIKeyEnc: "test-key",
	}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return testutil.JSONResponse(http.StatusOK, `{"content":[{"type":"text","text":"no tool"}]}`), nil
		}),
	})
	require.NoError(t, err)

	_, err = client.GenerateSQL(context.Background(), llmtypes.GenerateSQLRequest{Model: "claude-sonnet-4-5"})
	require.EqualError(t, err, "anthropic generate response did not include tool output")
}

func TestClient_GenerateSQL_ProviderError(t *testing.T) {
	client, err := NewClient(&llmtypes.ProviderConfig{
		APIKeyEnc: "bad-key",
	}, &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return testutil.JSONResponse(http.StatusUnauthorized, `{"error":{"message":"invalid x-api-key"}}`), nil
		}),
	})
	require.NoError(t, err)

	_, err = client.GenerateSQL(context.Background(), llmtypes.GenerateSQLRequest{Model: "claude-sonnet-4-5"})
	require.EqualError(t, err, "anthropic messages api failed with status 401: invalid x-api-key")
}
