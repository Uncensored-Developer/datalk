package staticweb

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterFSServesIndexAtRoot(t *testing.T) {
	e := newTestEcho()

	rec := performRequest(e, http.MethodGet, "/")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, rec.Body.String(), `<div id="root"></div>`)
}

func TestRegisterFSServesIndexForNestedSPAPath(t *testing.T) {
	e := newTestEcho()

	rec := performRequest(e, http.MethodGet, "/chat/conversations/123")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, rec.Body.String(), `<div id="root"></div>`)
}

func TestRegisterFSServesStaticAsset(t *testing.T) {
	e := newTestEcho()

	rec := performRequest(e, http.MethodGet, "/assets/app.txt")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/plain")
	assert.Equal(t, "asset-content", rec.Body.String())
}

func TestRegisterFSDoesNotHandleAPIPaths(t *testing.T) {
	e := newTestEcho()

	rec := performRequest(e, http.MethodGet, "/api/unknown")

	require.Equal(t, http.StatusNotFound, rec.Code)
	assert.NotContains(t, rec.Body.String(), `<div id="root"></div>`)
}

func newTestEcho() *echo.Echo {
	e := echo.New()
	RegisterFS(e, fstest.MapFS{
		"index.html": {
			Data: []byte(`<!doctype html><html><body><div id="root"></div></body></html>`),
		},
		"assets/app.txt": {
			Data: []byte("asset-content"),
		},
	})
	return e
}

func performRequest(e *echo.Echo, method string, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}
