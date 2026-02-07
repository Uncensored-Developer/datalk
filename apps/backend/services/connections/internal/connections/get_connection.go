package connections

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	serviceerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/errors"
	"github.com/mdobak/go-xerrors"
)

func (s *Service) GetConnection(ctx context.Context, ID int32) (*connections.Connection, error) {
	fetchedConnections, err := s.storage.ListConnections(ctx, storage.ListConnectionsParam{ID: []int32{ID}})
	if err != nil {
		return nil, xerrors.Newf("failed to list connections: %w", err)
	}
	if len(fetchedConnections) == 0 {
		return nil, serviceerrors.ErrConnectionNotFound
	}
	return fetchedConnections[0], nil
}
