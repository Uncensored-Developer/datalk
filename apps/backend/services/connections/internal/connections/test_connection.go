package connections

import (
	"context"
	"database/sql"
	"net/url"
	"strings"
	"time"

	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	connectionerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/errors"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/mdobak/go-xerrors"
)

const (
	defaultConnectionTestTimeout = 10 * time.Second
	connectionTestMaxOpenConns   = 1
	connectionTestMaxIdleConns   = 1
)

type TestConnection struct {
	Database connectiontypes.Database
	DSN      string
}

func (t *TestConnection) Validate() error {
	if t.Database == "" {
		return xerrors.Newf("database is required: %w", connectionerrors.ErrInvalidConnectionInput)
	}
	if strings.TrimSpace(t.DSN) == "" {
		return xerrors.Newf("dsn is required: %w", connectionerrors.ErrInvalidConnectionInput)
	}
	switch t.Database {
	case connectiontypes.DatabasePostgres, connectiontypes.DatabaseMySQL:
		return nil
	default:
		return xerrors.Newf("%s: %w", t.Database, connectionerrors.ErrUnsupportedConnection)
	}
}

func (s *Service) TestConnection(ctx context.Context, params TestConnection) error {
	if err := params.Validate(); err != nil {
		return err
	}

	driverName, dataSourceName, err := testDataSource(params.Database, params.DSN)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, defaultConnectionTestTimeout)
	defer cancel()

	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return xerrors.Newf("failed to open connection: %w: %w", err, connectionerrors.ErrConnectionTestFailed)
	}
	defer db.Close()

	db.SetConnMaxLifetime(defaultConnectionTestTimeout)
	db.SetMaxOpenConns(connectionTestMaxOpenConns)
	db.SetMaxIdleConns(connectionTestMaxIdleConns)

	if err := db.PingContext(ctx); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return xerrors.Newf("failed to connect before timeout: %w: %w", ctxErr, connectionerrors.ErrConnectionTestFailed)
		}
		return xerrors.Newf("failed to connect: %w: %w", err, connectionerrors.ErrConnectionTestFailed)
	}

	return nil
}

func testDataSource(database connectiontypes.Database, dsn string) (string, string, error) {
	switch database {
	case connectiontypes.DatabasePostgres:
		return "postgres", dsn, nil
	case connectiontypes.DatabaseMySQL:
		normalized, err := normalizeMySQLTestDataSourceName(dsn)
		if err != nil {
			return "", "", err
		}
		return "mysql", normalized, nil
	default:
		return "", "", xerrors.Newf("%s: %w", database, connectionerrors.ErrUnsupportedConnection)
	}
}

func normalizeMySQLTestDataSourceName(dsn string) (string, error) {
	if !strings.HasPrefix(dsn, "mysql://") {
		return dsn, nil
	}

	parsed, err := url.Parse(dsn)
	if err != nil {
		return "", xerrors.Newf("failed to parse mysql dsn: %w: %w", err, connectionerrors.ErrInvalidConnectionInput)
	}

	username := parsed.User.Username()
	password, hasPassword := parsed.User.Password()
	auth := username
	if hasPassword {
		auth += ":" + password
	}

	databaseName := strings.TrimPrefix(parsed.EscapedPath(), "/")
	query := parsed.RawQuery
	if query != "" {
		query = "?" + query
	}

	return auth + "@tcp(" + parsed.Host + ")/" + databaseName + query, nil
}
