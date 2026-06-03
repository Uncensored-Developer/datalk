package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	usererrors "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/errors"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/labstack/echo/v4"
)

const UserContextKey = "user"

type ErrorResponse struct {
	Error string `json:"error"`
}

func User(c echo.Context) (*users.User, error) {
	switch user := c.Get(UserContextKey).(type) {
	case *users.User:
		if user != nil {
			return user, nil
		}
	case users.User:
		return &user, nil
	}

	return nil, echo.NewHTTPError(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
}

func UserID(c echo.Context) (int32, error) {
	user, err := User(c)
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}

func Int64Param(c echo.Context, name string) (int64, error) {
	value, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || value <= 0 {
		return 0, echo.NewHTTPError(http.StatusBadRequest, ErrorResponse{Error: "invalid " + name})
	}

	return value, nil
}

func IntQuery(c echo.Context, name string) (int, error) {
	raw := c.QueryParam(name)
	if raw == "" {
		return 0, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, echo.NewHTTPError(http.StatusBadRequest, ErrorResponse{Error: "invalid " + name})
	}

	return value, nil
}

func Int32Query(c echo.Context, name string) ([]int32, error) {
	raw := c.QueryParam(name)
	if raw == "" {
		return nil, nil
	}

	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value <= 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, ErrorResponse{Error: "invalid " + name})
	}

	return []int32{int32(value)}, nil
}

func Error(c echo.Context, logger *slog.Logger, err error) error {
	if err == nil {
		return nil
	}

	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, usererrors.ErrUnauthorized), errors.Is(err, usererrors.ErrRefreshTokenInvalid):
		status = http.StatusUnauthorized
	case errors.Is(err, usererrors.ErrForbidden), errors.Is(err, usererrors.ErrPasswordChangeRequired):
		status = http.StatusForbidden
	case errors.Is(err, usererrors.ErrUserNotFound):
		status = http.StatusNotFound
	case errors.Is(err, usererrors.ErrSetupUnavailable), errors.Is(err, usererrors.ErrEmailAlreadyExists):
		status = http.StatusConflict
	case errors.Is(err, usererrors.ErrInactiveUser):
		status = http.StatusForbidden
	case errors.Is(err, chaterrors.ErrConversationNotFound), errors.Is(err, chaterrors.ErrMessageNotFound):
		status = http.StatusNotFound
	case errors.Is(err, chaterrors.ErrConnectionAccessDenied):
		status = http.StatusForbidden
	case errors.Is(err, chaterrors.ErrProviderNotAvailable),
		errors.Is(err, chaterrors.ErrInvalidProviderConfig),
		errors.Is(err, chaterrors.ErrModelNotAvailable),
		errors.Is(err, chaterrors.ErrEmbeddedSnapshotNotReady),
		errors.Is(err, chaterrors.ErrInvalidSQL),
		errors.Is(err, chaterrors.ErrPromptTooLarge),
		errors.Is(err, chaterrors.ErrUnsupportedDatabaseKind),
		errors.Is(err, chaterrors.ErrMessageExecutionFailed):
		status = http.StatusBadRequest
	}

	message := err.Error()
	if status >= http.StatusInternalServerError {
		logInternalError(c, logger, err)
		message = "internal server error"
	}

	return echo.NewHTTPError(status, ErrorResponse{Error: message})
}


func logInternalError(c echo.Context, logger *slog.Logger, err error) {
	if logger == nil {
		logger = slog.Default()
	}

	if c == nil || c.Request() == nil {
		logger.Error("internal handler error", slog.Any("err", err))
		return
	}

	req := c.Request()
	logger.Error(
		"internal handler error",
		slog.Any("err", err),
		slog.String("method", req.Method),
		slog.String("path", req.URL.Path),
		slog.String("route", c.Path()),
		slog.String("request_id", req.Header.Get(echo.HeaderXRequestID)),
	)
}
