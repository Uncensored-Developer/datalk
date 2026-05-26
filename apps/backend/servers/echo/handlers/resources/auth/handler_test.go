package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pkgauth "github.com/Uncensored-Developer/datalk/apps/backend/pkg/auth"
	usersapi "github.com/Uncensored-Developer/datalk/apps/backend/services/users/api"
	usersapitesting "github.com/Uncensored-Developer/datalk/apps/backend/services/users/api/testing"
	usererrors "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/errors"
	usertypes "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandler_Setup(t *testing.T) {
	t.Parallel()

	session := testSession()
	mockUsers := usersapitesting.NewAPI(t)
	mockUsers.
		On("Setup", mock.Anything, usersapi.NewUserParams{
			Name:     "Root",
			Email:    "root@example.com",
			Password: "secret",
		}).
		Return(session, nil).
		Once()

	e := newPublicTestEcho(mockUsers)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/setup", bytes.NewBufferString(`{"name":"Root","email":"root@example.com","password":"secret"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	assertSessionResponse(t, rec.Body.Bytes(), session)
}

func TestHandler_SetupStatus(t *testing.T) {
	t.Parallel()

	mockUsers := usersapitesting.NewAPI(t)
	mockUsers.
		On("SetupStatus", mock.Anything).
		Return(&usersapi.SetupStatus{SetupRequired: true}, nil).
		Once()

	e := newPublicTestEcho(mockUsers)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/setup/status", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]bool
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.True(t, body["setup_required"])
}

func TestHandler_SetupStatus_Error(t *testing.T) {
	t.Parallel()

	mockUsers := usersapitesting.NewAPI(t)
	mockUsers.
		On("SetupStatus", mock.Anything).
		Return(nil, usererrors.ErrSetupUnavailable).
		Once()

	e := newPublicTestEcho(mockUsers)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/setup/status", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestHandler_Setup_RejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		body string
	}{
		{name: "missing name", body: `{"email":"root@example.com","password":"secret"}`},
		{name: "missing email", body: `{"name":"Root","password":"secret"}`},
		{name: "missing password", body: `{"name":"Root","email":"root@example.com"}`},
		{name: "blank name", body: `{"name":" ","email":"root@example.com","password":"secret"}`},
		{name: "blank email", body: `{"name":"Root","email":" ","password":"secret"}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockUsers := usersapitesting.NewAPI(t)
			e := newPublicTestEcho(mockUsers)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/setup", bytes.NewBufferString(tc.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			mockUsers.AssertNotCalled(t, "Setup", mock.Anything, mock.Anything)
		})
	}
}

func TestHandler_Login(t *testing.T) {
	t.Parallel()

	session := testSession()
	mockUsers := usersapitesting.NewAPI(t)
	mockUsers.
		On("Login", mock.Anything, usersapi.LoginParams{Email: "root@example.com", Password: "secret"}).
		Return(session, nil).
		Once()

	e := newPublicTestEcho(mockUsers)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"email":"root@example.com","password":"secret"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assertSessionResponse(t, rec.Body.Bytes(), session)
}

func TestHandler_Refresh_MapsInvalidToken(t *testing.T) {
	t.Parallel()

	mockUsers := usersapitesting.NewAPI(t)
	mockUsers.
		On("Refresh", mock.Anything, "expired-refresh-token").
		Return((*pkgauth.Session)(nil), usererrors.ErrRefreshTokenInvalid).
		Once()

	e := newPublicTestEcho(mockUsers)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewBufferString(`{"refresh_token":"expired-refresh-token"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_Logout(t *testing.T) {
	t.Parallel()

	mockUsers := usersapitesting.NewAPI(t)
	mockUsers.
		On("Logout", mock.Anything, "refresh-token").
		Return(nil).
		Once()

	e := newProtectedTestEcho(mockUsers)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", bytes.NewBufferString(`{"refresh_token":"refresh-token"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func newPublicTestEcho(users usersapi.Client) *echo.Echo {
	e := echo.New()
	New(users, nil).RegisterPublic(e.Group("/api"))
	return e
}

func newProtectedTestEcho(users usersapi.Client) *echo.Echo {
	e := echo.New()
	New(users, nil).RegisterProtected(e.Group("/api"))
	return e
}

func testSession() *pkgauth.Session {
	return &pkgauth.Session{
		User: &usertypes.User{
			ID:                 7,
			Email:              "root@example.com",
			Name:               "Root",
			Role:               usertypes.RoleOwner,
			MustChangePassword: true,
		},
		Tokens: pkgauth.TokenPair{
			AccessToken:        "access-token",
			RefreshToken:       "refresh-token",
			ExpiresAt:          time.Date(2026, 5, 24, 12, 30, 0, 0, time.UTC),
			MustChangePassword: true,
		},
	}
}

func assertSessionResponse(t *testing.T, raw []byte, session *pkgauth.Session) {
	t.Helper()

	var body map[string]any
	require.NoError(t, json.Unmarshal(raw, &body))

	user := body["user"].(map[string]any)
	assert.Equal(t, float64(session.User.ID), user["id"])
	assert.Equal(t, session.User.Email, user["email"])
	assert.Equal(t, session.User.Name, user["name"])
	assert.Equal(t, string(session.User.Role), user["role"])
	assert.Equal(t, session.User.MustChangePassword, user["must_change_password"])

	tokens := body["tokens"].(map[string]any)
	assert.Equal(t, session.Tokens.AccessToken, tokens["access_token"])
	assert.Equal(t, session.Tokens.RefreshToken, tokens["refresh_token"])
	assert.Equal(t, session.Tokens.ExpiresAt.Format(timeFormat), tokens["expires_at"])
	assert.Equal(t, session.Tokens.MustChangePassword, body["must_change_password"])
}
