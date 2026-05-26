package connections

import (
	"context"
	"strings"
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage/testing"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testConnectionCipher struct{}

func (testConnectionCipher) Encrypt(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}

func (testConnectionCipher) Decrypt(ciphertext string) (string, error) {
	return strings.TrimPrefix(ciphertext, "enc:"), nil
}

func TestService_CreateConnection_EncryptsDSNBeforeStorage(t *testing.T) {
	t.Parallel()

	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("UpsertConnection", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			connection := args.Get(1).(*connections.Connection)
			assert.Equal(t, "enc:postgres://warehouse", connection.DSN)
			connection.ID = 10
		}).
		Return(nil).
		Once()

	service := &Service{
		Base:    &base.Base{},
		storage: mockStorage,
		cipher:  testConnectionCipher{},
	}

	got, err := service.CreateConnection(context.Background(), NewConnection{
		Name:     "warehouse",
		Database: connections.DatabasePostgres,
		DSN:      "postgres://warehouse",
		UserID:   7,
	})

	require.NoError(t, err)
	assert.Equal(t, "postgres://warehouse", got.DSN)
}

func TestService_GetConnection_DecryptsDSNAfterStorage(t *testing.T) {
	t.Parallel()

	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListConnections", mock.Anything, storage.ListConnectionsParam{ID: []int32{10}}).
		Return([]*connections.Connection{{
			ID:        10,
			Name:      "warehouse",
			Database:  connections.DatabasePostgres,
			DSN:       "enc:postgres://warehouse",
			UserID:    7,
			IsEnabled: true,
		}}, nil).
		Once()

	service := &Service{
		Base:    &base.Base{},
		storage: mockStorage,
		cipher:  testConnectionCipher{},
	}

	got, err := service.GetConnection(context.Background(), 10)

	require.NoError(t, err)
	assert.Equal(t, "postgres://warehouse", got.DSN)
}
