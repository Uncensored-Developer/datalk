package conversations

import (
	"net/http"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	"github.com/labstack/echo/v4"
)

func (h *Handler) GetConversation(c echo.Context) error {
	userID, err := echohandlers.UserID(c)
	if err != nil {
		return err
	}

	conversationID, err := echohandlers.Int64Param(c, conversationIDParam)
	if err != nil {
		return err
	}

	conversation, err := h.service.GetConversation(c.Request().Context(), userID, conversationID)
	if err != nil {
		return echohandlers.Error(err)
	}

	return c.JSON(http.StatusOK, toConversationResponse(conversation))
}
