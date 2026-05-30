package sqlrunner

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeColumnType string

func (f fakeColumnType) DatabaseTypeName() string {
	return string(f)
}

type fakeRows struct {
	columns     []string
	columnTypes []columnType
	rows        [][]any
	index       int
	err         error
	scanErr     error
}

func (f *fakeRows) Columns() ([]string, error) {
	return f.columns, nil
}

func (f *fakeRows) ColumnTypes() ([]columnType, error) {
	return f.columnTypes, nil
}

func (f *fakeRows) Next() bool {
	return f.index < len(f.rows)
}

func (f *fakeRows) Scan(dest ...any) error {
	if f.scanErr != nil {
		return f.scanErr
	}
	if f.index >= len(f.rows) {
		return errors.New("scan called after rows exhausted")
	}
	for index, value := range f.rows[f.index] {
		typedDest := dest[index].(*any)
		*typedDest = value
	}
	f.index++
	return nil
}

func (f *fakeRows) Err() error {
	return f.err
}

func TestCollectQueryResult_TableWithNullsAndBytes(t *testing.T) {
	t.Parallel()

	result, err := collectQueryResult(&fakeRows{
		columns:     []string{"id", "email", "deleted_at"},
		columnTypes: []columnType{fakeColumnType("INT8"), fakeColumnType("TEXT"), fakeColumnType("TIMESTAMPTZ")},
		rows: [][]any{
			{int64(1), []byte("a@example.com"), nil},
			{int64(2), []byte("b@example.com"), time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)},
		},
	}, 10)

	require.NoError(t, err)
	assert.Equal(t, chattype.QueryResultKindTable, result.Kind)
	assert.False(t, result.Truncated)
	assert.Equal(t, int32(2), result.RowCount)
	assert.Equal(t, []chattype.ResultColumn{
		{Name: "id", DataType: "INT8"},
		{Name: "email", DataType: "TEXT"},
		{Name: "deleted_at", DataType: "TIMESTAMPTZ"},
	}, result.Columns)
	assert.Equal(t, "a@example.com", result.Rows[0]["email"])
	assert.Nil(t, result.Rows[0]["deleted_at"])
}

func TestCollectQueryResult_Kinds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		rows           *fakeRows
		expectKind     chattype.QueryResultKind
		expectRowCount int32
		expectValue    any
		expectColumn   string
	}{
		{
			name: "scalar aggregate",
			rows: &fakeRows{
				columns:     []string{"subscriber_count"},
				columnTypes: []columnType{fakeColumnType("INT8")},
				rows:        [][]any{{int64(42)}},
			},
			expectKind:     chattype.QueryResultKindScalar,
			expectRowCount: 1,
			expectColumn:   "subscriber_count",
			expectValue:    int64(42),
		},
		{
			name: "record",
			rows: &fakeRows{
				columns:     []string{"id", "email"},
				columnTypes: []columnType{fakeColumnType("INT8"), fakeColumnType("TEXT")},
				rows:        [][]any{{int64(1), "a@example.com"}},
			},
			expectKind:     chattype.QueryResultKindRecord,
			expectRowCount: 1,
			expectColumn:   "email",
			expectValue:    "a@example.com",
		},
		{
			name: "empty",
			rows: &fakeRows{
				columns:     []string{"id"},
				columnTypes: []columnType{fakeColumnType("INT8")},
				rows:        nil,
			},
			expectKind:     chattype.QueryResultKindEmpty,
			expectRowCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := collectQueryResult(tc.rows, 10)

			require.NoError(t, err)
			assert.Equal(t, tc.expectKind, result.Kind)
			assert.Equal(t, tc.expectRowCount, result.RowCount)
			if tc.expectColumn != "" {
				require.NotEmpty(t, result.Rows)
				assert.Equal(t, tc.expectValue, result.Rows[0][tc.expectColumn])
			}
		})
	}
}

