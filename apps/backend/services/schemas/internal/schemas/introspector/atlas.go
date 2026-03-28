package introspector

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	atlasmysql "ariga.io/atlas/sql/mysql"
	atlaspostgres "ariga.io/atlas/sql/postgres"
	atlasschema "ariga.io/atlas/sql/schema"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/slices"
	"github.com/mdobak/go-xerrors"
)

type Atlas struct {
	kind      DBKind
	inspector atlasschema.Inspector
}

func NewAtlas(kind DBKind, inspector atlasschema.Inspector) (*Atlas, error) {
	if inspector == nil {
		return nil, xerrors.New("inspector cannot be nil")
	}
	switch kind {
	case DBPostgres, DBMySQL:
		return &Atlas{kind: kind, inspector: inspector}, nil
	default:
		return nil, ErrUnsupportedDBKind
	}
}

func NewPostgres(db *sql.DB) (*Atlas, error) {
	drv, err := atlaspostgres.Open(db)
	if err != nil {
		return nil, err
	}
	return NewAtlas(DBPostgres, drv)
}

func NewMySQL(db *sql.DB) (*Atlas, error) {
	drv, err := atlasmysql.Open(db)
	if err != nil {
		return nil, err
	}
	return NewAtlas(DBMySQL, drv)
}

func (a *Atlas) Kind() DBKind {
	return a.kind
}

func (a *Atlas) Introspect(ctx context.Context, opts IntrospectOptions) (*Catalog, error) {
	mode := atlasschema.InspectSchemas | atlasschema.InspectTables
	if opts.IncludeViews {
		mode |= atlasschema.InspectViews
	}

	realm, err := a.inspector.InspectRealm(ctx, &atlasschema.InspectRealmOption{
		Mode:    mode,
		Schemas: cloneStrings(opts.IncludeNamespaces),
		Exclude: buildRealmExcludes(opts),
	})
	if err != nil {
		return nil, fmt.Errorf("atlas inspect realm: %w", err)
	}

	out := &Catalog{
		Kind:       a.kind,
		Namespaces: make([]Namespace, 0, len(realm.Schemas)),
	}

	for _, s := range realm.Schemas {
		if s == nil {
			continue
		}
		out.Namespaces = append(out.Namespaces, a.mapNamespace(s, opts))
	}

	return out, nil
}

func (a *Atlas) mapNamespace(s *atlasschema.Schema, opts IntrospectOptions) Namespace {
	ns := Namespace{
		Name:   s.Name,
		Tables: make([]Table, 0, len(s.Tables)),
	}

	for _, t := range s.Tables {
		if t == nil {
			continue
		}
		if !shouldIncludeTable(s.Name, t.Name, opts) {
			continue
		}
		ns.Tables = append(ns.Tables, a.mapTable(t, opts))
	}

	if opts.IncludeViews {
		ns.Views = make([]View, 0, len(s.Views))
		for _, v := range s.Views {
			if v == nil {
				continue
			}
			if !shouldIncludeTable(s.Name, v.Name, opts) {
				continue
			}
			ns.Views = append(ns.Views, a.mapView(v, opts))
		}
	}

	return ns
}

func (a *Atlas) mapTable(t *atlasschema.Table, opts IntrospectOptions) Table {
	out := Table{
		Name:    t.Name,
		Columns: make([]Column, 0, len(t.Columns)),
	}

	if opts.IncludeComments {
		out.Comment = extractComment(t.Attrs)
	}

	for i, c := range t.Columns {
		if c == nil {
			continue
		}
		out.Columns = append(out.Columns, a.mapColumn(c, i+1, opts))
	}

	if t.PrimaryKey != nil {
		out.PrimaryKey = indexPartColumnNames(t.PrimaryKey.Parts)
	}

	if opts.IncludeForeignKeys {
		out.ForeignKeys = make([]ForeignKey, 0, len(t.ForeignKeys))
		for _, fk := range t.ForeignKeys {
			if fk == nil {
				continue
			}
			out.ForeignKeys = append(out.ForeignKeys, mapForeignKey(fk))
		}
	}

	if opts.IncludeIndexes {
		out.Indexes = make([]Index, 0, len(t.Indexes))
		for _, idx := range t.Indexes {
			if idx == nil {
				continue
			}
			// Avoid duplicating the PK if the driver also exposes it in Indexes.
			if t.PrimaryKey != nil && sameIndexParts(t.PrimaryKey, idx) {
				continue
			}
			out.Indexes = append(out.Indexes, a.mapIndex(idx))
		}
	}

	return out
}

func (a *Atlas) mapIndex(idx *atlasschema.Index) Index {
	return Index{
		Name:     idx.Name,
		Columns:  indexPartColumnNames(idx.Parts),
		IsUnique: idx.Unique,
		Method:   indexMethod(a.kind, idx.Attrs),
	}
}

