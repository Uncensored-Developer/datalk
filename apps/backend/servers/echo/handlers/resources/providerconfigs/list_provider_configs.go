package providerconfigs

import (
	"net/http"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/labstack/echo/v4"
)

func (h *Handler) ListProviderConfigs(c echo.Context) error {
	if err := requireAdmin(c); err != nil {
		return err
	}

	configs, err := h.service.ListProviderConfigs(c.Request().Context())
	if err != nil {
		return echohandlers.Error(c, h.logger, err)
	}

	return c.JSON(http.StatusOK, toProviderConfigResponses(configs))
}

func toProviderConfigResponses(configs []*llmtypes.ProviderConfig) []providerConfigResponse {
	out := make([]providerConfigResponse, 0, len(configs))
	for _, config := range configs {
		if config == nil {
			continue
		}
		out = append(out, toProviderConfigResponse(config))
	}

	return out
}
