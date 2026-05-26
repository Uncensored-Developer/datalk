package conversations

import (
	"net/http"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	"github.com/labstack/echo/v4"
)

func (h *Handler) DeleteConversation(c echo.Context) error {
	userID, err := echohandlers.UserID(c)
	if err != nil {
		return err
	}

	conversationID, err := echohandlers.Int64Param(c, conversationIDParam)
	if err != nil {
		return err
	}

	if err := h.service.DeleteConversation(c.Request().Context(), userID, conversationID); err != nil {
		return echohandlers.Error(c, h.logger, err)
	}

	return c.NoContent(http.StatusNoContent)
}