func TestCollectQueryResult_TruncatesRows(t *testing.T) {
	t.Parallel()

	rows := &fakeRows{
		columns:     []string{"id"},
		columnTypes: []columnType{fakeColumnType("INT8")},
		rows:        [][]any{{int64(1)}, {int64(2)}, {int64(3)}, {int64(4)}, {int64(5)}},
	}

	result, err := collectQueryResult(rows, 2)

	require.NoError(t, err)
	assert.True(t, result.Truncated)
	assert.Equal(t, int32(2), result.RowCount)
	require.Len(t, result.Rows, 2)
	assert.Equal(t, int64(1), result.Rows[0]["id"])
	assert.Equal(t, int64(2), result.Rows[1]["id"])
	assert.Equal(t, 3, rows.index)
}

func TestDriverNameForDatabase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		databaseKind connectiontypes.Database
		expect       string
		expectErr    error
	}{
		{
			name:         "postgres",
			databaseKind: connectiontypes.DatabasePostgres,
			expect:       "postgres",
		},
		{
			name:         "mysql",
			databaseKind: connectiontypes.DatabaseMySQL,
			expect:       "mysql",
		},
		{
			name:         "unsupported",
			databaseKind: connectiontypes.DatabaseCQL,
			expectErr:    chaterrors.ErrUnsupportedDatabaseKind,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := driverNameForDatabase(tc.databaseKind)

			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expect, got)
		})
	}
}

func TestDataSourceNameForDatabase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		databaseKind connectiontypes.Database
		dsn          string
		expect       string
	}{
		{
			name:         "postgres keeps original dsn",
			databaseKind: connectiontypes.DatabasePostgres,
			dsn:          "postgres://user:pass@localhost:5432/app?sslmode=disable",
			expect:       "postgres://user:pass@localhost:5432/app?sslmode=disable",
		},
		{
			name:         "mysql native dsn",
			databaseKind: connectiontypes.DatabaseMySQL,
			dsn:          "user:pass@tcp(localhost:3306)/app?parseTime=true",
			expect:       "user:pass@tcp(localhost:3306)/app?parseTime=true",
		},
		{
			name:         "mysql url dsn",
			databaseKind: connectiontypes.DatabaseMySQL,
			dsn:          "mysql://user:pass@localhost:3306/app?parseTime=true",
			expect:       "user:pass@tcp(localhost:3306)/app?parseTime=true",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := dataSourceNameForDatabase(tc.databaseKind, tc.dsn)

			require.NoError(t, err)
			assert.Equal(t, tc.expect, got)
		})
	}
}

