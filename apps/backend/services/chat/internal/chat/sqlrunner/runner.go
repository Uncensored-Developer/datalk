package sqlrunner

import (
	"context"
	"database/sql"
	"net/url"
	"strings"
	"time"

	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/mdobak/go-xerrors"
)

const (
	defaultQueryTimeout = 30 * time.Second
	defaultRowLimit     = 100
)

type RunOptions struct {
	Timeout  time.Duration
	RowLimit int
}

//go:generate go tool with-modfile mockery --name SQLRunner --outpkg testing --output ./testing --filename generated__sql_runner_mocks.go
type SQLRunner interface {
	Run(ctx context.Context, connection connectiontypes.Connection, query string, options RunOptions) (*chattype.QueryResult, error)
}

type Runner struct {
	validator *Validator
	openDB    func(driverName, dataSourceName string) (*sql.DB, error)
}

func NewRunner() *Runner {
	return &Runner{
		validator: NewValidator(),
		openDB:    sql.Open,
	}
}

func (r *Runner) Run(ctx context.Context, connection connectiontypes.Connection, query string, options RunOptions) (*chattype.QueryResult, error) {
	if err := r.validator.Validate(connection.Database, query); err != nil {
		return nil, err
	}
	if connection.DSN == "" {
		return nil, xerrors.Newf("connection dsn is required: %w", chaterrors.ErrMessageExecutionFailed)
	}
	driverName, err := driverNameForDatabase(connection.Database)
	if err != nil {
		return nil, err
	}
	dataSourceName, err := dataSourceNameForDatabase(connection.Database, connection.DSN)
	if err != nil {
		return nil, err
	}

	timeout := options.Timeout
	if timeout <= 0 {
		timeout = defaultQueryTimeout
	}
	rowLimit := options.RowLimit
	if rowLimit <= 0 {
		rowLimit = defaultRowLimit
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	db, err := r.openDB(driverName, dataSourceName)
	if err != nil {
		return nil, xerrors.Newf("failed to open query connection: %w", err)
	}
	defer db.Close()

	// Query is done in a read-only transaction as a runtime guard.
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, xerrors.Newf("failed to start read-only query transaction: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, xerrors.Newf("failed to execute query: %w", err)
	}
	defer rows.Close()

	result, err := collectQueryResult(sqlRows{rows: rows}, rowLimit)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, xerrors.Newf("failed to commit read-only query transaction: %w", err)
	}

	return result, nil
}

func driverNameForDatabase(databaseKind connectiontypes.Database) (string, error) {
	switch databaseKind {
	case connectiontypes.DatabasePostgres:
		return "postgres", nil
	case connectiontypes.DatabaseMySQL:
		return "mysql", nil
	default:
		return "", xerrors.Newf("%s: %w", databaseKind, chaterrors.ErrUnsupportedDatabaseKind)
	}
}

func dataSourceNameForDatabase(databaseKind connectiontypes.Database, dsn string) (string, error) {
	switch databaseKind {
	case connectiontypes.DatabasePostgres:
		return dsn, nil
	case connectiontypes.DatabaseMySQL:
		return normalizeMySQLDataSourceName(dsn)
	default:
		return "", xerrors.Newf("%s: %w", databaseKind, chaterrors.ErrUnsupportedDatabaseKind)
	}
}

func normalizeMySQLDataSourceName(dsn string) (string, error) {
	if !strings.HasPrefix(dsn, "mysql://") {
		return dsn, nil
	}

	parsed, err := url.Parse(dsn)
	if err != nil {
		return "", xerrors.Newf("failed to parse mysql dsn: %w", err)
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

type queryRows interface {
	Columns() ([]string, error)
	ColumnTypes() ([]columnType, error)
	Next() bool
	Scan(dest ...any) error
	Err() error
}

type columnType interface {
	DatabaseTypeName() string
}

type sqlRows struct {
	rows *sql.Rows
}

func (r sqlRows) Columns() ([]string, error) {
	return r.rows.Columns()
}

func (r sqlRows) ColumnTypes() ([]columnType, error) {
	dbTypes, err := r.rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	columnTypes := make([]columnType, 0, len(dbTypes))
	for _, dbType := range dbTypes {
		columnTypes = append(columnTypes, dbType)
	}
	return columnTypes, nil
}

func (r sqlRows) Next() bool {
	return r.rows.Next()
}

func (r sqlRows) Scan(dest ...any) error {
	return r.rows.Scan(dest...)
}

func (r sqlRows) Err() error {
	return r.rows.Err()
}

func collectQueryResult(rows queryRows, rowLimit int) (*chattype.QueryResult, error) {
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, xerrors.Newf("failed to read query columns: %w", err)
	}

	dbColumnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, xerrors.Newf("failed to read query column types: %w", err)
	}

	columns := make([]chattype.ResultColumn, 0, len(columnNames))
	for index, name := range columnNames {
		dataType := ""
		if index < len(dbColumnTypes) {
			dataType = dbColumnTypes[index].DatabaseTypeName()
		}
		columns = append(columns, chattype.ResultColumn{
			Name:     name,
			DataType: dataType,
		})
	}

	resultRows := make([]map[string]any, 0)
	truncated := false
	for rows.Next() {
		values := make([]any, len(columnNames))
		dest := make([]any, len(columnNames))
		for index := range values {
			dest[index] = &values[index]
		}

		if err := rows.Scan(dest...); err != nil {
			return nil, xerrors.Newf("failed to scan query row: %w", err)
		}
		if len(resultRows) >= rowLimit {
			truncated = true
			break
		}

		row := make(map[string]any, len(columnNames))
		for index, columnName := range columnNames {
			row[columnName] = normalizeResultValue(values[index])
		}
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, xerrors.Newf("failed while reading query rows: %w", err)
	}

	return &chattype.QueryResult{
		Columns:   columns,
		Rows:      resultRows,
		RowCount:  int32(len(resultRows)),
		Truncated: truncated,
		Kind:      inferQueryResultKind(len(columns), len(resultRows)),
	}, nil
}

func normalizeResultValue(value any) any {
	switch typedValue := value.(type) {
	case []byte:
		return string(typedValue)
	default:
		return typedValue
	}
}

func inferQueryResultKind(columnCount, rowCount int) chattype.QueryResultKind {
	switch {
	case rowCount == 0:
		return chattype.QueryResultKindEmpty
	case rowCount == 1 && columnCount == 1:
		return chattype.QueryResultKindScalar
	case rowCount == 1:
		return chattype.QueryResultKindRecord
	default:
		return chattype.QueryResultKindTable
	}
}
