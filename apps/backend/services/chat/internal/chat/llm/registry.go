package llm

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/secrets"
	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/gotidy/ptr"
	"github.com/mdobak/go-xerrors"
)

type Registry struct {
	storage   chatstorage.Storage
	factories map[llmtypes.Provider]ClientFactory
	cipher    secrets.Cipher
}

type ResolvedModel struct {
	ProviderConfig   *llmtypes.ProviderConfig
	Model            llmtypes.Model
	ProviderModelID  string
	QualifiedModelID string
}

type ResolvedClient struct {
	*ResolvedModel
	Client Client
}

//go:generate go tool with-modfile mockery --name ClientResolver --outpkg testing --output ./testing --filename generated__client_resolver_mocks.go
type ClientResolver interface {
	ResolveClient(ctx context.Context, provider llmtypes.Provider, modelID string) (*ResolvedClient, error)
}

type ProviderTester interface {
	TestProviderConfig(ctx context.Context, config *llmtypes.ProviderConfig) ([]llmtypes.Model, error)
}

func NewRegistry(storage chatstorage.Storage, factories map[llmtypes.Provider]ClientFactory, ciphers ...secrets.Cipher) *Registry {
	cipher := secrets.Cipher(secrets.PlaintextCipher{})
	if len(ciphers) > 0 && ciphers[0] != nil {
		cipher = ciphers[0]
	}

	return &Registry{
		storage:   storage,
		factories: maps.Clone(factories),
		cipher:    cipher,
	}
}

func (r *Registry) ListAvailableModels(ctx context.Context) ([]llmtypes.Model, error) {
	configsByProvider, err := r.listEnabledProviderConfigs(ctx)
	if err != nil {
		return nil, err
	}

	models := make([]llmtypes.Model, 0)
	for _, provider := range sortedProviders(configsByProvider) {
		config := configsByProvider[provider]
		normalizedModels, err := r.listProviderModels(ctx, config)
		if err != nil {
			return nil, err
		}

		models = append(models, normalizedModels...)
	}

	slices.SortFunc(models, func(a, b llmtypes.Model) int {
		return cmp.Compare(a.ID, b.ID)
	})

	return models, nil
}

func (r *Registry) TestProviderConfig(ctx context.Context, config *llmtypes.ProviderConfig) ([]llmtypes.Model, error) {
	if config == nil {
		return nil, xerrors.New("provider config cannot be nil")
	}

	client, err := r.clientForPlainConfig(config)
	if err != nil {
		return nil, err
	}

	providerModels, err := client.ListModels(ctx)
	if err != nil {
		return nil, xerrors.Newf("failed to list models for provider %s: %w", config.Provider, err)
	}

	return normalizeProviderModels(config.Provider, providerModels)
}

func (r *Registry) ResolveQualifiedModel(ctx context.Context, qualifiedModelID string) (*ResolvedModel, error) {
	provider, modelID, err := ParseQualifiedModelID(qualifiedModelID)
	if err != nil {
		return nil, err
	}

	return r.ResolveModel(ctx, provider, modelID)
}

func (r *Registry) ResolveModel(ctx context.Context, provider llmtypes.Provider, modelID string) (*ResolvedModel, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, xerrors.Newf("model id is required: %w", chaterrors.ErrModelNotAvailable)
	}
	providerModelID := modelID
	if strings.Contains(modelID, ":") {
		parsedProvider, parsedModelID, err := ParseQualifiedModelID(modelID)
		if err != nil {
			return nil, err
		}
		if parsedProvider != provider {
			return nil, xerrors.Newf("model %s is not available for provider %s: %w", modelID, provider, chaterrors.ErrModelNotAvailable)
		}
		providerModelID = parsedModelID
	}

	configsByProvider, err := r.listEnabledProviderConfigs(ctx)
	if err != nil {
		return nil, err
	}

	config, ok := configsByProvider[provider]
	if !ok {
		return nil, xerrors.Newf("provider %s is not enabled: %w", provider, chaterrors.ErrProviderNotAvailable)
	}

	normalizedModels, err := r.listProviderModels(ctx, config)
	if err != nil {
		return nil, err
	}

	qualifiedModelID := QualifiedModelID(provider, providerModelID)
	for _, model := range normalizedModels {
		if model.ID != qualifiedModelID {
			continue
		}

		return &ResolvedModel{
			ProviderConfig:   config,
			Model:            model,
			ProviderModelID:  providerModelID,
			QualifiedModelID: qualifiedModelID,
		}, nil
	}

	return nil, xerrors.Newf("model %s is not available for provider %s: %w", modelID, provider, chaterrors.ErrModelNotAvailable)
}

func (r *Registry) ResolveClient(ctx context.Context, provider llmtypes.Provider, modelID string) (*ResolvedClient, error) {
	resolved, err := r.ResolveModel(ctx, provider, modelID)
	if err != nil {
		return nil, err
	}

	client, err := r.clientForConfig(resolved.ProviderConfig)
	if err != nil {
		return nil, err
	}

	return &ResolvedClient{
		ResolvedModel: resolved,
		Client:        client,
	}, nil
}

func QualifiedModelID(provider llmtypes.Provider, modelID string) string {
	modelID = strings.TrimSpace(modelID)
	// Some providers may already return ids in "provider:model" form. Strip a
	// matching prefix first, so we always end up with exactly one canonical form.
	modelID = strings.TrimPrefix(modelID, string(provider)+":")
	return fmt.Sprintf("%s:%s", provider, modelID)
}

