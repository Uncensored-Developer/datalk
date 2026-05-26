package connections

import (
	"context"
	"log/slog"

	"github.com/mdobak/go-xerrors"
)

func (s *Service) DeleteConnection(ctx context.Context, id int32) error {
	if id <= 0 {
		return xerrors.New("connection id is required")
	}

	if err := s.storage.DeleteConnection(ctx, id); err != nil {
		return xerrors.Newf("failed to delete connection: %w", err)
	}

	s.Logger().Info("connection deleted", slog.Int("connection_id", int(id)))
	return nil
}
