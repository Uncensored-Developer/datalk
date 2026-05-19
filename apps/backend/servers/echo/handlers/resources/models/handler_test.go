package models

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	chatapitesting "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/api/testing"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandler_ListModels(t *testing.T) {
	t.Parallel()

	maxContextTokens := 128000
	mockService := chatapitesting.NewAPI(t)
	mockService.
		On("ListAvailableModels", mock.Anything).
		Return([]llmtypes.Model{
			{
				ID:          "openai:gpt-5.2",
				Provider:    llmtypes.ProviderOpenAI,
				DisplayName: "GPT 5.2",
				IsEnabled:   true,
				Capabilities: llmtypes.ModelCapabilities{
					SupportsStructuredOutput: true,
					MaxContextTokens:         &maxContextTokens,
				},
			},
		}, nil).
		Once()

	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(echohandlers.UserContextKey, &users.User{ID: 7})
			return next(c)
		}
	})
	New(mockService, nil).Register(e.Group("/api/chat"))

	req := httptest.NewRequest(http.MethodGet, "/api/chat/models", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body, 1)
	assert.Equal(t, "openai:gpt-5.2", body[0]["id"])
	assert.Equal(t, "openai", body[0]["provider"])
	assert.Equal(t, "GPT 5.2", body[0]["display_name"])
	capabilities := body[0]["capabilities"].(map[string]any)
	assert.Equal(t, true, capabilities["supports_structured_output"])
	assert.Equal(t, float64(maxContextTokens), capabilities["max_context_tokens"])
}
