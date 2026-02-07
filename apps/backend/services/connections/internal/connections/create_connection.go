package connections

import (
	"context"
	"errors"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/mdobak/go-xerrors"
)

type NewConnection struct {
	Name     string
	Database connections.Database
	DSN      string
	UserID   int32
}

func (n *NewConnection) Validate() error {
	if n.Name == "" {
		return errors.New("name is required")
	}
	if n.Database == "" {
		return errors.New("database is required")
	}
	switch n.Database {
	case connections.DatabasePostgres, connections.DatabaseMySQL, connections.DatabaseCQL:
		// valid
	default:
		return errors.New("database is invalid")
	}
	if n.UserID <= 0 {
		return errors.New("user id is required")
	}
	return nil
}

func (s *Service) CreateConnection(ctx context.Context, newConnection NewConnection) (*connections.Connection, error) {
	if err := newConnection.Validate(); err != nil {
		return nil, err
	}

	connection := connections.Connection{
		Name:      newConnection.Name,
		Database:  newConnection.Database,
		DSN:       newConnection.DSN,
		UserID:    newConnection.UserID,
		IsEnabled: true,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.storage.UpsertConnection(ctx, &connection); err != nil {
		return nil, xerrors.Newf("failed to insert connection: %w", err)
	}

	return &connection, nil
}
