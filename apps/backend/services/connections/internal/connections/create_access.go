package connections

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/mdobak/go-xerrors"
)

type NewAccess struct {
	UserID       int32
	ConnectionID int32
	CanQuery     bool
	AllowWrites  bool
	CanManage    bool
}

func (n *NewAccess) Validate() error {
	if n.UserID <= 0 {
		return errors.New("user id is required")
	}
	if n.ConnectionID <= 0 {
		return errors.New("connection id is required")
	}
	return nil
}

func (s *Service) CreateAccess(ctx context.Context, newAccess NewAccess) (*connections.Access, error) {
	if err := newAccess.Validate(); err != nil {
		return nil, err
	}

	access := connections.Access{
		UserID:       newAccess.UserID,
		ConnectionID: newAccess.ConnectionID,
		CanQuery:     newAccess.CanQuery,
		AllowWrites:  newAccess.AllowWrites,
		CanManage:    newAccess.CanManage,
		GrantedAt:    time.Now().UTC(),
	}
	if err := s.storage.UpsertAccess(ctx, &access); err != nil {
		return nil, xerrors.Newf("failed to insert access: %w", err)
	}

	s.Logger().Info(
		"connection access granted",
		slog.Int("user_id", int(access.UserID)),
		slog.Int("connection_id", int(access.ConnectionID)),
		slog.Bool("can_query", access.CanQuery),
		slog.Bool("allow_writes", access.AllowWrites),
		slog.Bool("can_manage", access.CanManage),
	)

	return &access, nil
}
