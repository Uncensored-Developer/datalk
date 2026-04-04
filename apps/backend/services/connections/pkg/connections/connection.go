package connections

import "time"

type Database string

const (
	DatabasePostgres Database = "postgres"
	DatabaseMySQL    Database = "mysql"
	DatabaseCQL      Database = "cql"
)

type Metadata struct {
	IncludeNamespaces []string `json:"include_namespaces"`
	ExcludeNamespaces []string `json:"exclude_namespaces"`

	IncludeTablesByNamespace map[string][]string `json:"include_tables_by_namespace"`
	ExcludeTablesByNamespace map[string][]string `json:"exclude_tables_by_namespace"`

	IncludeViews       bool `json:"include_views"`
	IncludeIndexes     bool `json:"include_indexes"`
	IncludeForeignKeys bool `json:"include_foreign_keys"`
	IncludeComments    bool `json:"include_comments"`
}

type Connection struct {
	ID        int32
	UserID    int32
	Name      string
	Database  Database
	DSN       string
	IsEnabled bool
	Metadata  Metadata
	CreatedAt time.Time
}
