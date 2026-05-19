package conversations

import (
	"net/http"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	"github.com/labstack/echo/v4"
)

func (h *Handler) ListConversations(c echo.Context) error {
	userID, err := echohandlers.UserID(c)
	if err != nil {
		return err
	}

	connectionID, err := echohandlers.Int32Query(c, "connection_id")
	if err != nil {
		return err
	}
	limit, err := echohandlers.IntQuery(c, "limit")
	if err != nil {
		return err
	}
	offset, err := echohandlers.IntQuery(c, "offset")
	if err != nil {
		return err
	}

	conversations, err := h.service.ListConversations(c.Request().Context(), userID, chattype.ListConversationsFilter{
		ConnectionID: connectionID,
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		return echohandlers.Error(c, h.logger, err)
	}

	return c.JSON(http.StatusOK, toConversationResponses(conversations))
}
