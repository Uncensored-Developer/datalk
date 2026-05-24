package connections

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	connectionsapi "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/api"
	connectionsapitesting "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/api/testing"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	usertypes "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandler_CreateConnection(t *testing.T) {
	t.Parallel()

	mockService := connectionsapitesting.NewService(t)
	mockService.
		On("CreateConnection", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			params := reflect.ValueOf(args.Get(1))
			assert.Equal(t, "warehouse", params.FieldByName("Name").String())
			assert.Equal(t, string(connectiontypes.DatabasePostgres), params.FieldByName("Database").String())
			assert.Equal(t, "postgres://example", params.FieldByName("DSN").String())
			assert.Equal(t, int32(7), int32(params.FieldByName("UserID").Int()))
		}).
		Return(&connectiontypes.Connection{
			ID:        10,
			Name:      "warehouse",
			Database:  connectiontypes.DatabasePostgres,
			UserID:    7,
			IsEnabled: true,
		}, nil).
		Once()

	e := newConnectionsTestEcho(mockService, &usertypes.User{ID: 7, Role: usertypes.RoleAdmin})
	req := httptest.NewRequest(http.MethodPost, "/api/connections", bytes.NewBufferString(`{"name":"warehouse","database":"postgres","dsn":"postgres://example"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, float64(10), body["id"])
	assert.Equal(t, "warehouse", body["name"])
	assert.Equal(t, "postgres", body["database"])
	assert.Equal(t, float64(7), body["user_id"])
	assert.Equal(t, true, body["is_enabled"])
}

func TestHandler_CreateConnection_RejectsNonAdmin(t *testing.T) {
	t.Parallel()

	mockService := connectionsapitesting.NewService(t)
	e := newConnectionsTestEcho(mockService, &usertypes.User{ID: 7, Role: usertypes.RoleMember})
	req := httptest.NewRequest(http.MethodPost, "/api/connections", bytes.NewBufferString(`{"name":"warehouse","database":"postgres","dsn":"postgres://example"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	mockService.AssertNotCalled(t, "CreateConnection", mock.Anything, mock.Anything)
}

func TestHandler_CreateAccess(t *testing.T) {
	t.Parallel()

	mockService := connectionsapitesting.NewService(t)
	mockService.
		On("CreateAccess", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			params := reflect.ValueOf(args.Get(1))
			assert.Equal(t, int32(9), int32(params.FieldByName("UserID").Int()))
			assert.Equal(t, int32(10), int32(params.FieldByName("ConnectionID").Int()))
			assert.True(t, params.FieldByName("CanQuery").Bool())
			assert.True(t, params.FieldByName("AllowWrites").Bool())
			assert.False(t, params.FieldByName("CanManage").Bool())
		}).
		Return(&connectiontypes.Access{
			UserID:       9,
			ConnectionID: 10,
			CanQuery:     true,
			AllowWrites:  true,
			CanManage:    false,
		}, nil).
		Once()

	e := newConnectionsTestEcho(mockService, &usertypes.User{ID: 7, Role: usertypes.RoleOwner})
	req := httptest.NewRequest(http.MethodPost, "/api/connections/10/access", bytes.NewBufferString(`{"user_id":9,"can_query":true,"allow_writes":true,"can_manage":false}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, float64(9), body["user_id"])
	assert.Equal(t, float64(10), body["connection_id"])
	assert.Equal(t, true, body["can_query"])
	assert.Equal(t, true, body["allow_writes"])
	assert.Equal(t, false, body["can_manage"])
}

func newConnectionsTestEcho(service connectionsapi.Service, user *usertypes.User) *echo.Echo {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(echohandlers.UserContextKey, user)
			return next(c)
		}
	})
	New(connectionsapi.New(nil, config.Config{}, service), nil).Register(e.Group("/api"))
	return e
}
