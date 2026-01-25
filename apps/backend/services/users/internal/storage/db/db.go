package db

import (
	"context"
	"database/sql"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/common"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/info"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/models"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/mdobak/go-xerrors"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/im"
)

type Storage struct {
	*common.Storage
}

func NewStorage(conn *sql.DB) *Storage {
	return &Storage{
		common.NewStorage("users", conn),
	}
}

func (s *Storage) UpsertUser(ctx context.Context, user *users.User) error {
	userSetter := userToDB(user)
	dbUser, err := models.Users.Insert(
		userSetter,
		im.OnConflict(info.Users.Columns.Email.Name).DoUpdate(
			im.SetExcluded(
				info.Users.Columns.Name.Name,
				info.Users.Columns.PasswordHash.Name,
				info.Users.Columns.Role.Name,
				info.Users.Columns.IsActive.Name,
				info.Users.Columns.LastLoginAt.Name,
			)),
	).One(ctx, s.Executor(ctx))
	if err != nil {
		return err
	}

	upsertedUser, err := userFromDB(dbUser)
	if err != nil {
		return xerrors.Newf("failed to map db user to user: %w", err)
	}

	user.ID = upsertedUser.ID
	user.Name = upsertedUser.Name
	user.PasswordHash = upsertedUser.PasswordHash
	user.Role = upsertedUser.Role
	user.IsActive = upsertedUser.IsActive
	user.LastLoginAt = upsertedUser.LastLoginAt
	return nil
}

func (s *Storage) ListUsers(ctx context.Context, params storage.ListUsersParam) ([]*users.User, error) {
	var queryMods []bob.Mod[*dialect.SelectQuery]

	if len(params.ID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.Users.ID.In(params.ID...))
	}

	if len(params.Email) > 0 {
		queryMods = append(queryMods, models.SelectWhere.Users.Email.In(params.Email...))
	}

	fetchedUsers, err := models.Users.Query(queryMods...).All(ctx, s.Executor(ctx))
	if err != nil {
		return nil, err
	}

	return usersFromDB(fetchedUsers)
}