func (a *Atlas) mapView(v *atlasschema.View, opts IntrospectOptions) View {
	out := View{
		Name:       v.Name,
		Definition: v.Def,
		Columns:    make([]Column, 0, len(v.Columns)),
	}

	if opts.IncludeComments {
		out.Comment = extractComment(v.Attrs)
	}

	for i, c := range v.Columns {
		if c == nil {
			continue
		}
		out.Columns = append(out.Columns, a.mapColumn(c, i+1, opts))
	}

	return out
}

func (a *Atlas) mapColumn(c *atlasschema.Column, pos int, opts IntrospectOptions) Column {
	out := Column{
		Name:     c.Name,
		Nullable: c.Type == nil || c.Type.Null,
		Position: pos,
	}

	if c.Type != nil {
		out.DataType = normalizeTypeName(a.kind, c.Type)
	}

	if opts.IncludeComments {
		out.Comment = extractComment(c.Attrs)
	}

	return out
}

func shouldIncludeNamespace(ns string, opts IntrospectOptions) bool {
	if len(opts.IncludeNamespaces) > 0 && !slices.Contains(opts.IncludeNamespaces, ns) {
		return false
	}

	if slices.Contains(opts.ExcludeNamespaces, ns) {
		return false
	}
	return true
}

func shouldIncludeTable(namespace, table string, opts IntrospectOptions) bool {
	if !shouldIncludeNamespace(namespace, opts) {
		return false
	}

	// Default: include all tables in the namespace.
	included := true

	// If a namespace-specific include list exists, restrict to it.
	if includeTables, ok := opts.IncludeTablesByNamespace[namespace]; ok && len(includeTables) > 0 {
		included = slices.Contains(includeTables, table)
	}

	// Namespace-specific excludes override includes.
	if excludeTables, ok := opts.ExcludeTablesByNamespace[namespace]; ok && slices.Contains(excludeTables, table) {
		return false
	}

	return included
}

func mapForeignKey(fk *atlasschema.ForeignKey) ForeignKey {
	out := ForeignKey{
		Columns:    columnNames(fk.Columns),
		RefColumns: columnNames(fk.RefColumns),
	}
	if fk.RefTable != nil {
		out.RefTable = fk.RefTable.Name
		if fk.RefTable.Schema != nil {
			out.RefNamespace = fk.RefTable.Schema.Name
		}
	}
	return out
}

func extractComment(attrs []atlasschema.Attr) string {
	for _, a := range attrs {
		if c, ok := a.(*atlasschema.Comment); ok {
			return c.Text
		}
	}
	return ""
}

func normalizeTypeName(kind DBKind, ct *atlasschema.ColumnType) string {
	if ct == nil {
		return ""
	}
	if ct.Raw != "" {
		return strings.ToLower(strings.TrimSpace(ct.Raw))
	}
	// Fallback only if Raw is empty.
	return strings.ToLower(fmt.Sprintf("%T", ct.Type))
}

func columnNames(cols []*atlasschema.Column) []string {
	if len(cols) == 0 {
		return nil
	}
	out := make([]string, 0, len(cols))
	for _, c := range cols {
		if c != nil {
			out = append(out, c.Name)
		}
	}
	return out
}

func indexPartColumnNames(parts []*atlasschema.IndexPart) []string {
	if len(parts) == 0 {
		return nil
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != nil && p.C != nil {
			out = append(out, p.C.Name)
		}
	}
	return out
}

func indexMethod(kind DBKind, attrs []atlasschema.Attr) string {
	switch kind {
	case DBPostgres:
		for _, a := range attrs {
			if it, ok := a.(*atlaspostgres.IndexType); ok {
				return it.T
			}
		}
	case DBMySQL:
		for _, a := range attrs {
			if it, ok := a.(*atlasmysql.IndexType); ok {
				return it.T
			}
		}
	}
	return ""
}

func sameIndexParts(a, b *atlasschema.Index) bool {
	if a == nil || b == nil || len(a.Parts) != len(b.Parts) {
		return false
	}
	for i := range a.Parts {
		ap, bp := a.Parts[i], b.Parts[i]
		if ap == nil || bp == nil || ap.C == nil || bp.C == nil {
			return false
		}
		if ap.C.Name != bp.C.Name {
			return false
		}
	}
	return true
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func buildRealmExcludes(opts IntrospectOptions) []string {
	var out []string

	for _, ns := range opts.ExcludeNamespaces {
		out = append(out, ns)
	}

	for ns, tables := range opts.ExcludeTablesByNamespace {
		for _, tbl := range tables {
			out = append(out, ns+"."+tbl)
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}
