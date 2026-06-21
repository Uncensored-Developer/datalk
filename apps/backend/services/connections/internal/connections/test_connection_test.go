package connections

import (
	"testing"

	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	connectionerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestConnection_Validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		params      TestConnection
		expectError error
	}{
		{
			name:        "missing database",
			params:      TestConnection{DSN: "postgres://example"},
			expectError: connectionerrors.ErrInvalidConnectionInput,
		},
		{
			name:        "missing dsn",
			params:      TestConnection{Database: connectiontypes.DatabasePostgres},
			expectError: connectionerrors.ErrInvalidConnectionInput,
		},
		{
			name:        "unsupported cql",
			params:      TestConnection{Database: connectiontypes.DatabaseCQL, DSN: "cql://example"},
			expectError: connectionerrors.ErrUnsupportedConnection,
		},
		{
			name:   "postgres",
			params: TestConnection{Database: connectiontypes.DatabasePostgres, DSN: "postgres://example"},
		},
		{
			name:   "mysql",
			params: TestConnection{Database: connectiontypes.DatabaseMySQL, DSN: "mysql://example"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.params.Validate()

			if tc.expectError != nil {
				require.ErrorIs(t, err, tc.expectError)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestTestDataSource(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		databaseKind connectiontypes.Database
		dsn          string
		expectDriver string
		expectDSN    string
	}{
		{
			name:         "postgres",
			databaseKind: connectiontypes.DatabasePostgres,
			dsn:          "postgres://user:pass@localhost:5432/app?sslmode=require",
			expectDriver: "postgres",
			expectDSN:    "postgres://user:pass@localhost:5432/app?sslmode=require",
		},
		{
			name:         "mysql native",
			databaseKind: connectiontypes.DatabaseMySQL,
			dsn:          "user:pass@tcp(localhost:3306)/app?parseTime=true",
			expectDriver: "mysql",
			expectDSN:    "user:pass@tcp(localhost:3306)/app?parseTime=true",
		},
		{
			name:         "mysql url",
			databaseKind: connectiontypes.DatabaseMySQL,
			dsn:          "mysql://user:pass@localhost:3306/app?parseTime=true",
			expectDriver: "mysql",
			expectDSN:    "user:pass@tcp(localhost:3306)/app?parseTime=true",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			driverName, dataSourceName, err := testDataSource(tc.databaseKind, tc.dsn)

			require.NoError(t, err)
			assert.Equal(t, tc.expectDriver, driverName)
			assert.Equal(t, tc.expectDSN, dataSourceName)
		})
	}
}
