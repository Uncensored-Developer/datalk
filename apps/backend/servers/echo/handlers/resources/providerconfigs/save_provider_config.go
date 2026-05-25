package providerconfigs

import (
	"encoding/json"
	"net/http"
	"time"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	chatapi "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/api"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/labstack/echo/v4"
)

type saveProviderConfigRequest struct {
	DisplayName string          `json:"display_name"`
	APIKey      *string         `json:"api_key"`
	BaseURL     *string         `json:"base_url"`
	IsEnabled   *bool           `json:"is_enabled"`
	Metadata    json.RawMessage `json:"metadata"`
}

type providerConfigResponse struct {
	ID          int64             `json:"id"`
	Provider    llmtypes.Provider `json:"provider"`
	DisplayName string            `json:"display_name"`
	BaseURL     *string           `json:"base_url"`
	IsEnabled   bool              `json:"is_enabled"`
	HasAPIKey   bool              `json:"has_api_key"`
	Metadata    json.RawMessage   `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

func (h *Handler) SaveProviderConfig(c echo.Context) error {
	if err := requireAdmin(c); err != nil {
		return err
	}

	var req saveProviderConfigRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, echohandlers.ErrorResponse{Error: "invalid request body"})
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	metadata := req.Metadata
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}

	config, err := h.service.SaveProviderConfig(c.Request().Context(), chatapi.SaveProviderConfigParams{
		Provider:    llmtypes.Provider(c.Param("provider")),
		DisplayName: req.DisplayName,
		APIKey:      req.APIKey,
		BaseURL:     req.BaseURL,
		IsEnabled:   isEnabled,
		Metadata:    metadata,
	})
	if err != nil {
		return echohandlers.Error(c, h.logger, err)
	}

	return c.JSON(http.StatusOK, toProviderConfigResponse(config))
}

func toProviderConfigResponse(config *llmtypes.ProviderConfig) providerConfigResponse {
	metadata := config.Metadata
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}

	return providerConfigResponse{
		ID:          config.ID,
		Provider:    config.Provider,
		DisplayName: config.DisplayName,
		BaseURL:     config.BaseURL,
		IsEnabled:   config.IsEnabled,
		HasAPIKey:   config.APIKeyEnc != "",
		Metadata:    metadata,
		CreatedAt:   config.CreatedAt,
		UpdatedAt:   config.UpdatedAt,
	}
}

func requireAdmin(c echo.Context) error {
	user, err := echohandlers.User(c)
	if err != nil {
		return err
	}
	if !user.IsAdmin() {
		return echo.NewHTTPError(http.StatusForbidden, echohandlers.ErrorResponse{Error: "forbidden"})
	}

	return nil
}
