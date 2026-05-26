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

func TestService_ListConnections_AdminListsAll(t *testing.T) {
	t.Parallel()

	expected := []*connections.Connection{{ID: 10, Name: "warehouse"}}
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListConnections", mock.Anything, storage.ListConnectionsParam{}).
		Return(expected, nil).
		Once()

	service := &Service{Base: &base.Base{}, storage: mockStorage}
	got, err := service.ListConnections(t.Context(), ListConnections{UserID: 7, IsAdmin: true})

	require.NoError(t, err)
	assert.Equal(t, expected, got)
	mockStorage.AssertNotCalled(t, "ListAccess", mock.Anything, mock.Anything)
}

func TestService_ListConnections_MemberListsGrantedConnections(t *testing.T) {
	t.Parallel()

	expected := []*connections.Connection{{ID: 10, Name: "warehouse"}, {ID: 11, Name: "analytics"}}
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListAccess", mock.Anything, storage.ListAccessParam{UserID: []int32{7}}).
		Return([]*connections.Access{
			{UserID: 7, ConnectionID: 10},
			{UserID: 7, ConnectionID: 11},
			{UserID: 7, ConnectionID: 10},
		}, nil).
		Once()
	mockStorage.
		On("ListConnections", mock.Anything, storage.ListConnectionsParam{ID: []int32{10, 11}}).
		Return(expected, nil).
		Once()

	service := &Service{Base: &base.Base{}, storage: mockStorage}
	got, err := service.ListConnections(t.Context(), ListConnections{UserID: 7})

	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestService_ListConnections_MemberWithoutAccessReturnsEmpty(t *testing.T) {
	t.Parallel()

	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListAccess", mock.Anything, storage.ListAccessParam{UserID: []int32{7}}).
		Return([]*connections.Access{}, nil).
		Once()

	service := &Service{Base: &base.Base{}, storage: mockStorage}
	got, err := service.ListConnections(t.Context(), ListConnections{UserID: 7})

	require.NoError(t, err)
	assert.Empty(t, got)
	mockStorage.AssertNotCalled(t, "ListConnections", mock.Anything, mock.Anything)
}

func TestService_ListConnections_WrapsStorageErrors(t *testing.T) {
	t.Parallel()

	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListConnections", mock.Anything, storage.ListConnectionsParam{}).
		Return(nil, errors.New("db offline")).
		Once()

	service := &Service{Base: &base.Base{}, storage: mockStorage}
	_, err := service.ListConnections(t.Context(), ListConnections{UserID: 7, IsAdmin: true})

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list connections")
}
