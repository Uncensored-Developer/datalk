package connections

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/mdobak/go-xerrors"
)

type UpdateConnection struct {
	ID        int32
	Name      *string
	Database  *connections.Database
	DSN       *string
	IsEnabled *bool
	Metadata  *connections.Metadata
}

func (u *UpdateConnection) Validate() error {
	if u.ID <= 0 {
		return errors.New("connection id is required")
	}
	if u.Name != nil && strings.TrimSpace(*u.Name) == "" {
		return errors.New("name is required")
	}
	if u.Database != nil {
		switch *u.Database {
		case connections.DatabasePostgres, connections.DatabaseMySQL, connections.DatabaseCQL:
			// valid
		default:
			return errors.New("database is invalid")
		}
	}
	return nil
}

func (s *Service) UpdateConnection(ctx context.Context, params UpdateConnection) (*connections.Connection, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	connection, err := s.GetConnection(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	if params.Name != nil {
		connection.Name = strings.TrimSpace(*params.Name)
	}
	if params.Database != nil {
		connection.Database = *params.Database
	}
	if params.DSN != nil {
		connection.DSN = *params.DSN
	}
	if params.IsEnabled != nil {
		connection.IsEnabled = *params.IsEnabled
	}
	if params.Metadata != nil {
		connection.Metadata = *params.Metadata
	}

	if err := s.encryptConnectionDSN(connection); err != nil {
		return nil, err
	}
	if err := s.storage.UpdateConnection(ctx, connection); err != nil {
		return nil, xerrors.Newf("failed to update connection: %w", err)
	}
	if err := s.decryptConnectionDSN(connection); err != nil {
		return nil, err
	}

	s.Logger().Info(
		"connection updated",
		slog.Int("connection_id", int(connection.ID)),
		slog.Int("user_id", int(connection.UserID)),
		slog.String("database", string(connection.Database)),
	)

	return connection, nil
}
