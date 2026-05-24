package authenticator

import (
	"net/http"
	"strings"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	"github.com/labstack/echo/v4"
)

func Middleware(authenticator Authenticator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := bearerToken(c.Request().Header.Get(echo.HeaderAuthorization))
			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, echohandlers.ErrorResponse{Error: "unauthorized"})
			}

			user, err := authenticator.Authenticate(c.Request().Context(), token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, echohandlers.ErrorResponse{Error: "unauthorized"})
			}
			c.Set(echohandlers.UserContextKey, user)
			if user.MustChangePassword && !passwordChangeAllowed(c) {
				return echo.NewHTTPError(http.StatusForbidden, echohandlers.ErrorResponse{Error: "password change required"})
			}

			return next(c)
		}
	}
}

func bearerToken(header string) string {
	prefix := "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func passwordChangeAllowed(c echo.Context) bool {
	switch c.Path() {
	case "/api/users/me", "/api/users/me/password", "/api/auth/logout":
		return true
	default:
		return false
	}
}
