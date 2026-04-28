package users

import (
	"database/sql"
	"log/slog"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage/db"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/users/hashers"
)

type Service struct {
	*base.Base

	storage storage.Storage
	hasher  hashers.Hasher
}

func NewService(conn *sql.DB, logger *slog.Logger, cfg config.Config) *Service {
	return &Service{
		Base:    base.NewBase("users-core", logger, cfg),
		storage: db.NewStorage(conn),
		hasher:  hashers.NewArgon2Hasher(),
	}
}
