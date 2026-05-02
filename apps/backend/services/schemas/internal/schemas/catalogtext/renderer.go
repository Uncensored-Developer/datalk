package catalogtext

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/introspector"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/utils"
	"github.com/mdobak/go-xerrors"
)

var (
	ErrNilCatalog = xerrors.New("catalog cannot be nil")

	catalogTemplate = template.Must(template.New("catalog").Parse(`{{if .Objects}}{{range $index, $object := .Objects}}{{if gt $index 0}}

{{end}}{{$object}}{{end}}{{else}}database_kind: {{.Kind}}{{end}}`))

	objectTemplate = template.Must(template.New("object").Parse(`database_kind: {{.DatabaseKind}}
namespace: {{.Namespace}}
object_type: {{.ObjectType}}
object_name: {{.ObjectName}}
qualified_name: {{.QualifiedName}}{{if .Comment}}

comment: {{.Comment}}{{end}}{{if .PrimaryKey}}

primary_key: {{.PrimaryKey}}{{end}}{{if .ColumnsSection}}

{{.ColumnsSection}}{{end}}{{if .ForeignKeysSection}}

{{.ForeignKeysSection}}{{end}}{{if .IndexesSection}}

{{.IndexesSection}}{{end}}{{if .PurposeHintsSection}}

{{.PurposeHintsSection}}{{end}}`))
)

type catalogTemplateData struct {
	Kind    introspector.DBKind
	Objects []string
}

type objectTemplateData struct {
	DatabaseKind        introspector.DBKind
	Namespace           string
	ObjectType          string
	ObjectName          string
	QualifiedName       string
	Comment             string
	PrimaryKey          string
	ColumnsSection      string
	ForeignKeysSection  string
	IndexesSection      string
	PurposeHintsSection string
}

func RenderCatalog(catalog *introspector.Catalog) (string, error) {
	if catalog == nil {
		return "", ErrNilCatalog
	}

	objects := make([]string, 0, len(catalog.Namespaces))
	for _, namespace := range utils.SortedNamespaces(catalog.Namespaces) {
		for _, table := range utils.SortedTables(namespace.Tables) {
			objects = append(objects, RenderTable(catalog.Kind, namespace.Name, &table))
		}
		for _, view := range utils.SortedViews(namespace.Views) {
			objects = append(objects, RenderView(catalog.Kind, namespace.Name, &view))
		}
	}

	return executeTemplate(catalogTemplate, catalogTemplateData{
		Kind:    catalog.Kind,
		Objects: objects,
	})
}

func RenderNamespace(namespace introspector.Namespace, kind introspector.DBKind) string {
	objects := make([]string, 0, len(namespace.Tables)+len(namespace.Views))
	for _, table := range utils.SortedTables(namespace.Tables) {
		objects = append(objects, RenderTable(kind, namespace.Name, &table))
	}
	for _, view := range utils.SortedViews(namespace.Views) {
		objects = append(objects, RenderView(kind, namespace.Name, &view))
	}

	return strings.Join(objects, "\n\n")
}

func RenderTable(kind introspector.DBKind, namespace string, table *introspector.Table) string {
	rendered, err := executeTemplate(objectTemplate, buildObjectTemplateData(
		kind,
		namespace,
		"table",
		table.Name,
		table.Comment,
		table.PrimaryKey,
		table.Columns,
		table.ForeignKeys,
		table.Indexes,
		table.Comment,
	))
	if err != nil {
		panic(fmt.Sprintf("render table template: %v", err))
	}
	return rendered
}

func RenderView(kind introspector.DBKind, namespace string, view *introspector.View) string {
	rendered, err := executeTemplate(objectTemplate, buildObjectTemplateData(
		kind,
		namespace,
		"view",
		view.Name,
		view.Comment,
		nil,
		view.Columns,
		nil,
		nil,
		"",
	))
	if err != nil {
		panic(fmt.Sprintf("render view template: %v", err))
	}
	return rendered
}

