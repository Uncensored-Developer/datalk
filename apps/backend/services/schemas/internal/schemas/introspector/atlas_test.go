package introspector

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/schema_snapshots_introspect.json
var schemaSnapshotJSONByte []byte

func TestNewPostgres(t *testing.T) {
	postgresIntrospector, err := NewPostgres(runner.Conn)
	require.NoError(t, err)

	namespace := "testatlas_storage"

	t.Run("full testatlas_storage schema introspection", func(t *testing.T) {
		catalog, err := postgresIntrospector.Introspect(t.Context(), IntrospectOptions{
			IncludeNamespaces:  []string{namespace},
			IncludeComments:    true,
			IncludeViews:       true,
			IncludeForeignKeys: true,
			IncludeIndexes:     true,
		})
		require.NoError(t, err)

		require.Equal(t, DBPostgres, catalog.Kind)
		require.Len(t, catalog.Namespaces, 1)
		assert.Equal(t, namespace, catalog.Namespaces[0].Name)
		assert.Len(t, catalog.Namespaces[0].Tables, 16)
	})

	t.Run("include only a few tables introspection", func(t *testing.T) {
		catalog, err := postgresIntrospector.Introspect(t.Context(), IntrospectOptions{
			IncludeNamespaces: []string{namespace},
			IncludeTablesByNamespace: map[string][]string{
				namespace: {"users", "connection_access"},
			},
			IncludeComments:    true,
			IncludeViews:       true,
			IncludeForeignKeys: true,
			IncludeIndexes:     true,
		})
		require.NoError(t, err)

		require.Equal(t, DBPostgres, catalog.Kind)
		require.Len(t, catalog.Namespaces, 1)
		assert.Equal(t, namespace, catalog.Namespaces[0].Name)
		assert.Len(t, catalog.Namespaces[0].Tables, 2)
	})

	t.Run("exclude a few tables introspection", func(t *testing.T) {
		catalog, err := postgresIntrospector.Introspect(t.Context(), IntrospectOptions{
			IncludeNamespaces: []string{namespace},
			ExcludeTablesByNamespace: map[string][]string{
				namespace: {"users", "connection_access"},
			},
			IncludeComments:    true,
			IncludeViews:       true,
			IncludeForeignKeys: true,
			IncludeIndexes:     true,
		})
		require.NoError(t, err)

		require.Equal(t, DBPostgres, catalog.Kind)
		require.Len(t, catalog.Namespaces, 1)
		assert.Equal(t, namespace, catalog.Namespaces[0].Name)
		assert.Len(t, catalog.Namespaces[0].Tables, 14)
	})

	t.Run("schema_snapshots introspection", func(t *testing.T) {
		catalog, err := postgresIntrospector.Introspect(t.Context(), IntrospectOptions{
			IncludeNamespaces: []string{namespace},
			IncludeTablesByNamespace: map[string][]string{
				namespace: {"schema_snapshots"},
			},
			IncludeComments:    true,
			IncludeViews:       true,
			IncludeForeignKeys: true,
			IncludeIndexes:     true,
		})
		require.NoError(t, err)

		var expectedCatalog Catalog
		err = json.Unmarshal(schemaSnapshotJSONByte, &expectedCatalog)
		require.NoError(t, err)

		assert.Exactly(t, &expectedCatalog, catalog)
	})
}