func TestRunnerErrorClassification(t *testing.T) {
	t.Parallel()

	queryErr := errors.New(`pq: column "missing" does not exist`)
	driverCancelErr := errors.New("pq: canceling statement due to user request")
	openErr := errors.New("dial tcp failed")
	beginErr := errors.New("read-only transactions are not supported")
	commitErr := errors.New("commit failed")

	tests := []struct {
		name             string
		runner           *Runner
		connection       connectiontypes.Connection
		query            string
		expectKind       ErrorKind
		expectCorrection bool
		expectErr        error
		options          RunOptions
	}{
		{
			name: "validation error is correction eligible",
			connection: connectiontypes.Connection{
				Database: connectiontypes.DatabasePostgres,
				DSN:      "unused",
			},
			query:            "DELETE FROM users",
			expectKind:       ErrorKindValidation,
			expectCorrection: true,
			expectErr:        chaterrors.ErrInvalidSQL,
		},
		{
			name: "missing dsn is runtime",
			connection: connectiontypes.Connection{
				Database: connectiontypes.DatabasePostgres,
			},
			query:            "SELECT 1",
			expectKind:       ErrorKindRuntime,
			expectCorrection: false,
			expectErr:        chaterrors.ErrMessageExecutionFailed,
		},
		{
			name: "open failure is runtime",
			runner: &Runner{
				validator: NewValidator(),
				openDB: func(string, string) (*sql.DB, error) {
					return nil, openErr
				},
			},
			connection: connectiontypes.Connection{
				Database: connectiontypes.DatabasePostgres,
				DSN:      "postgres://user:pass@localhost/app",
			},
			query:            "SELECT 1",
			expectKind:       ErrorKindRuntime,
			expectCorrection: false,
			expectErr:        openErr,
		},
		{
			name: "begin transaction failure is runtime",
			runner: runnerWithStubDB(t, stubSQLConfig{
				beginErr: beginErr,
			}),
			connection: connectiontypes.Connection{
				Database: connectiontypes.DatabasePostgres,
				DSN:      "postgres://user:pass@localhost/app",
			},
			query:            "SELECT 1",
			expectKind:       ErrorKindRuntime,
			expectCorrection: false,
			expectErr:        beginErr,
		},
		{
			name: "query execution failure is correction eligible",
			runner: runnerWithStubDB(t, stubSQLConfig{
				queryErr: queryErr,
			}),
			connection: connectiontypes.Connection{
				Database: connectiontypes.DatabasePostgres,
				DSN:      "postgres://user:pass@localhost/app",
			},
			query:            "SELECT missing FROM users",
			expectKind:       ErrorKindQueryExecution,
			expectCorrection: true,
			expectErr:        queryErr,
		},
		{
			name: "query context cancellation is runtime",
			runner: runnerWithStubDB(t, stubSQLConfig{
				queryErr: context.DeadlineExceeded,
			}),
			connection: connectiontypes.Connection{
				Database: connectiontypes.DatabasePostgres,
				DSN:      "postgres://user:pass@localhost/app",
			},
			query:            "SELECT pg_sleep(10)",
			expectKind:       ErrorKindRuntime,
			expectCorrection: false,
			expectErr:        context.DeadlineExceeded,
		},
		{
			name: "driver specific cancellation after context expiry is runtime",
			runner: runnerWithStubDB(t, stubSQLConfig{
				queryErr:       driverCancelErr,
				expireQueryCtx: true,
			}),
			connection: connectiontypes.Connection{
				Database: connectiontypes.DatabasePostgres,
				DSN:      "postgres://user:pass@localhost/app",
			},
			query:            "SELECT pg_sleep(10)",
			expectKind:       ErrorKindRuntime,
			expectCorrection: false,
			expectErr:        context.DeadlineExceeded,
			options:          RunOptions{Timeout: time.Nanosecond, RowLimit: 10},
		},
		{
			name: "commit failure is runtime",
			runner: runnerWithStubDB(t, stubSQLConfig{
				commitErr: commitErr,
			}),
			connection: connectiontypes.Connection{
				Database: connectiontypes.DatabasePostgres,
				DSN:      "postgres://user:pass@localhost/app",
			},
			query:            "SELECT 1",
			expectKind:       ErrorKindRuntime,
			expectCorrection: false,
			expectErr:        commitErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runner := tc.runner
			if runner == nil {
				runner = NewRunner()
			}

			options := tc.options
			if options.RowLimit == 0 {
				options.RowLimit = 10
			}
			_, err := runner.Run(t.Context(), tc.connection, tc.query, options)

			require.Error(t, err)
			require.ErrorIs(t, err, tc.expectErr)
			assert.Equal(t, tc.expectCorrection, IsCorrectionEligible(err))
			kind, ok := Kind(err)
			require.True(t, ok)
			assert.Equal(t, tc.expectKind, kind)
		})
	}
}

