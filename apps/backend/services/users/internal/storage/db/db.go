package db

import (
	"context"
	"database/sql"
	stderrors "errors"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/common"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/info"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/models"
	userauth "github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/auth"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/lib/pq"
	"github.com/mdobak/go-xerrors"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/im"
)

type Storage struct {
	*common.Storage
}

const singleOwnerIndexName = "users_single_owner_idx"

func NewStorage(conn *sql.DB) *Storage {
	return &Storage{
		common.NewStorage("users", conn),
	}
}

func (s *Storage) UpsertUser(ctx context.Context, user *users.User) error {
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = time.Now().UTC()
	}
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
				info.Users.Columns.MustChangePassword.Name,
				info.Users.Columns.UpdatedAt.Name,
			)),
	).One(ctx, s.Executor(ctx))
	if err != nil {
		if isUniqueViolation(err, singleOwnerIndexName) {
			return storage.ErrOwnerAlreadyExists
		}
		return err
	}

	upsertedUser, err := userFromDB(dbUser)
	if err != nil {
		return xerrors.Newf("failed to map db user to user: %w", err)
	}

	*user = *upsertedUser
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

	dbUsers, err := models.Users.Query(queryMods...).All(ctx, s.Executor(ctx))
	if err != nil {
		return nil, err
	}
	return usersFromDB(dbUsers)
}

func (s *Storage) InsertRefreshToken(ctx context.Context, token *userauth.RefreshToken) error {
	tokenSetter := refreshTokenToDB(token)
	dbToken, err := models.RefreshTokens.Insert(tokenSetter).One(ctx, s.Executor(ctx))
	if err != nil {
		return err
	}

	insertedToken, err := refreshTokenFromDB(dbToken)
	if err != nil {
		return xerrors.Newf("failed to map db refresh token: %w", err)
	}

	*token = *insertedToken
	return nil
}

func (s *Storage) GetRefreshToken(ctx context.Context, tokenHash string) (*userauth.RefreshToken, error) {
	dbToken, err := models.RefreshTokens.Query(
		models.SelectWhere.RefreshTokens.TokenHash.EQ(tokenHash),
	).One(ctx, s.Executor(ctx))
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrRefreshTokenNotFound
		}
		return nil, err
	}

	token, err := refreshTokenFromDB(dbToken)
	if err != nil {
		return nil, xerrors.Newf("failed to map db refresh token: %w", err)
	}
	return token, nil
}

func (s *Storage) RevokeRefreshToken(ctx context.Context, tokenHash string, at time.Time) error {
	tokenSetter := refreshTokenRevokeToDB(at)
	revokedTokens, err := models.RefreshTokens.Update(
		tokenSetter.UpdateMod(),
		models.UpdateWhere.RefreshTokens.TokenHash.EQ(tokenHash),
		models.UpdateWhere.RefreshTokens.RevokedAt.IsNull(),
	).All(ctx, s.Executor(ctx))
	if err != nil {
		return err
	}
	if len(revokedTokens) == 0 {
		return storage.ErrRefreshTokenNotRevoked
	}
	return nil
}

func isUniqueViolation(err error, constraint string) bool {
	var pqErr *pq.Error
	if !stderrors.As(err, &pqErr) {
		return false
	}
	return pqErr.Code == "23505" && pqErr.Constraint == constraint
}
