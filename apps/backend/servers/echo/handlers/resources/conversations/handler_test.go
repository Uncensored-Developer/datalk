package conversations

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	chatapi "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/api"
	chatapitesting "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/api/testing"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandler_CreateConversation(t *testing.T) {
	t.Parallel()

	title := "Revenue"
	mockService := chatapitesting.NewAPI(t)
	mockService.
		On("CreateConversation", mock.Anything, int32(7), mock.MatchedBy(func(params chattype.CreateConversationParams) bool {
			assert.Equal(t, int32(42), params.ConnectionID)
			require.NotNil(t, params.Title)
			assert.Equal(t, title, *params.Title)
			return true
		})).
		Return(&chattype.Conversation{ID: 10, UserID: 7, ConnectionID: 42, Title: &title}, nil).
		Once()

	e := newTestEcho(mockService)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/conversations", bytes.NewBufferString(`{"connection_id":42,"title":"Revenue"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, float64(10), body["id"])
	assert.Equal(t, float64(7), body["user_id"])
	assert.Equal(t, float64(42), body["connection_id"])
	assert.Equal(t, title, body["title"])
}

func TestHandler_ListMessages(t *testing.T) {
	t.Parallel()

	mockService := chatapitesting.NewAPI(t)
	mockService.
		On("ListMessages", mock.Anything, int32(7), mock.MatchedBy(func(filter chattype.ListMessagesFilter) bool {
			assert.Equal(t, int64(10), filter.ConversationID)
			assert.Equal(t, 20, filter.Limit)
			assert.Equal(t, 40, filter.Offset)
			return true
		})).
		Return([]*chattype.MessageDetails{
			{Message: &chattype.Message{
				ID:              100,
				ConversationID:  10,
				Role:            chattype.MessageRoleAssistant,
				Content:         "Counts users.",
				Status:          chattype.MessageStatusCompleted,
				NaturalResponse: ptr("There are 3 users."),
			}},
		}, nil).
		Once()

	e := newTestEcho(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/chat/conversations/10/messages?limit=20&offset=40", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body, 1)
	message := body[0]["message"].(map[string]any)
	assert.Equal(t, float64(100), message["id"])
	assert.Equal(t, "assistant", message["role"])
	assert.Equal(t, "Counts users.", message["content"])
	assert.Equal(t, "There are 3 users.", message["natural_response"])
}

func TestHandler_SendMessage_MapsDomainError(t *testing.T) {
	t.Parallel()

	mockService := chatapitesting.NewAPI(t)
	mockService.
		On("SendMessage", mock.Anything, mock.MatchedBy(func(params chattype.SendMessageParams) bool {
			assert.Equal(t, int32(7), params.UserID)
			assert.Equal(t, int64(10), params.ConversationID)
			assert.Equal(t, "how many users?", params.Content)
			assert.Equal(t, llmtypes.ProviderOpenAI, params.Provider)
			assert.Equal(t, "gpt-5.2", params.Model)
			assert.False(t, params.RequireNaturalResponse)
			return true
		})).
		Return((*chattype.AssistantTurn)(nil), chaterrors.ErrModelNotAvailable).
		Once()

	e := newTestEcho(mockService)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/conversations/10/messages", bytes.NewBufferString(`{"content":"how many users?","provider":"openai","model":"gpt-5.2"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_SendMessage_PassesRequireNaturalResponse(t *testing.T) {
	t.Parallel()

	naturalResponse := "There are 3 users."
	mockService := chatapitesting.NewAPI(t)
	mockService.
		On("SendMessage", mock.Anything, mock.MatchedBy(func(params chattype.SendMessageParams) bool {
			assert.Equal(t, int32(7), params.UserID)
			assert.Equal(t, int64(10), params.ConversationID)
			assert.Equal(t, "how many users?", params.Content)
			assert.Equal(t, llmtypes.ProviderOpenAI, params.Provider)
			assert.Equal(t, "gpt-5.2", params.Model)
			assert.True(t, params.RequireNaturalResponse)
			return true
		})).
		Return(&chattype.AssistantTurn{
			Conversation: &chattype.Conversation{ID: 10, UserID: 7, ConnectionID: 42},
			UserMessage: &chattype.Message{
				ID:             100,
				ConversationID: 10,
				Role:           chattype.MessageRoleUser,
				Content:        "how many users?",
				Status:         chattype.MessageStatusCompleted,
			},
			AssistantMessage: &chattype.Message{
				ID:              101,
				ConversationID:  10,
				Role:            chattype.MessageRoleAssistant,
				Content:         "Counts users.",
				Provider:        ptr(llmtypes.ProviderOpenAI),
				Model:           ptr("gpt-5.2"),
				Status:          chattype.MessageStatusCompleted,
				NaturalResponse: &naturalResponse,
			},
		}, nil).
		Once()

	e := newTestEcho(mockService)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/conversations/10/messages", bytes.NewBufferString(`{"content":"how many users?","provider":"openai","model":"gpt-5.2","require_natural_response":true}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assistantMessage := body["assistant_message"].(map[string]any)
	assert.Equal(t, "Counts users.", assistantMessage["content"])
	assert.Equal(t, naturalResponse, assistantMessage["natural_response"])
}

func TestHandler_SendMessageStream_WritesProgressAndFinalEvents(t *testing.T) {
	t.Parallel()

	mockService := &streamingTestAPI{
		API: chatapitesting.NewAPI(t),
		turn: &chattype.AssistantTurn{
			Conversation: &chattype.Conversation{ID: 10, UserID: 7, ConnectionID: 42},
			UserMessage: &chattype.Message{
				ID:             100,
				ConversationID: 10,
				Role:           chattype.MessageRoleUser,
				Content:        "how many users?",
				Status:         chattype.MessageStatusCompleted,
			},
			AssistantMessage: &chattype.Message{
				ID:             101,
				ConversationID: 10,
				Role:           chattype.MessageRoleAssistant,
				Content:        "Counts users.",
				Provider:       ptr(llmtypes.ProviderOpenAI),
				Model:          ptr("gpt-5.2"),
				Status:         chattype.MessageStatusCompleted,
			},
		},
	}

	e := newTestEcho(mockService)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/conversations/10/messages/stream", bytes.NewBufferString(`{"content":"how many users?","provider":"openai","model":"gpt-5.2"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/event-stream", rec.Header().Get(echo.HeaderContentType))
	body := rec.Body.String()
	assert.Contains(t, body, "event: progress")
	assert.Contains(t, body, `"stage":"generating_sql"`)
	assert.Contains(t, body, "event: final")
	assert.Contains(t, body, `"assistant_message"`)
	assert.True(t, strings.Contains(body, `"content":"Counts users."`))
}