func TestRunnerIntegration_PostgresReadOnlyExecution(t *testing.T) {
	runner, cfg := requireIntegrationRunner(t, "sqlrunner")
	_, err := runner.Conn.ExecContext(t.Context(), `
		CREATE TABLE runner_subscriptions (
			id SERIAL PRIMARY KEY,
			subscribed_at TIMESTAMPTZ NOT NULL
		);
		INSERT INTO runner_subscriptions (subscribed_at) VALUES
			('2026-05-03T10:00:00Z'),
			('2026-05-12T10:00:00Z');
	`)
	require.NoError(t, err)

	connection := connectiontypes.Connection{
		Database: connectiontypes.DatabasePostgres,
		DSN:      sqlRunnerIntegrationPostgresDSN(cfg, runner.Schema),
	}

	result, err := NewRunner().Run(t.Context(), connection, "SELECT count(*)::int AS total FROM runner_subscriptions", RunOptions{RowLimit: 10})
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)
	assert.EqualValues(t, int64(2), result.Rows[0]["total"])
	assert.False(t, result.Truncated)

	_, err = NewRunner().Run(t.Context(), connection, "DELETE FROM runner_subscriptions", RunOptions{RowLimit: 10})
	require.ErrorIs(t, err, chaterrors.ErrInvalidSQL)

	var remaining int
	require.NoError(t, runner.Conn.QueryRowContext(t.Context(), "SELECT count(*) FROM runner_subscriptions").Scan(&remaining))
	assert.Equal(t, 2, remaining)
}

const stubSQLDriverName = "datalk_sqlrunner_stub"

var (
	stubSQLRegisterOnce sync.Once
	stubSQLMu           sync.Mutex
	stubSQLConfigs      = map[string]stubSQLConfig{}
)

type stubSQLConfig struct {
	beginErr       error
	queryErr       error
	expireQueryCtx bool
	commitErr      error
}

func runnerWithStubDB(t *testing.T, cfg stubSQLConfig) *Runner {
	t.Helper()
	registerStubSQLDriver()

	key := t.Name() + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	stubSQLMu.Lock()
	stubSQLConfigs[key] = cfg
	stubSQLMu.Unlock()
	t.Cleanup(func() {
		stubSQLMu.Lock()
		delete(stubSQLConfigs, key)
		stubSQLMu.Unlock()
	})

	return &Runner{
		validator: NewValidator(),
		openDB: func(string, string) (*sql.DB, error) {
			return sql.Open(stubSQLDriverName, key)
		},
	}
}

func registerStubSQLDriver() {
	stubSQLRegisterOnce.Do(func() {
		sql.Register(stubSQLDriverName, stubSQLDriver{})
	})
}

type stubSQLDriver struct{}

func (d stubSQLDriver) Open(name string) (driver.Conn, error) {
	stubSQLMu.Lock()
	cfg := stubSQLConfigs[name]
	stubSQLMu.Unlock()
	return &stubSQLConn{cfg: cfg}, nil
}

type stubSQLConn struct {
	cfg stubSQLConfig
}

func (c *stubSQLConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not implemented")
}

func (c *stubSQLConn) Close() error {
	return nil
}

func (c *stubSQLConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *stubSQLConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	if c.cfg.beginErr != nil {
		return nil, c.cfg.beginErr
	}
	return &stubSQLTx{commitErr: c.cfg.commitErr}, nil
}

func (c *stubSQLConn) QueryContext(ctx context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	if c.cfg.expireQueryCtx {
		<-ctx.Done()
	}
	if c.cfg.queryErr != nil {
		return nil, c.cfg.queryErr
	}
	return &stubSQLRows{
		columns: []string{"value"},
		rows:    [][]driver.Value{{int64(1)}},
	}, nil
}

type stubSQLTx struct {
	commitErr error
}

func (tx *stubSQLTx) Commit() error {
	return tx.commitErr
}

func (tx *stubSQLTx) Rollback() error {
	return nil
}

type stubSQLRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func (r *stubSQLRows) Columns() []string {
	return r.columns
}

func (r *stubSQLRows) Close() error {
	return nil
}

func (r *stubSQLRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.index])
	r.index++
	return nil
}

func sqlRunnerIntegrationPostgresDSN(cfg config.Config, schema string) string {
	return fmt.Sprintf(
		"user=%s password=%s host=%s port=%d dbname=%s sslmode=%s search_path=test%s",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBSSLMode,
		schema,
	)
}
