package conversations

import (
	"net/http"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	"github.com/labstack/echo/v4"
)

func (h *Handler) CreateConversation(c echo.Context) error {
	userID, err := echohandlers.UserID(c)
	if err != nil {
		return err
	}

	var req createConversationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, echohandlers.ErrorResponse{Error: "invalid request body"})
	}

	conversation, err := h.service.CreateConversation(c.Request().Context(), userID, chattype.CreateConversationParams{
		ConnectionID: req.ConnectionID,
		Title:        req.Title,
	})
	if err != nil {
		return echohandlers.Error(err)
	}

	return c.JSON(http.StatusCreated, toConversationResponse(conversation))
}
