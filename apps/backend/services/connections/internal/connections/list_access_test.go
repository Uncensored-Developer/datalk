package connections

import (
	"errors"
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage/testing"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_ListAccess(t *testing.T) {
	t.Parallel()

	expected := []*connections.Access{{UserID: 7, ConnectionID: 10, CanQuery: true}}
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListAccess", mock.Anything, storage.ListAccessParam{ConnectionID: []int32{10}}).
		Return(expected, nil).
		Once()

	service := &Service{Base: &base.Base{}, storage: mockStorage}
	got, err := service.ListAccess(t.Context(), ListAccess{ConnectionID: []int32{10}})

	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestService_ListAccess_WrapsStorageErrors(t *testing.T) {
	t.Parallel()

	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListAccess", mock.Anything, storage.ListAccessParam{ConnectionID: []int32{10}}).
		Return(nil, errors.New("db offline")).
		Once()

	service := &Service{Base: &base.Base{}, storage: mockStorage}
	_, err := service.ListAccess(t.Context(), ListAccess{ConnectionID: []int32{10}})

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list access")
}