func TestHandler_SendMessageStream_MapsPreProgressDomainError(t *testing.T) {
	t.Parallel()

	mockService := &streamingTestAPI{
		API: chatapitesting.NewAPI(t),
		err: chaterrors.ErrModelNotAvailable,
	}

	e := newTestEcho(mockService)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/conversations/10/messages/stream", bytes.NewBufferString(`{"content":"how many users?","provider":"openai","model":"missing-model"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Header().Get(echo.HeaderContentType), echo.MIMEApplicationJSON)
	assert.NotContains(t, rec.Body.String(), "event: error")
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["error"], chaterrors.ErrModelNotAvailable.Error())
}

func TestHandler_SendMessage_RejectsEmptyContent(t *testing.T) {
	t.Parallel()

	mockService := chatapitesting.NewAPI(t)
	e := newTestEcho(mockService)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/conversations/10/messages", bytes.NewBufferString(`{"content":"   ","provider":"openai","model":"gpt-5.2"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	mockService.AssertNotCalled(t, "SendMessage", mock.Anything, mock.Anything)
}

type streamingTestAPI struct {
	*chatapitesting.API
	turn *chattype.AssistantTurn
	err  error
}

func (s *streamingTestAPI) SendMessageWithProgress(_ context.Context, _ chattype.SendMessageParams, progress chattype.SendMessageProgressHandler) (*chattype.AssistantTurn, error) {
	if s.err != nil {
		return nil, s.err
	}
	if err := progress(chattype.SendMessageProgress{
		Stage:   chattype.SendMessageProgressGeneratingSQL,
		Attempt: 1,
	}); err != nil {
		return nil, err
	}
	return s.turn, nil
}

func newTestEcho(service chatapi.Client) *echo.Echo {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(echohandlers.UserContextKey, &users.User{ID: 7})
			return next(c)
		}
	})
	New(service, nil).Register(e.Group("/api/chat"))
	return e
}

func ptr[T any](value T) *T {
	return &value
}
