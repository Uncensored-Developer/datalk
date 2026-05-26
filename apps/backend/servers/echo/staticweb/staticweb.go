package staticweb

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

//go:embed dist
var embeddedDist embed.FS

// Register mounts the embedded frontend assets on the Echo router.
func Register(e *echo.Echo) {
	dist, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		panic("staticweb: embedded dist directory is unavailable: " + err.Error())
	}
	RegisterFS(e, dist)
}

// RegisterFS mounts a frontend filesystem on the Echo router.
func RegisterFS(e *echo.Echo, files fs.FS) {
	handler := echo.WrapHandler(NewHandler(files))
	e.GET("/*", handler)
	e.HEAD("/*", handler)
}

// NewHandler serves static assets directly and falls back to index.html for
// non-API routes so the frontend router can handle deep links.
func NewHandler(files fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(files))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAPIPath(r.URL.Path) {
			http.NotFound(w, r)
			return
		}

		filePath := cleanRequestPath(r.URL.Path)
		if filePath != "" && regularFileExists(files, filePath) {
			fileServer.ServeHTTP(w, r)
			return
		}

		serveIndex(w, r, files)
	})
}

func isAPIPath(requestPath string) bool {
	return requestPath == "/api" || strings.HasPrefix(requestPath, "/api/")
}

func cleanRequestPath(requestPath string) string {
	cleaned := strings.TrimPrefix(path.Clean("/"+requestPath), "/")
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func regularFileExists(files fs.FS, name string) bool {
	file, err := files.Open(name)
	if err != nil {
		return false
	}
	defer file.Close()

	info, err := file.Stat()
	return err == nil && !info.IsDir()
}

func serveIndex(w http.ResponseWriter, r *http.Request, files fs.FS) {
	index, err := fs.ReadFile(files, "index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(index))
}