func buildObjectTemplateData(
	kind introspector.DBKind,
	namespace string,
	objectType string,
	objectName string,
	comment string,
	primaryKey []string,
	columns []introspector.Column,
	foreignKeys []introspector.ForeignKey,
	indexes []introspector.Index,
	purposeHint string,
) objectTemplateData {
	data := objectTemplateData{
		DatabaseKind:  kind,
		Namespace:     namespace,
		ObjectType:    objectType,
		ObjectName:    objectName,
		QualifiedName: qualifiedName(namespace, objectName),
		Comment:       comment,
	}

	if len(primaryKey) > 0 {
		data.PrimaryKey = strings.Join(primaryKey, ", ")
	}

	sortedColumns := sortedColumns(columns)
	columnLines := make([]string, 0, len(sortedColumns)+1)
	if len(sortedColumns) > 0 {
		columnLines = append(columnLines, "columns:")
	}
	for i, column := range sortedColumns {
		columnLines = append(columnLines, fmt.Sprintf("%d. %s", i+1, renderColumn(column)))
	}
	if len(columnLines) > 0 {
		data.ColumnsSection = strings.Join(columnLines, "\n")
	}

	sortedForeignKeys := sortedForeignKeys(foreignKeys)
	foreignKeyLines := make([]string, 0, len(sortedForeignKeys)+1)
	if len(sortedForeignKeys) > 0 {
		foreignKeyLines = append(foreignKeyLines, "foreign_keys:")
	}
	for _, foreignKey := range sortedForeignKeys {
		foreignKeyLines = append(foreignKeyLines, "- "+renderForeignKey(foreignKey))
	}
	if len(foreignKeyLines) > 0 {
		data.ForeignKeysSection = strings.Join(foreignKeyLines, "\n")
	}

	sortedIndexes := sortedIndexes(indexes)
	indexLines := make([]string, 0, len(sortedIndexes)+1)
	if len(sortedIndexes) > 0 {
		indexLines = append(indexLines, "indexes:")
	}
	for _, index := range sortedIndexes {
		indexLines = append(indexLines, "- "+renderIndex(index))
	}
	if len(indexLines) > 0 {
		data.IndexesSection = strings.Join(indexLines, "\n")
	}

	if purposeHint != "" {
		data.PurposeHintsSection = strings.Join([]string{
			"Purpose Hints:",
			"- " + purposeHint,
		}, "\n")
	}

	return data
}

func executeTemplate(tmpl *template.Template, data any) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

func renderColumn(column introspector.Column) string {
	line := fmt.Sprintf("%s | type=%s | nullable=%t", column.Name, column.DataType, column.Nullable)
	if column.Comment != "" {
		line += fmt.Sprintf(" | comment=%q", column.Comment)
	}
	return line
}

func renderForeignKey(foreignKey introspector.ForeignKey) string {
	target := foreignKey.RefTable
	if foreignKey.RefNamespace != "" {
		target = foreignKey.RefNamespace + "." + target
	}

	return fmt.Sprintf(
		"columns=%s | references=%s(%s)",
		strings.Join(foreignKey.Columns, ","),
		target,
		strings.Join(foreignKey.RefColumns, ", "),
	)
}

func renderIndex(index introspector.Index) string {
	return fmt.Sprintf(
		"name=%s | unique=%t | method=%s | columns=%s",
		index.Name,
		index.IsUnique,
		index.Method,
		strings.Join(index.Columns, ","),
	)
}

func qualifiedName(namespace, objectName string) string {
	if namespace == "" {
		return objectName
	}
	return namespace + "." + objectName
}

func sortedColumns(columns []introspector.Column) []introspector.Column {
	out := append([]introspector.Column(nil), columns...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Position != out[j].Position {
			switch {
			case out[i].Position == 0:
				return false
			case out[j].Position == 0:
				return true
			default:
				return out[i].Position < out[j].Position
			}
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func sortedForeignKeys(foreignKeys []introspector.ForeignKey) []introspector.ForeignKey {
	out := append([]introspector.ForeignKey(nil), foreignKeys...)
	sort.Slice(out, func(i, j int) bool {
		return foreignKeySortKey(out[i]) < foreignKeySortKey(out[j])
	})
	return out
}

func foreignKeySortKey(foreignKey introspector.ForeignKey) string {
	return strings.Join([]string{
		strings.Join(foreignKey.Columns, ","),
		foreignKey.RefNamespace,
		foreignKey.RefTable,
		strings.Join(foreignKey.RefColumns, ","),
	}, "|")
}

func sortedIndexes(indexes []introspector.Index) []introspector.Index {
	out := append([]introspector.Index(nil), indexes...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return strings.Join(out[i].Columns, ",") < strings.Join(out[j].Columns, ",")
	})
	return out
}
