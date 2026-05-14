package sqlrunner

import (
	"strings"

	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/mdobak/go-xerrors"
)

type Validator struct {
	mysql    mysqlValidator
	postgres postgresValidator
}

func NewValidator() *Validator {
	return &Validator{
		mysql:    newMySQLValidator(),
		postgres: postgresValidator{},
	}
}

func (v *Validator) Validate(databaseKind connectiontypes.Database, query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return invalidSQL("query cannot be empty")
	}

	switch databaseKind {
	case connectiontypes.DatabasePostgres:
		return v.postgres.Validate(query)
	case connectiontypes.DatabaseMySQL:
		return v.mysql.Validate(query)
	default:
		return xerrors.Newf("%s: %w", databaseKind, chaterrors.ErrUnsupportedDatabaseKind)
	}
}

func invalidSQL(message string) error {
	return xerrors.Newf("%s: %w", message, chaterrors.ErrInvalidSQL)
}

func invalidSQLf(format string, args ...any) error {
	return xerrors.Newf(format+": %w", append(args, chaterrors.ErrInvalidSQL)...)
}
