package sqlrunner

import (
	"testing"

	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/stretchr/testify/require"
)

func TestValidator_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		databaseKind connectiontypes.Database
		query        string
		expectErr    error
	}{
		{
			name:         "postgres select",
			databaseKind: connectiontypes.DatabasePostgres,
			query:        "select count(*) from users",
		},
		{
			name:         "postgres select with semicolon literal",
			databaseKind: connectiontypes.DatabasePostgres,
			query:        "SELECT ';' AS semicolon_text;",
		},
		{
			name:         "postgres cte",
			databaseKind: connectiontypes.DatabasePostgres,
			query:        "with active_users as (select * from users) select count(*) from active_users",
		},
		{
			name:         "postgres union",
			databaseKind: connectiontypes.DatabasePostgres,
			query:        "select id from users union select user_id from subscriptions",
		},
		{
			name:         "postgres empty query",
			databaseKind: connectiontypes.DatabasePostgres,
			query:        "",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "postgres multiple statements",
			databaseKind: connectiontypes.DatabasePostgres,
			query:        "select * from users; select * from subscriptions",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "postgres insert",
			databaseKind: connectiontypes.DatabasePostgres,
			query:        "insert into users(id) values (1)",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "postgres update",
			databaseKind: connectiontypes.DatabasePostgres,
			query:        "update users set email = 'x'",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "postgres delete",
			databaseKind: connectiontypes.DatabasePostgres,
			query:        "delete from users",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "postgres create temp",
			databaseKind: connectiontypes.DatabasePostgres,
			query:        "create temp table x as select 1",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "postgres statement after comment",
			databaseKind: connectiontypes.DatabasePostgres,
			query:        "select * from users; -- comment\n delete from users",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "postgres unterminated string",
			databaseKind: connectiontypes.DatabasePostgres,
			query:        "select * from users where name = 'unterminated",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "postgres data modifying cte",
			databaseKind: connectiontypes.DatabasePostgres,
			query:        "with deleted_users as (delete from users returning id) select id from deleted_users",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "postgres locking select",
			databaseKind: connectiontypes.DatabasePostgres,
			query:        "select * from users for update",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "postgres select into",
			databaseKind: connectiontypes.DatabasePostgres,
			query:        "select * into temp active_users from users",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "mysql select",
			databaseKind: connectiontypes.DatabaseMySQL,
			query:        "select count(*) from users",
		},
		{
			name:         "mysql select with semicolon literal",
			databaseKind: connectiontypes.DatabaseMySQL,
			query:        "SELECT ';' AS semicolon_text;",
		},
		{
			name:         "mysql cte",
			databaseKind: connectiontypes.DatabaseMySQL,
			query:        "with active_users as (select * from users) select count(*) from active_users",
		},
		{
			name:         "mysql union",
			databaseKind: connectiontypes.DatabaseMySQL,
			query:        "select id from users union select user_id from subscriptions",
		},
		{
			name:         "mysql empty query",
			databaseKind: connectiontypes.DatabaseMySQL,
			query:        "",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "mysql multiple statements",
			databaseKind: connectiontypes.DatabaseMySQL,
			query:        "select * from users; select * from subscriptions",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "mysql insert",
			databaseKind: connectiontypes.DatabaseMySQL,
			query:        "insert into users(id) values (1)",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "mysql update",
			databaseKind: connectiontypes.DatabaseMySQL,
			query:        "update users set email = 'x'",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "mysql delete",
			databaseKind: connectiontypes.DatabaseMySQL,
			query:        "delete from users",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "mysql create temp",
			databaseKind: connectiontypes.DatabaseMySQL,
			query:        "create temporary table x as select 1",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "mysql unterminated string",
			databaseKind: connectiontypes.DatabaseMySQL,
			query:        "select * from users where name = 'unterminated",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "mysql select into outfile",
			databaseKind: connectiontypes.DatabaseMySQL,
			query:        "select * from users into outfile '/tmp/users.csv'",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "mysql locking select for update",
			databaseKind: connectiontypes.DatabaseMySQL,
			query:        "select * from users for update",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "mysql locking select share mode",
			databaseKind: connectiontypes.DatabaseMySQL,
			query:        "select * from users lock in share mode",
			expectErr:    chaterrors.ErrInvalidSQL,
		},
		{
			name:         "unsupported database",
			databaseKind: connectiontypes.DatabaseCQL,
			query:        "select 1",
			expectErr:    chaterrors.ErrUnsupportedDatabaseKind,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := NewValidator().Validate(tc.databaseKind, tc.query)

			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
