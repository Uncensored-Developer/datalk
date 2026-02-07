package connections

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	serviceerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/errors"
	"github.com/mdobak/go-xerrors"
)

func (s *Service) GetAccess(ctx context.Context, userID int32, connectionID int32) (*connections.Access, error) {
	fetchedAccess, err := s.storage.ListAccess(ctx, storage.ListAccessParam{
		UserID:       []int32{userID},
		ConnectionID: []int32{connectionID},
	})
	if err != nil {
		return nil, xerrors.Newf("failed to list access: %w", err)
	}
	if len(fetchedAccess) == 0 {
		return nil, serviceerrors.ErrAccessNotFound
	}
	return fetchedAccess[0], nil
}
