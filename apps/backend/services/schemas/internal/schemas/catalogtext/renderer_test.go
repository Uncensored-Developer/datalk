package catalogtext

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/introspector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/schema_sample.txt
var schemaSampleText string

func TestRenderCatalog(t *testing.T) {
	t.Run("matches schema txt sample", func(t *testing.T) {
		catalog := &introspector.Catalog{
			Kind: introspector.DBPostgres,
			Namespaces: []introspector.Namespace{
				{
					Name: "testatlas_storage",
					Tables: []introspector.Table{
						{
							Name:       "schema_snapshots",
							PrimaryKey: []string{"id"},
							Comment:    "Stores schema snapshot versions",
							Columns: []introspector.Column{
								{Name: "id", DataType: "serial", Nullable: false, Position: 1},
								{Name: "connection_id", DataType: "integer", Nullable: false, Position: 2},
								{Name: "namespace_id", DataType: "integer", Nullable: false, Position: 3},
								{Name: "schema_hash", DataType: "text", Nullable: false, Position: 4, Comment: "Hash of the schema snapshot"},
								{Name: "slice_json", DataType: "jsonb", Nullable: false, Position: 5, Comment: "Normalized schema JSON"},
								{Name: "status", DataType: "text", Nullable: false, Position: 6},
								{Name: "error_message", DataType: "text", Nullable: true, Position: 7},
								{Name: "introspected_at", DataType: "timestamp with time zone", Nullable: false, Position: 8},
							},
							ForeignKeys: []introspector.ForeignKey{
								{
									Columns:      []string{"connection_id"},
									RefNamespace: "testatlas_storage",
									RefTable:     "connections",
									RefColumns:   []string{"id"},
								},
								{
									Columns:      []string{"namespace_id"},
									RefNamespace: "testatlas_storage",
									RefTable:     "connection_namespaces",
									RefColumns:   []string{"id"},
								},
							},
							Indexes: []introspector.Index{
								{
									Name:     "schema_snapshots_connection_id_schema_hash_key",
									Columns:  []string{"connection_id", "schema_hash"},
									IsUnique: true,
									Method:   "btree",
								},
								{
									Name:    "schema_snapshots_latest_complete_idx",
									Columns: []string{"connection_id", "namespace_id", "introspected_at"},
									Method:  "btree",
								},
							},
						},
					},
				},
			},
		}

		rendered, err := RenderCatalog(catalog)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimRight(schemaSampleText, "\n"), rendered)
	})

	t.Run("sorts namespaces and objects and omits empty sections", func(t *testing.T) {
		catalog := &introspector.Catalog{
			Kind: introspector.DBPostgres,
			Namespaces: []introspector.Namespace{
				{
					Name: "zeta",
					Tables: []introspector.Table{
						{Name: "beta"},
					},
					Views: []introspector.View{
						{Name: "alpha_view"},
					},
				},
				{
					Name: "alpha",
					Tables: []introspector.Table{
						{
							Name:       "users",
							Comment:    "Application users",
							PrimaryKey: []string{"id"},
							Columns: []introspector.Column{
								{Name: "created_at", DataType: "timestamp", Nullable: false},
								{Name: "email", DataType: "text", Nullable: false, Position: 2},
								{Name: "id", DataType: "uuid", Nullable: false, Position: 1},
							},
							ForeignKeys: []introspector.ForeignKey{
								{Columns: []string{"account_id"}, RefNamespace: "alpha", RefTable: "accounts", RefColumns: []string{"id"}},
								{Columns: []string{"org_id"}, RefNamespace: "alpha", RefTable: "organizations", RefColumns: []string{"id"}},
							},
							Indexes: []introspector.Index{
								{Name: "users_lookup_idx", Columns: []string{"email"}},
								{Name: "users_email_key", Columns: []string{"email"}, IsUnique: true, Method: "btree"},
							},
						},
					},
					Views: []introspector.View{
						{
							Name: "user_emails",
							Columns: []introspector.Column{
								{Name: "email", DataType: "text", Nullable: false, Position: 1},
							},
						},
					},
				},
			},
		}

		rendered, err := RenderCatalog(catalog)
		require.NoError(t, err)

		assert.Equal(t, `database_kind: postgres
namespace: alpha
object_type: table
object_name: users
qualified_name: alpha.users

comment: Application users

primary_key: id

columns:
1. id | type=uuid | nullable=false
2. email | type=text | nullable=false
3. created_at | type=timestamp | nullable=false

foreign_keys:
- columns=account_id | references=alpha.accounts(id)
- columns=org_id | references=alpha.organizations(id)

indexes:
- name=users_email_key | unique=true | method=btree | columns=email
- name=users_lookup_idx | unique=false | method= | columns=email

Purpose Hints:
- Application users

database_kind: postgres
namespace: alpha
object_type: view
object_name: user_emails
qualified_name: alpha.user_emails

columns:
1. email | type=text | nullable=false

database_kind: postgres
namespace: zeta
object_type: table
object_name: beta
qualified_name: zeta.beta

database_kind: postgres
namespace: zeta
object_type: view
object_name: alpha_view
qualified_name: zeta.alpha_view`, rendered)
	})

	t.Run("returns error for nil catalog", func(t *testing.T) {
		rendered, err := RenderCatalog(nil)
		require.ErrorIs(t, err, ErrNilCatalog)
		assert.Empty(t, rendered)
	})
}
