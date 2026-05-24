package auth

import (
	"net/http"

	pkgauth "github.com/Uncensored-Developer/datalk/apps/backend/pkg/auth"
	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	usersapi "github.com/Uncensored-Developer/datalk/apps/backend/services/users/api"
	"github.com/labstack/echo/v4"
)

const timeFormat = "2006-01-02T15:04:05Z07:00"

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
}

type userResponse struct {
	ID                 int32  `json:"id"`
	Email              string `json:"email"`
	Name               string `json:"name"`
	Role               string `json:"role"`
	MustChangePassword bool   `json:"must_change_password"`
}

type sessionResponseBody struct {
	User               userResponse  `json:"user"`
	Tokens             tokenResponse `json:"tokens"`
	MustChangePassword bool          `json:"must_change_password"`
}

func sessionResponse(session *pkgauth.Session) sessionResponseBody {
	return sessionResponseBody{
		User: userResponse{
			ID:                 session.User.ID,
			Email:              session.User.Email,
			Name:               session.User.Name,
			Role:               string(session.User.Role),
			MustChangePassword: session.User.MustChangePassword,
		},
		Tokens: tokenResponse{
			AccessToken:  session.Tokens.AccessToken,
			RefreshToken: session.Tokens.RefreshToken,
			ExpiresAt:    session.Tokens.ExpiresAt.Format(timeFormat),
		},
		MustChangePassword: session.Tokens.MustChangePassword,
	}
}

func (h *Handler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, echohandlers.ErrorResponse{Error: "invalid request"})
	}

	session, err := h.users.Login(c.Request().Context(), usersapi.LoginParams{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return echohandlers.Error(c, h.logger, err)
	}
	return c.JSON(http.StatusOK, sessionResponse(session))
}
