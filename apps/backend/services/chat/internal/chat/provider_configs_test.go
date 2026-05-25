package chat

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage/testing"
	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testCipher struct{}

func (testCipher) Encrypt(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}

func (testCipher) Decrypt(ciphertext string) (string, error) {
	return ciphertext, nil
}

func newProviderConfigTestService(storage chatstorage.Storage) *Service {
	return NewService(config.Config{}, nil, storage, nil, nil, nil, nil, testCipher{})
}

func TestService_ListProviderConfigs(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	expected := []*llm.ProviderConfig{
		{ID: 1, Provider: llm.ProviderOpenAI, DisplayName: "OpenAI", APIKeyEnc: "enc:key"},
	}
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListProviderConfigs", ctx, chatstorage.ProviderConfigsFilter{}).
		Return(expected, nil).
		Once()

	got, err := newProviderConfigTestService(mockStorage).ListProviderConfigs(ctx)

	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestService_SaveProviderConfig_CreatesAndEncryptsAPIKey(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	apiKey := "secret-key"
	baseURL := "https://api.openai.test"
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListProviderConfigs", ctx, mock.MatchedBy(func(filter chatstorage.ProviderConfigsFilter) bool {
			return assert.Equal(t, []llm.Provider{llm.ProviderOpenAI}, filter.Provider)
		})).
		Return(nil, nil).
		Once()
	mockStorage.
		On("UpsertProviderConfig", ctx, mock.MatchedBy(func(config *llm.ProviderConfig) bool {
			assert.Equal(t, llm.ProviderOpenAI, config.Provider)
			assert.Equal(t, "OpenAI", config.DisplayName)
			assert.Equal(t, "enc:secret-key", config.APIKeyEnc)
			require.NotNil(t, config.BaseURL)
			assert.Equal(t, baseURL, *config.BaseURL)
			assert.True(t, config.IsEnabled)
			assert.JSONEq(t, `{"tier":"prod"}`, string(config.Metadata))
			config.ID = 10
			return true
		})).
		Return(nil).
		Once()

	got, err := newProviderConfigTestService(mockStorage).SaveProviderConfig(ctx, SaveProviderConfigParams{
		Provider:    llm.ProviderOpenAI,
		DisplayName: " OpenAI ",
		APIKey:      &apiKey,
		BaseURL:     &baseURL,
		IsEnabled:   true,
		Metadata:    json.RawMessage(`{"tier":"prod"}`),
	})

	require.NoError(t, err)
	assert.Equal(t, int64(10), got.ID)
	assert.Equal(t, "enc:secret-key", got.APIKeyEnc)
}

func TestService_SaveProviderConfig_UpdatesWithoutAPIKeyPreservesExistingKey(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListProviderConfigs", ctx, mock.Anything).
		Return([]*llm.ProviderConfig{{Provider: llm.ProviderOpenAI, APIKeyEnc: "enc:existing"}}, nil).
		Once()
	mockStorage.
		On("UpsertProviderConfig", ctx, mock.MatchedBy(func(config *llm.ProviderConfig) bool {
			return assert.Equal(t, "enc:existing", config.APIKeyEnc)
		})).
		Return(nil).
		Once()

	got, err := newProviderConfigTestService(mockStorage).SaveProviderConfig(ctx, SaveProviderConfigParams{
		Provider:    llm.ProviderOpenAI,
		DisplayName: "OpenAI",
		IsEnabled:   true,
	})

	require.NoError(t, err)
	assert.Equal(t, "enc:existing", got.APIKeyEnc)
}

func TestService_SaveProviderConfig_RejectsCreateWithoutAPIKey(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListProviderConfigs", ctx, mock.Anything).
		Return(nil, nil).
		Once()

	_, err := newProviderConfigTestService(mockStorage).SaveProviderConfig(ctx, SaveProviderConfigParams{
		Provider:    llm.ProviderOpenAI,
		DisplayName: "OpenAI",
		IsEnabled:   true,
	})

	require.ErrorIs(t, err, chaterrors.ErrInvalidProviderConfig)
}

func TestService_SaveProviderConfig_AllowsOllamaWithoutAPIKey(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	baseURL := "http://localhost:11434"
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListProviderConfigs", ctx, mock.MatchedBy(func(filter chatstorage.ProviderConfigsFilter) bool {
			return assert.Equal(t, []llm.Provider{llm.ProviderOllama}, filter.Provider)
		})).
		Return(nil, nil).
		Once()
	mockStorage.
		On("UpsertProviderConfig", ctx, mock.MatchedBy(func(config *llm.ProviderConfig) bool {
			assert.Equal(t, llm.ProviderOllama, config.Provider)
			assert.Equal(t, "", config.APIKeyEnc)
			require.NotNil(t, config.BaseURL)
			assert.Equal(t, baseURL, *config.BaseURL)
			config.ID = 10
			return true
		})).
		Return(nil).
		Once()

	got, err := newProviderConfigTestService(mockStorage).SaveProviderConfig(ctx, SaveProviderConfigParams{
		Provider:    llm.ProviderOllama,
		DisplayName: "Ollama",
		BaseURL:     &baseURL,
		IsEnabled:   true,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(10), got.ID)
	assert.Equal(t, "", got.APIKeyEnc)
}

func TestService_SaveProviderConfig_ValidationRunsBeforeStorage(t *testing.T) {
	t.Parallel()

	mockStorage := storagetesting.NewStorage(t)

	_, err := newProviderConfigTestService(mockStorage).SaveProviderConfig(t.Context(), SaveProviderConfigParams{
		Provider:    "unknown",
		DisplayName: "OpenAI",
	})
	require.ErrorIs(t, err, chaterrors.ErrInvalidProviderConfig)

	_, err = newProviderConfigTestService(mockStorage).SaveProviderConfig(t.Context(), SaveProviderConfigParams{
		Provider:    llm.ProviderOpenAI,
		DisplayName: "   ",
	})
	require.ErrorIs(t, err, chaterrors.ErrInvalidProviderConfig)
}

func TestService_SaveProviderConfig_WrapsStorageErrors(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	storageErr := errors.New("db down")
	apiKey := "secret"
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListProviderConfigs", ctx, mock.Anything).
		Return(nil, nil).
		Once()
	mockStorage.
		On("UpsertProviderConfig", ctx, mock.Anything).
		Return(storageErr).
		Once()

	_, err := newProviderConfigTestService(mockStorage).SaveProviderConfig(ctx, SaveProviderConfigParams{
		Provider:    llm.ProviderOpenAI,
		DisplayName: "OpenAI",
		APIKey:      &apiKey,
		IsEnabled:   true,
	})

	require.ErrorIs(t, err, storageErr)
	assert.Contains(t, err.Error(), "failed to save provider config")
}
