package utils

import (
	"sort"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/introspector"
)

type namedObject interface {
	GetName() string
}

func sortByName[T namedObject](items []T) []T {
	if len(items) < 2 {
		return append([]T(nil), items...)
	}

	sorted := append([]T(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].GetName() < sorted[j].GetName()
	})

	return sorted
}

func SortedNamespaces(namespaces []introspector.Namespace) []introspector.Namespace {
	return sortByName(namespaces)
}

func SortedTables(tables []introspector.Table) []introspector.Table {
	return sortByName(tables)
}

func SortedViews(views []introspector.View) []introspector.View {
	return sortByName(views)
}

func SortedColumns(columns []introspector.Column) []introspector.Column {
	return sortByName(columns)
}
