package providerconfigs

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	chatapi "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/api"
	chatapitesting "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/api/testing"
	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/labstack/echo/v4"
	"github.com/mdobak/go-xerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandler_ListProviderConfigs_Admin(t *testing.T) {
	t.Parallel()

	baseURL := "https://api.openai.test"
	mockService := chatapitesting.NewAPI(t)
	mockService.
		On("ListProviderConfigs", mock.Anything).
		Return([]*llmtypes.ProviderConfig{
			{
				ID:          10,
				Provider:    llmtypes.ProviderOpenAI,
				DisplayName: "OpenAI",
				APIKeyEnc:   "encrypted-key",
				BaseURL:     &baseURL,
				IsEnabled:   true,
				Metadata:    json.RawMessage(`{"tier":"prod"}`),
			},
			{
				ID:          11,
				Provider:    llmtypes.ProviderOllama,
				DisplayName: "Ollama",
				IsEnabled:   false,
			},
		}, nil).
		Once()

	e := newTestEcho(mockService, users.RoleAdmin)
	req := httptest.NewRequest(http.MethodGet, "/api/chat/provider-configs", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body, 2)
	assert.Equal(t, float64(10), body[0]["id"])
	assert.Equal(t, "openai", body[0]["provider"])
	assert.Equal(t, "OpenAI", body[0]["display_name"])
	assert.Equal(t, true, body[0]["has_api_key"])
	assert.NotContains(t, body[0], "api_key")
	assert.NotContains(t, body[0], "api_key_enc")
	assert.Equal(t, false, body[1]["has_api_key"])
}

func TestHandler_ListProviderConfigs_NonAdminForbidden(t *testing.T) {
	t.Parallel()

	e := newTestEcho(chatapitesting.NewAPI(t), users.RoleMember)
	req := httptest.NewRequest(http.MethodGet, "/api/chat/provider-configs", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandler_SaveProviderConfig_Admin(t *testing.T) {
	t.Parallel()

	apiKey := "secret-key"
	baseURL := "https://api.openai.test"
	mockService := chatapitesting.NewAPI(t)
	mockService.
		On("SaveProviderConfig", mock.Anything, mock.MatchedBy(func(params chatapi.SaveProviderConfigParams) bool {
			assert.Equal(t, llmtypes.ProviderOpenAI, params.Provider)
			assert.Equal(t, "OpenAI", params.DisplayName)
			require.NotNil(t, params.APIKey)
			assert.Equal(t, apiKey, *params.APIKey)
			require.NotNil(t, params.BaseURL)
			assert.Equal(t, baseURL, *params.BaseURL)
			assert.False(t, params.IsEnabled)
			assert.JSONEq(t, `{"tier":"prod"}`, string(params.Metadata))
			return true
		})).
		Return(&llmtypes.ProviderConfig{
			ID:          10,
			Provider:    llmtypes.ProviderOpenAI,
			DisplayName: "OpenAI",
			APIKeyEnc:   "encrypted-key",
			BaseURL:     &baseURL,
			IsEnabled:   false,
			Metadata:    json.RawMessage(`{"tier":"prod"}`),
		}, nil).
		Once()

	e := newTestEcho(mockService, users.RoleAdmin)
	req := httptest.NewRequest(http.MethodPut, "/api/chat/provider-configs/openai", bytes.NewBufferString(`{"display_name":"OpenAI","api_key":"secret-key","base_url":"https://api.openai.test","is_enabled":false,"metadata":{"tier":"prod"}}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, float64(10), body["id"])
	assert.Equal(t, "openai", body["provider"])
	assert.Equal(t, "OpenAI", body["display_name"])
	assert.Equal(t, false, body["is_enabled"])
	assert.Equal(t, true, body["has_api_key"])
	assert.NotContains(t, body, "api_key")
	assert.NotContains(t, body, "api_key_enc")
}

func TestHandler_SaveProviderConfig_NonAdminForbidden(t *testing.T) {
	t.Parallel()

	e := newTestEcho(chatapitesting.NewAPI(t), users.RoleMember)
	req := httptest.NewRequest(http.MethodPut, "/api/chat/provider-configs/openai", bytes.NewBufferString(`{"display_name":"OpenAI"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandler_SaveProviderConfig_InvalidProvider(t *testing.T) {
	t.Parallel()

	mockService := chatapitesting.NewAPI(t)
	mockService.
		On("SaveProviderConfig", mock.Anything, mock.MatchedBy(func(params chatapi.SaveProviderConfigParams) bool {
			return assert.Equal(t, llmtypes.Provider("unknown"), params.Provider)
		})).
		Return(nil, xerrors.Newf("provider is invalid: %w", chaterrors.ErrInvalidProviderConfig)).
		Once()

	e := newTestEcho(mockService, users.RoleAdmin)
	req := httptest.NewRequest(http.MethodPut, "/api/chat/provider-configs/unknown", bytes.NewBufferString(`{"display_name":"OpenAI"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_SaveProviderConfig_DefaultsIsEnabledTrueAndAllowsOmittedAPIKey(t *testing.T) {
	t.Parallel()

	mockService := chatapitesting.NewAPI(t)
	mockService.
		On("SaveProviderConfig", mock.Anything, mock.MatchedBy(func(params chatapi.SaveProviderConfigParams) bool {
			assert.True(t, params.IsEnabled)
			assert.Nil(t, params.APIKey)
			assert.JSONEq(t, `{}`, string(params.Metadata))
			return true
		})).
		Return(&llmtypes.ProviderConfig{
			ID:          10,
			Provider:    llmtypes.ProviderOpenAI,
			DisplayName: "OpenAI",
			IsEnabled:   true,
			Metadata:    json.RawMessage(`{}`),
		}, nil).
		Once()

	e := newTestEcho(mockService, users.RoleAdmin)
	req := httptest.NewRequest(http.MethodPut, "/api/chat/provider-configs/openai", bytes.NewBufferString(`{"display_name":"OpenAI"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_SaveProviderConfig_CreateWithoutAPIKeyReturnsBadRequest(t *testing.T) {
	t.Parallel()

	mockService := chatapitesting.NewAPI(t)
	mockService.
		On("SaveProviderConfig", mock.Anything, mock.MatchedBy(func(params chatapi.SaveProviderConfigParams) bool {
			return assert.Nil(t, params.APIKey)
		})).
		Return(nil, xerrors.Newf("api key is required: %w", chaterrors.ErrInvalidProviderConfig)).
		Once()

	e := newTestEcho(mockService, users.RoleAdmin)
	req := httptest.NewRequest(http.MethodPut, "/api/chat/provider-configs/openai", bytes.NewBufferString(`{"display_name":"OpenAI"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func newTestEcho(service *chatapitesting.API, role users.Role) *echo.Echo {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(echohandlers.UserContextKey, &users.User{ID: 7, Role: role})
			return next(c)
		}
	})
	New(service, nil).Register(e.Group("/api/chat"))
	return e
}
