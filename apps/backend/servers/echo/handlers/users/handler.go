package users

import (
	"log/slog"
	"math"
	"net/http"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	usersapi "github.com/Uncensored-Developer/datalk/apps/backend/services/users/api"
	usererrors "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/errors"
	usertypes "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	users  usersapi.Client
	logger *slog.Logger
}

func New(users usersapi.Client, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		users:  users,
		logger: logger.With("resource", "users"),
	}
}

func (h *Handler) Register(group *echo.Group) {
	group.GET("", h.ListUsers)
	group.PUT("/:user_id", h.UpdateUser)
	group.GET("/me", h.Me)
	group.POST("/me/password", h.ChangePassword)
	group.POST("", h.CreateUser)
}

func (h *Handler) ListUsers(c echo.Context) error {
	user, err := echohandlers.User(c)
	if err != nil {
		return err
	}
	if !user.IsAdmin() {
		return echohandlers.Error(c, h.logger, usererrors.ErrForbidden)
	}

	users, err := h.users.ListUsers(c.Request().Context())
	if err != nil {
		return echohandlers.Error(c, h.logger, err)
	}
	return c.JSON(http.StatusOK, toUserResponses(users))
}

func (h *Handler) Me(c echo.Context) error {
	user, err := echohandlers.User(c)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toUserResponse(user))
}

func (h *Handler) CreateUser(c echo.Context) error {
	user, err := echohandlers.User(c)
	if err != nil {
		return err
	}
	if !user.IsAdmin() {
		return echohandlers.Error(c, h.logger, usererrors.ErrForbidden)
	}

	var req createUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, echohandlers.ErrorResponse{Error: "invalid request"})
	}
	role := usertypes.Role(req.Role)
	if role == "" {
		role = usertypes.RoleMember
	}

	createdUser, err := h.users.CreateUser(c.Request().Context(), usersapi.NewUserParams{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Role:     role,
	})
	if err != nil {
		return echohandlers.Error(c, h.logger, err)
	}
	return c.JSON(http.StatusCreated, toUserResponse(createdUser))
}

func (h *Handler) UpdateUser(c echo.Context) error {
	user, err := echohandlers.User(c)
	if err != nil {
		return err
	}
	if !user.IsAdmin() {
		return echohandlers.Error(c, h.logger, usererrors.ErrForbidden)
	}

	userID, err := userID(c)
	if err != nil {
		return err
	}

	var req updateUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, echohandlers.ErrorResponse{Error: "invalid request"})
	}

	var role *usertypes.Role
	if req.Role != nil {
		value := usertypes.Role(*req.Role)
		role = &value
	}

	updatedUser, err := h.users.UpdateUser(c.Request().Context(), usersapi.UpdateUserParams{
		ID:       userID,
		Name:     req.Name,
		Role:     role,
		IsActive: req.IsActive,
	})
	if err != nil {
		return echohandlers.Error(c, h.logger, err)
	}
	return c.JSON(http.StatusOK, toUserResponse(updatedUser))
}

func (h *Handler) ChangePassword(c echo.Context) error {
	user, err := echohandlers.User(c)
	if err != nil {
		return err
	}

	var req changePasswordRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, echohandlers.ErrorResponse{Error: "invalid request"})
	}

	updatedUser, err := h.users.ChangePassword(c.Request().Context(), usersapi.ChangePasswordParams{
		UserID:          user.ID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		return echohandlers.Error(c, h.logger, err)
	}
	return c.JSON(http.StatusOK, toUserResponse(updatedUser))
}

type createUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type updateUserRequest struct {
	Name     *string `json:"name"`
	Role     *string `json:"role"`
	IsActive *bool   `json:"is_active"`
}

type userResponse struct {
	ID                 int32  `json:"id"`
	Email              string `json:"email"`
	Name               string `json:"name"`
	Role               string `json:"role"`
	IsActive           bool   `json:"is_active"`
	MustChangePassword bool   `json:"must_change_password"`
}

func toUserResponses(users []*usertypes.User) []userResponse {
	out := make([]userResponse, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		out = append(out, toUserResponse(user))
	}
	return out
}

func toUserResponse(user *usertypes.User) userResponse {
	return userResponse{
		ID:                 user.ID,
		Email:              user.Email,
		Name:               user.Name,
		Role:               string(user.Role),
		IsActive:           user.IsActive,
		MustChangePassword: user.MustChangePassword,
	}
}

func userID(c echo.Context) (int32, error) {
	rawID, err := echohandlers.Int64Param(c, "user_id")
	if err != nil {
		return 0, err
	}
	if rawID > math.MaxInt32 {
		return 0, echo.NewHTTPError(http.StatusBadRequest, echohandlers.ErrorResponse{Error: "invalid user_id"})
	}
	return int32(rawID), nil
}