func ParseQualifiedModelID(qualifiedModelID string) (llmtypes.Provider, string, error) {
	qualifiedModelID = strings.TrimSpace(qualifiedModelID)
	providerPart, modelPart, ok := strings.Cut(qualifiedModelID, ":")
	if !ok || strings.TrimSpace(providerPart) == "" || strings.TrimSpace(modelPart) == "" {
		return "", "", xerrors.Newf("invalid qualified model id: %s", qualifiedModelID)
	}

	provider := llmtypes.Provider(strings.TrimSpace(providerPart))
	if !isKnownProvider(provider) {
		return "", "", xerrors.Newf("unknown provider in model id: %s", provider)
	}

	return provider, strings.TrimSpace(modelPart), nil
}

func (r *Registry) listEnabledProviderConfigs(ctx context.Context) (map[llmtypes.Provider]*llmtypes.ProviderConfig, error) {
	configs, err := r.storage.ListProviderConfigs(ctx, chatstorage.ProviderConfigsFilter{
		IsEnabled: ptr.Of(true),
	})
	if err != nil {
		return nil, xerrors.Newf("failed to list provider configs: %w", err)
	}

	configsByProvider := make(map[llmtypes.Provider]*llmtypes.ProviderConfig, len(configs))
	for _, config := range configs {
		if config == nil {
			continue
		}
		// The registry deliberately supports only one enabled config per provider
		// so model resolution stays deterministic for ids like "openai:gpt-5.2".
		if _, exists := configsByProvider[config.Provider]; exists {
			return nil, xerrors.Newf("multiple enabled provider configs found for provider %s", config.Provider)
		}
		configsByProvider[config.Provider] = config
	}

	return configsByProvider, nil
}

func (r *Registry) clientForConfig(config *llmtypes.ProviderConfig) (Client, error) {
	factory, ok := r.factories[config.Provider]
	if !ok {
		return nil, xerrors.Newf("provider %s is not configured in registry: %w", config.Provider, chaterrors.ErrProviderNotAvailable)
	}

	decryptedConfig := *config
	apiKey, err := r.cipher.Decrypt(config.APIKeyEnc)
	if err != nil {
		return nil, xerrors.Newf("provider %s credentials are unavailable: %w", config.Provider, chaterrors.ErrProviderNotAvailable)
	}
	decryptedConfig.APIKeyEnc = apiKey

	client, err := factory(&decryptedConfig)
	if err != nil {
		return nil, xerrors.Newf("failed to create provider client for %s: %w", config.Provider, err)
	}

	return client, nil
}

func (r *Registry) clientForPlainConfig(config *llmtypes.ProviderConfig) (Client, error) {
	factory, ok := r.factories[config.Provider]
	if !ok {
		return nil, xerrors.Newf("provider %s is not configured in registry: %w", config.Provider, chaterrors.ErrProviderNotAvailable)
	}

	client, err := factory(config)
	if err != nil {
		return nil, xerrors.Newf("failed to create provider client for %s: %w", config.Provider, err)
	}

	return client, nil
}

func (r *Registry) listProviderModels(ctx context.Context, config *llmtypes.ProviderConfig) ([]llmtypes.Model, error) {
	client, err := r.clientForConfig(config)
	if err != nil {
		return nil, err
	}

	providerModels, err := client.ListModels(ctx)
	if err != nil {
		return nil, xerrors.Newf("failed to list models for provider %s: %w", config.Provider, err)
	}

	return normalizeProviderModels(config.Provider, providerModels)
}

var _ ProviderTester = (*Registry)(nil)

// normalizeProviderModels normalizes and deduplicates a list of models for a provider, ensuring consistent ID formatting.
func normalizeProviderModels(provider llmtypes.Provider, models []llmtypes.Model) ([]llmtypes.Model, error) {
	normalizedModels := make([]llmtypes.Model, 0, len(models))
	seen := make(map[string]struct{}, len(models))

	for _, model := range models {
		rawModelID := strings.TrimSpace(model.ID)
		if rawModelID == "" {
			return nil, xerrors.Newf("provider %s returned empty model id", provider)
		}

		qualifiedModelID := QualifiedModelID(provider, rawModelID)
		if strings.Contains(rawModelID, ":") {
			parsedProvider, parsedModelID, err := ParseQualifiedModelID(rawModelID)
			if err != nil {
				return nil, err
			}
			// Providers are allowed to return already-qualified ids, but the
			// embedded provider prefix still has to match the provider being queried.
			if parsedProvider != provider {
				return nil, xerrors.Newf("provider %s returned model id for different provider %s", provider, parsedProvider)
			}
			qualifiedModelID = QualifiedModelID(parsedProvider, parsedModelID)
			rawModelID = parsedModelID
		}

		// Deduplicate after normalization so "gpt-5.2" and "openai:gpt-5.2"
		// collapse to the same canonical catalog entry.
		if _, exists := seen[qualifiedModelID]; exists {
			return nil, xerrors.Newf("provider %s returned duplicate model id %s", provider, qualifiedModelID)
		}
		seen[qualifiedModelID] = struct{}{}

		model.ID = qualifiedModelID
		model.Provider = provider
		model.IsEnabled = true
		if strings.TrimSpace(model.DisplayName) == "" {
			model.DisplayName = rawModelID
		}

		normalizedModels = append(normalizedModels, model)
	}

	return normalizedModels, nil
}

func sortedProviders(configsByProvider map[llmtypes.Provider]*llmtypes.ProviderConfig) []llmtypes.Provider {
	return slices.Sorted(maps.Keys(configsByProvider))
}

func isKnownProvider(provider llmtypes.Provider) bool {
	switch provider {
	case llmtypes.ProviderOpenAI, llmtypes.ProviderAnthropic, llmtypes.ProviderGemini, llmtypes.ProviderOllama:
		return true
	default:
		return false
	}
}
