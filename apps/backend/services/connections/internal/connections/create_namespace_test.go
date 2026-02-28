package connections

import (
	"errors"
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage/testing"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_CreateNamespace(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		newNamespace NewNamespace
		modifyFn     func(newNamespace NewNamespace, mockStorage *storagetesting.Storage)
		expect       *connections.Namespace
		expectError  error
	}{
		{
			name: "invalid: empty name",
			newNamespace: NewNamespace{
				Name:          "",
				NamespaceType: connections.NamespaceTypeSchema,
				ConnectionID:  1,
			},
			expectError: errors.New("name is required"),
			modifyFn:    nil,
		},
		{
			name: "invalid: empty namespace type",
			newNamespace: NewNamespace{
				Name:          "public",
				NamespaceType: "",
				ConnectionID:  1,
			},
			expectError: errors.New("namespace type is required"),
			modifyFn:    nil,
		},
		{
			name: "invalid: unknown namespace type",
			newNamespace: NewNamespace{
				Name:          "public",
				NamespaceType: connections.NamespaceType("blob"),
				ConnectionID:  1,
			},
			expectError: errors.New("namespace type is invalid"),
			modifyFn:    nil,
		},
		{
			name: "invalid: missing connection id",
			newNamespace: NewNamespace{
				Name:          "public",
				NamespaceType: connections.NamespaceTypeSchema,
				ConnectionID:  0,
			},
			expectError: errors.New("connection id is required"),
			modifyFn:    nil,
		},
		{
			name: "storage failure",
			newNamespace: NewNamespace{
				Name:          "public",
				NamespaceType: connections.NamespaceTypeSchema,
				ConnectionID:  7,
			},
			modifyFn: func(newNamespace NewNamespace, mockStorage *storagetesting.Storage) {
				matcher := mock.MatchedBy(func(n *connections.Namespace) bool {
					return n != nil &&
						n.Name == newNamespace.Name &&
						n.NamespaceType == newNamespace.NamespaceType &&
						n.ConnectionID == newNamespace.ConnectionID &&
						n.IsEnabled
				})
				mockStorage.
					On("UpsertNamespace", mock.Anything, matcher).
					Return(errors.New("db write failed"))
			},
			expectError: errors.New("failed to insert namespace"),
		},
		{
			name: "success",
			newNamespace: NewNamespace{
				Name:          "analytics",
				NamespaceType: connections.NamespaceTypeDatabase,
				ConnectionID:  3,
			},
			modifyFn: func(newNamespace NewNamespace, mockStorage *storagetesting.Storage) {
				mockStorage.
					On("UpsertNamespace", mock.Anything, mock.MatchedBy(func(n *connections.Namespace) bool {
						return n != nil &&
							n.Name == newNamespace.Name &&
							n.NamespaceType == newNamespace.NamespaceType &&
							n.ConnectionID == newNamespace.ConnectionID &&
							n.IsEnabled && !n.CreatedAt.IsZero()
					})).
					Return(nil)
			},
			expect: &connections.Namespace{
				Name:          "analytics",
				NamespaceType: connections.NamespaceTypeDatabase,
				ConnectionID:  3,
				IsEnabled:     true,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockStorage := storagetesting.NewStorage(t)
			if tc.modifyFn != nil {
				tc.modifyFn(tc.newNamespace, mockStorage)
			}

			service := &Service{
				Base:    &base.Base{},
				storage: mockStorage,
			}

			got, err := service.CreateNamespace(t.Context(), tc.newNamespace)
			if tc.expectError != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expectError.Error())
				return
			}

			require.NoError(t, err)
			if tc.expect != nil {
				assert.Equal(t, tc.expect.Name, got.Name)
				assert.Equal(t, tc.expect.NamespaceType, got.NamespaceType)
				assert.Equal(t, tc.expect.ConnectionID, got.ConnectionID)
				assert.True(t, got.IsEnabled)
				assert.False(t, got.CreatedAt.IsZero())
			}
		})
	}
}
