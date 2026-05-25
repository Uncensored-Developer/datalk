package schemas

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	schemasapi "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/api"
	schemasapitesting "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/api/testing"
	usertypes "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandler_RefreshSchemaSnapshot(t *testing.T) {
	t.Parallel()

	mockService := schemasapitesting.NewService(t)
	mockService.
		On("RefreshSchemaSnapshot", mock.Anything, int32(10)).
		Return(nil).
		Once()

	e := newSchemasTestEcho(mockService, &usertypes.User{ID: 7, Role: usertypes.RoleMember})
	req := httptest.NewRequest(http.MethodPost, "/api/connections/10/schema-snapshot/refresh", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, float64(10), body["connection_id"])
	assert.Equal(t, "accepted", body["status"])
}

func TestHandler_RefreshSchemaSnapshot_RejectsUnauthenticated(t *testing.T) {
	t.Parallel()

	mockService := schemasapitesting.NewService(t)
	e := newSchemasTestEcho(mockService, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/connections/10/schema-snapshot/refresh", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	mockService.AssertNotCalled(t, "RefreshSchemaSnapshot", mock.Anything, mock.Anything)
}

func TestHandler_RefreshSchemaSnapshot_InvalidConnectionID(t *testing.T) {
	t.Parallel()

	mockService := schemasapitesting.NewService(t)
	e := newSchemasTestEcho(mockService, &usertypes.User{ID: 7, Role: usertypes.RoleAdmin})
	req := httptest.NewRequest(http.MethodPost, "/api/connections/not-a-number/schema-snapshot/refresh", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	mockService.AssertNotCalled(t, "RefreshSchemaSnapshot", mock.Anything, mock.Anything)
}

func TestHandler_RefreshSchemaSnapshot_RejectsConnectionIDOutsideInt32Range(t *testing.T) {
	t.Parallel()

	mockService := schemasapitesting.NewService(t)
	e := newSchemasTestEcho(mockService, &usertypes.User{ID: 7, Role: usertypes.RoleAdmin})
	req := httptest.NewRequest(http.MethodPost, "/api/connections/4294967297/schema-snapshot/refresh", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	mockService.AssertNotCalled(t, "RefreshSchemaSnapshot", mock.Anything, mock.Anything)
}

func TestHandler_RefreshSchemaSnapshot_MapsServiceError(t *testing.T) {
	t.Parallel()

	mockService := schemasapitesting.NewService(t)
	mockService.
		On("RefreshSchemaSnapshot", mock.Anything, int32(10)).
		Return(errors.New("refresh failed")).
		Once()

	e := newSchemasTestEcho(mockService, &usertypes.User{ID: 7, Role: usertypes.RoleAdmin})
	req := httptest.NewRequest(http.MethodPost, "/api/connections/10/schema-snapshot/refresh", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func newSchemasTestEcho(service schemasapi.Service, user *usertypes.User) *echo.Echo {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(echohandlers.UserContextKey, user)
			return next(c)
		}
	})
	New(schemasapi.New(nil, config.Config{}, service), nil).Register(e.Group("/api"))
	return e
}
