package connections

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	serviceerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/errors"
	"github.com/mdobak/go-xerrors"
)

func (s *Service) GetNamespace(ctx context.Context, ID int32) (*connections.Namespace, error) {
	fetchedNamespaces, err := s.storage.ListNamespace(ctx, storage.ListNamespaceParam{ID: []int32{ID}})
	if err != nil {
		return nil, xerrors.Newf("failed to list namespaces: %w", err)
	}
	if len(fetchedNamespaces) == 0 {
		return nil, serviceerrors.ErrNamespaceNotFound
	}
	return fetchedNamespaces[0], nil
}
