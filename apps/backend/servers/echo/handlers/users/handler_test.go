package users

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	usersapi "github.com/Uncensored-Developer/datalk/apps/backend/services/users/api"
	usersapitesting "github.com/Uncensored-Developer/datalk/apps/backend/services/users/api/testing"
	usertypes "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandler_ListUsers_Admin(t *testing.T) {
	t.Parallel()

	mockUsers := usersapitesting.NewAPI(t)
	mockUsers.
		On("ListUsers", mock.Anything).
		Return([]*usertypes.User{
			{ID: 1, Email: "owner@example.com", Name: "Owner", Role: usertypes.RoleOwner, IsActive: true},
			{ID: 2, Email: "member@example.com", Name: "Member", Role: usertypes.RoleMember, IsActive: true},
		}, nil).
		Once()

	e := newUsersTestEcho(mockUsers, &usertypes.User{ID: 1, Role: usertypes.RoleAdmin})
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body, 2)
	assert.Equal(t, "owner@example.com", body[0]["email"])
	assert.Equal(t, "member@example.com", body[1]["email"])
}

func TestHandler_ListUsers_RejectsNonAdmin(t *testing.T) {
	t.Parallel()

	mockUsers := usersapitesting.NewAPI(t)
	e := newUsersTestEcho(mockUsers, &usertypes.User{ID: 2, Role: usertypes.RoleMember})
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	mockUsers.AssertNotCalled(t, "ListUsers", mock.Anything)
}

func TestHandler_UpdateUser_Admin(t *testing.T) {
	t.Parallel()

	mockUsers := usersapitesting.NewAPI(t)
	mockUsers.
		On("UpdateUser", mock.Anything, mock.MatchedBy(func(params usersapi.UpdateUserParams) bool {
			return params.ID == 2 &&
				params.Name != nil && *params.Name == "Updated Member" &&
				params.Role != nil && *params.Role == usertypes.RoleAdmin &&
				params.IsActive != nil && !*params.IsActive
		})).
		Return(&usertypes.User{
			ID:       2,
			Email:    "member@example.com",
			Name:     "Updated Member",
			Role:     usertypes.RoleAdmin,
			IsActive: false,
		}, nil).
		Once()

	e := newUsersTestEcho(mockUsers, &usertypes.User{ID: 1, Role: usertypes.RoleOwner})
	req := httptest.NewRequest(http.MethodPut, "/api/users/2", bytes.NewBufferString(`{
		"name":"Updated Member",
		"role":"admin",
		"is_active":false
	}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, float64(2), body["id"])
	assert.Equal(t, "member@example.com", body["email"])
	assert.Equal(t, "admin", body["role"])
	assert.Equal(t, false, body["is_active"])
}

func TestHandler_UpdateUser_RejectsNonAdmin(t *testing.T) {
	t.Parallel()

	mockUsers := usersapitesting.NewAPI(t)
	e := newUsersTestEcho(mockUsers, &usertypes.User{ID: 2, Role: usertypes.RoleMember})
	req := httptest.NewRequest(http.MethodPut, "/api/users/2", bytes.NewBufferString(`{"name":"Blocked"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	mockUsers.AssertNotCalled(t, "UpdateUser", mock.Anything, mock.Anything)
}

func newUsersTestEcho(users usersapi.Client, user *usertypes.User) *echo.Echo {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(echohandlers.UserContextKey, user)
			return next(c)
		}
	})
	New(users, nil).Register(e.Group("/api/users"))
	return e
}
