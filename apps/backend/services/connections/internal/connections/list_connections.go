package connections

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/mdobak/go-xerrors"
)

type ListConnections struct {
	UserID  int32
	IsAdmin bool
}

func (s *Service) ListConnections(ctx context.Context, params ListConnections) ([]*connections.Connection, error) {
	if params.IsAdmin {
		connections, err := s.storage.ListConnections(ctx, storage.ListConnectionsParam{})
		if err != nil {
			return nil, xerrors.Newf("failed to list connections: %w", err)
		}
		if err := s.decryptConnectionDSNs(connections); err != nil {
			return nil, err
		}
		return connections, nil
	}

	if params.UserID <= 0 {
		return []*connections.Connection{}, nil
	}

	access, err := s.storage.ListAccess(ctx, storage.ListAccessParam{UserID: []int32{params.UserID}})
	if err != nil {
		return nil, xerrors.Newf("failed to list connection access: %w", err)
	}
	if len(access) == 0 {
		return []*connections.Connection{}, nil
	}

	connectionIDs := make([]int32, 0, len(access))
	seen := make(map[int32]struct{}, len(access))
	for _, item := range access {
		if item == nil {
			continue
		}
		if _, ok := seen[item.ConnectionID]; ok {
			continue
		}
		seen[item.ConnectionID] = struct{}{}
		connectionIDs = append(connectionIDs, item.ConnectionID)
	}
	if len(connectionIDs) == 0 {
		return []*connections.Connection{}, nil
	}

	connections, err := s.storage.ListConnections(ctx, storage.ListConnectionsParam{ID: connectionIDs})
	if err != nil {
		return nil, xerrors.Newf("failed to list connections: %w", err)
	}
	if err := s.decryptConnectionDSNs(connections); err != nil {
		return nil, err
	}
	return connections, nil
}
