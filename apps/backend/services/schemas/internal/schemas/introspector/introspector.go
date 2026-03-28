package introspector

import (
	"context"
	"time"

	"github.com/mdobak/go-xerrors"
)

var ErrUnsupportedDBKind = xerrors.New("unsupported database kind")

type DBKind string

const (
	DBPostgres DBKind = "postgres"
	DBMySQL    DBKind = "mysql"
	DBCQL      DBKind = "cql"
)

//go:generate go tool with-modfile mockery --name Introspector --outpkg testing --output ./testing --filename generated__introspecter_mocks.go
type Introspector interface {
	Kind() DBKind
	Introspect(ctx context.Context, opts IntrospectOptions) (*Catalog, error)
}

type IntrospectOptions struct {
	IncludeNamespaces []string
	ExcludeNamespaces []string

	IncludeTablesByNamespace map[string][]string
	ExcludeTablesByNamespace map[string][]string

	IncludeViews       bool
	IncludeIndexes     bool
	IncludeForeignKeys bool
	IncludeComments    bool
}

type Catalog struct {
	Kind         DBKind         `json:"kind"`
	DiscoveredAt time.Time      `json:"discovered_at"`
	Version      string         `json:"version,omitempty"`
	Namespaces   []Namespace    `json:"namespaces"`
	Extras       map[string]any `json:"extras,omitempty"`
}

type Namespace struct {
	Name   string         `json:"name"`
	Tables []Table        `json:"tables"`
	Views  []View         `json:"views"`
	Extras map[string]any `json:"extras"`
}

type Table struct {
	Name        string       `json:"name"`
	Columns     []Column     `json:"columns"`
	PrimaryKey  []string     `json:"primary_key,omitempty"`
	ForeignKeys []ForeignKey `json:"foreign_keys,omitempty"`
	Indexes     []Index      `json:"indexes,omitempty"`
	Comment     string       `json:"comment,omitempty"`
}

type View struct {
	Name       string         `json:"name"`
	Columns    []Column       `json:"columns,omitempty"`
	Definition string         `json:"definition,omitempty"`
	Comment    string         `json:"comment,omitempty"`
	Extras     map[string]any `json:"extras,omitempty"`
}

type Column struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Nullable bool   `json:"nullable"`
	Position int    `json:"position"`
	Comment  string `json:"comment,omitempty"`
}

type ForeignKey struct {
	Columns      []string `json:"columns"`
	RefNamespace string   `json:"ref_namespace,omitempty"`
	RefTable     string   `json:"ref_table"`
	RefColumns   []string `json:"ref_columns"`
}

type Index struct {
	Name     string   `json:"name"`
	Columns  []string `json:"columns,omitempty"`
	IsUnique bool     `json:"is_unique"`
	Method   string   `json:"method,omitempty"`
}
