package connections

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/mdobak/go-xerrors"
)

type ListAccess struct {
	UserID       []int32
	ConnectionID []int32
}

func (s *Service) ListAccess(ctx context.Context, params ListAccess) ([]*connections.Access, error) {
	access, err := s.storage.ListAccess(ctx, storage.ListAccessParam{
		UserID:       params.UserID,
		ConnectionID: params.ConnectionID,
	})
	if err != nil {
		return nil, xerrors.Newf("failed to list access: %w", err)
	}

	return access, nil
}
