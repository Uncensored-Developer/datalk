package chat

import (
	"context"
	"encoding/json"
	"strings"

	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/mdobak/go-xerrors"
)

type SaveProviderConfigParams struct {
	Provider    llm.Provider
	DisplayName string
	APIKey      *string
	BaseURL     *string
	IsEnabled   bool
	Metadata    json.RawMessage
}

func (s *Service) ListProviderConfigs(ctx context.Context) ([]*llm.ProviderConfig, error) {
	configs, err := s.storage.ListProviderConfigs(ctx, chatstorage.ProviderConfigsFilter{})
	if err != nil {
		return nil, xerrors.Newf("failed to list provider configs: %w", err)
	}

	return configs, nil
}

func (s *Service) SaveProviderConfig(ctx context.Context, params SaveProviderConfigParams) (*llm.ProviderConfig, error) {
	if err := validateSaveProviderConfigParams(params); err != nil {
		return nil, err
	}

	existing, err := s.providerConfigByProvider(ctx, params.Provider)
	if err != nil {
		return nil, err
	}

	apiKeyEnc := ""
	apiKey := optionalAPIKey(params.APIKey)
	if apiKey != nil {
		apiKeyEnc, err = s.cipher.Encrypt(*apiKey)
		if err != nil {
			return nil, xerrors.Newf("failed to encrypt provider api key: %w", err)
		}
	} else if existing != nil {
		apiKeyEnc = existing.APIKeyEnc
	} else {
		if providerRequiresAPIKey(params.Provider) {
			return nil, xerrors.Newf("api key is required: %w", chaterrors.ErrInvalidProviderConfig)
		}
	}

	config := &llm.ProviderConfig{
		Provider:    params.Provider,
		DisplayName: strings.TrimSpace(params.DisplayName),
		APIKeyEnc:   apiKeyEnc,
		BaseURL:     trimOptionalString(params.BaseURL),
		IsEnabled:   params.IsEnabled,
		Metadata:    params.Metadata,
	}

	if err := s.storage.UpsertProviderConfig(ctx, config); err != nil {
		return nil, xerrors.Newf("failed to save provider config: %w", err)
	}

	return config, nil
}

func (s *Service) providerConfigByProvider(ctx context.Context, provider llm.Provider) (*llm.ProviderConfig, error) {
	configs, err := s.storage.ListProviderConfigs(ctx, chatstorage.ProviderConfigsFilter{
		Provider: []llm.Provider{provider},
	})
	if err != nil {
		return nil, xerrors.Newf("failed to fetch provider config: %w", err)
	}
	if len(configs) == 0 {
		return nil, nil
	}
	if len(configs) > 1 {
		return nil, xerrors.Newf("multiple provider configs found for provider %s", provider)
	}

	return configs[0], nil
}

func validateSaveProviderConfigParams(params SaveProviderConfigParams) error {
	if !isKnownProvider(params.Provider) {
		return xerrors.Newf("provider is invalid: %w", chaterrors.ErrInvalidProviderConfig)
	}
	if strings.TrimSpace(params.DisplayName) == "" {
		return xerrors.Newf("display name is required: %w", chaterrors.ErrInvalidProviderConfig)
	}

	return nil
}

func isKnownProvider(provider llm.Provider) bool {
	switch provider {
	case llm.ProviderOpenAI, llm.ProviderAnthropic, llm.ProviderGemini, llm.ProviderOllama:
		return true
	default:
		return false
	}
}

func providerRequiresAPIKey(provider llm.Provider) bool {
	switch provider {
	case llm.ProviderOpenAI, llm.ProviderAnthropic, llm.ProviderGemini:
		return true
	default:
		return false
	}
}

func optionalAPIKey(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}
