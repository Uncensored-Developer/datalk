package providerconfigs

import (
	"encoding/json"
	"net/http"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	chatapi "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/api"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/labstack/echo/v4"
)

type testProviderConfigResponse struct {
	OK         bool `json:"ok"`
	ModelCount int  `json:"model_count"`
}

func (h *Handler) TestProviderConfig(c echo.Context) error {
	if err := requireAdmin(c); err != nil {
		return err
	}

	var req saveProviderConfigRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, echohandlers.ErrorResponse{Error: "invalid request body"})
	}

	metadata := req.Metadata
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}

	result, err := h.service.TestProviderConfig(c.Request().Context(), chatapi.TestProviderConfigParams{
		Provider:    llmtypes.Provider(c.Param("provider")),
		DisplayName: req.DisplayName,
		APIKey:      req.APIKey,
		BaseURL:     req.BaseURL,
		Metadata:    metadata,
	})
	if err != nil {
		return echohandlers.Error(c, h.logger, err)
	}

	return c.JSON(http.StatusOK, testProviderConfigResponse{
		OK:         true,
		ModelCount: result.ModelCount,
	})
}
