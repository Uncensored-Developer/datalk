package users

import (
	"database/sql"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	pkglogger "github.com/Uncensored-Developer/datalk/apps/backend/pkg/logger"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/api"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/users"
)

type Users struct {
	API *api.Api
}

func New(cfg config.Config, conn *sql.DB) Users {
	logger := pkglogger.SetupLogger(cfg)
	usersApi := users.NewService(conn, logger)
	return Users{
		API: api.New(logger, usersApi),
	}
}
