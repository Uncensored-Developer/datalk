package chat

import (
	"encoding/json"
	"strings"
	"testing"

	chatllm "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactSensitiveJSON(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{
		"api_key":"sk-secret",
		"headers":{"authorization":"Bearer token"},
		"dsn":"postgres://user:pass@localhost:5432/app",
		"messages":[{"content":"safe prompt"}],
		"schema":{"foreign_key":"not redacted"}
	}`)

	redacted := redactSensitiveJSON(raw)
	redactedText := string(redacted)

	assert.NotContains(t, redactedText, "sk-secret")
	assert.NotContains(t, redactedText, "Bearer token")
	assert.NotContains(t, redactedText, "postgres://user:pass")
	assert.Contains(t, redactedText, redactedValue)
	assert.Contains(t, redactedText, "safe prompt")
	assert.Contains(t, redactedText, "foreign_key")
}

func TestBuildLLMCall_RedactsPersistedProviderPayloads(t *testing.T) {
	t.Parallel()

	call := buildLLMCall(
		10,
		&chatllm.ResolvedClient{
			ResolvedModel: &chatllm.ResolvedModel{
				ProviderConfig: &llmtypes.ProviderConfig{
					ID:       20,
					Provider: llmtypes.ProviderOpenAI,
				},
			},
		},
		llmtypes.GenerateSQLRequest{Model: "gpt-5.2"},
		&llmtypes.GenerateSQLResponse{
			RawRequest:  json.RawMessage(`{"api_key":"sk-secret","prompt":"safe"}`),
			RawResponse: json.RawMessage(`{"password":"secret","sql":"select 1"}`),
		},
		12,
	)

	require.IsType(t, &chattype.MessageLLMCall{}, call)
	assert.False(t, strings.Contains(string(call.RequestJSON), "sk-secret"))
	assert.False(t, strings.Contains(string(call.ResponseJSON), "secret"))
	assert.Contains(t, string(call.RequestJSON), redactedValue)
	assert.Contains(t, string(call.ResponseJSON), redactedValue)
	assert.Contains(t, string(call.ResponseJSON), "select 1")
}
