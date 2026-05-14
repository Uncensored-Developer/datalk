package sqlrunner

import (
	"errors"
	"testing"
	"time"

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
