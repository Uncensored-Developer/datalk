package llm

import (
	"context"
	"errors"
	"testing"

	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage/testing"
	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type stubClient struct {
	models []llmtypes.Model
	err    error
}

func (s stubClient) ListModels(context.Context) ([]llmtypes.Model, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.models, nil
}

func (s stubClient) GenerateSQL(context.Context, llmtypes.GenerateSQLRequest) (*llmtypes.GenerateSQLResponse, error) {
	return nil, errors.New("not implemented in step 6 tests")
}

func TestRegistry_ListAvailableModels_FiltersDisabledConfigs(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListProviderConfigs", ctx, mock.MatchedBy(func(filter chatstorage.ProviderConfigsFilter) bool {
			return filter.IsEnabled != nil && *filter.IsEnabled
		})).
		Return([]*llmtypes.ProviderConfig{
			{
				ID:        1,
				Provider:  llmtypes.ProviderOpenAI,
				IsEnabled: true,
			},
		}, nil).
		Once()

	registry := NewRegistry(mockStorage, map[llmtypes.Provider]ClientFactory{
		llmtypes.ProviderOpenAI: func(config *llmtypes.ProviderConfig) (Client, error) {
			require.Equal(t, int64(1), config.ID)
			return stubClient{
				models: []llmtypes.Model{
					{ID: "gpt-5.2"},
				},
			}, nil
		},
	})

	models, err := registry.ListAvailableModels(ctx)
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, "openai:gpt-5.2", models[0].ID)
	assert.Equal(t, llmtypes.ProviderOpenAI, models[0].Provider)
	assert.True(t, models[0].IsEnabled)
}

func TestRegistry_ListAvailableModels_MergesProvidersAndQualifiesIDs(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListProviderConfigs", ctx, mock.MatchedBy(func(filter chatstorage.ProviderConfigsFilter) bool {
			return filter.IsEnabled != nil && *filter.IsEnabled
		})).
		Return([]*llmtypes.ProviderConfig{
			{ID: 1, Provider: llmtypes.ProviderOpenAI, IsEnabled: true},
			{ID: 2, Provider: llmtypes.ProviderAnthropic, IsEnabled: true},
		}, nil).
		Once()

	registry := NewRegistry(mockStorage, map[llmtypes.Provider]ClientFactory{
		llmtypes.ProviderOpenAI: func(config *llmtypes.ProviderConfig) (Client, error) {
			return stubClient{models: []llmtypes.Model{{ID: "shared-model"}}}, nil
		},
		llmtypes.ProviderAnthropic: func(config *llmtypes.ProviderConfig) (Client, error) {
			return stubClient{models: []llmtypes.Model{{ID: "shared-model"}}}, nil
		},
	})

	models, err := registry.ListAvailableModels(ctx)
	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.Equal(t, "anthropic:shared-model", models[0].ID)
	assert.Equal(t, "openai:shared-model", models[1].ID)
}

func TestRegistry_ListAvailableModels_ErrorsOnMultipleEnabledConfigsForProvider(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListProviderConfigs", ctx, mock.MatchedBy(func(filter chatstorage.ProviderConfigsFilter) bool {
			return filter.IsEnabled != nil && *filter.IsEnabled
		})).
		Return([]*llmtypes.ProviderConfig{
			{ID: 1, Provider: llmtypes.ProviderOpenAI, IsEnabled: true},
			{ID: 2, Provider: llmtypes.ProviderOpenAI, IsEnabled: true},
		}, nil).
		Once()

	registry := NewRegistry(mockStorage, nil)

	_, err := registry.ListAvailableModels(ctx)
	require.EqualError(t, err, "multiple enabled provider configs found for provider openai")
}

func TestRegistry_ListAvailableModels_ErrorsOnDuplicateModelIDsFromProvider(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListProviderConfigs", ctx, mock.MatchedBy(func(filter chatstorage.ProviderConfigsFilter) bool {
			return filter.IsEnabled != nil && *filter.IsEnabled
		})).
		Return([]*llmtypes.ProviderConfig{
			{ID: 1, Provider: llmtypes.ProviderOpenAI, IsEnabled: true},
		}, nil).
		Once()

	registry := NewRegistry(mockStorage, map[llmtypes.Provider]ClientFactory{
		llmtypes.ProviderOpenAI: func(config *llmtypes.ProviderConfig) (Client, error) {
			return stubClient{
				models: []llmtypes.Model{
					{ID: "gpt-5.2"},
					{ID: "openai:gpt-5.2"},
				},
			}, nil
		},
	})

	_, err := registry.ListAvailableModels(ctx)
	require.EqualError(t, err, "provider openai returned duplicate model id openai:gpt-5.2")
}

func TestRegistry_ResolveQualifiedModel(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListProviderConfigs", ctx, mock.MatchedBy(func(filter chatstorage.ProviderConfigsFilter) bool {
			return filter.IsEnabled != nil && *filter.IsEnabled
		})).
		Return([]*llmtypes.ProviderConfig{
			{ID: 1, Provider: llmtypes.ProviderOpenAI, IsEnabled: true},
		}, nil).
		Twice()

	registry := NewRegistry(mockStorage, map[llmtypes.Provider]ClientFactory{
		llmtypes.ProviderOpenAI: func(config *llmtypes.ProviderConfig) (Client, error) {
			return stubClient{
				models: []llmtypes.Model{
					{ID: "gpt-5.2", DisplayName: "GPT 5.2"},
				},
			}, nil
		},
	})

	resolved, err := registry.ResolveQualifiedModel(ctx, "openai:gpt-5.2")
	require.NoError(t, err)
	assert.Equal(t, int64(1), resolved.ProviderConfig.ID)
	assert.Equal(t, "openai:gpt-5.2", resolved.QualifiedModelID)
	assert.Equal(t, "gpt-5.2", resolved.ProviderModelID)
	assert.Equal(t, "openai:gpt-5.2", resolved.Model.ID)

	_, err = registry.ResolveQualifiedModel(ctx, "openai:missing-model")
	require.ErrorIs(t, err, chaterrors.ErrModelNotAvailable)
}

func TestParseQualifiedModelID(t *testing.T) {
	t.Parallel()

	provider, modelID, err := ParseQualifiedModelID("openai:gpt-5.2")
	require.NoError(t, err)
	assert.Equal(t, llmtypes.ProviderOpenAI, provider)
	assert.Equal(t, "gpt-5.2", modelID)

	_, _, err = ParseQualifiedModelID("gpt-5.2")
	require.EqualError(t, err, "invalid qualified model id: gpt-5.2")

	_, _, err = ParseQualifiedModelID("unknown:model")
	require.EqualError(t, err, "unknown provider in model id: unknown")
}
