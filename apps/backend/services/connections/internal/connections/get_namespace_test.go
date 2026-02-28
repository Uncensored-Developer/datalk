package connections

import (
	"errors"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage/testing"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	serviceerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_GetNamespace(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	expectedNamespace := &connections.Namespace{
		ID:            12,
		ConnectionID:  3,
		Name:          "public",
		NamespaceType: connections.NamespaceTypeSchema,
		IsEnabled:     true,
		CreatedAt:     now,
	}

	testCases := []struct {
		name        string
		namespaceID int32
		modifyFn    func(namespaceID int32, mockStorage *storagetesting.Storage)
		expect      *connections.Namespace
		expectError error
	}{
		{
			name:        "storage failure",
			namespaceID: 7,
			modifyFn: func(namespaceID int32, mockStorage *storagetesting.Storage) {
				mockStorage.
					On("ListNamespace", mock.Anything, storage.ListNamespaceParam{ID: []int32{namespaceID}}).
					Return(nil, errors.New("db offline"))
			},
			expectError: errors.New("failed to list namespaces"),
		},
		{
			name:        "namespace not found",
			namespaceID: 99,
			modifyFn: func(namespaceID int32, mockStorage *storagetesting.Storage) {
				mockStorage.
					On("ListNamespace", mock.Anything, storage.ListNamespaceParam{ID: []int32{namespaceID}}).
					Return([]*connections.Namespace{}, nil)
			},
			expectError: serviceerrors.ErrNamespaceNotFound,
		},
		{
			name:        "success: returns first result",
			namespaceID: expectedNamespace.ID,
			modifyFn: func(namespaceID int32, mockStorage *storagetesting.Storage) {
				mockStorage.
					On("ListNamespace", mock.Anything, storage.ListNamespaceParam{ID: []int32{namespaceID}}).
					Return([]*connections.Namespace{expectedNamespace}, nil)
			},
			expect: expectedNamespace,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockStorage := storagetesting.NewStorage(t)
			if tc.modifyFn != nil {
				tc.modifyFn(tc.namespaceID, mockStorage)
			}

			service := &Service{
				Base:    &base.Base{},
				storage: mockStorage,
			}

			got, err := service.GetNamespace(t.Context(), tc.namespaceID)
			if tc.expectError != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expectError.Error())
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tc.expect, got)
		})
	}
}
