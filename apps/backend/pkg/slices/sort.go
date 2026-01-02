package slices

import (
	"sort"

	"golang.org/x/exp/constraints"
)

// Sort sorts the elements using the sort function.
func Sort[T any](elements []T, sortFunc func(a, b T) bool) {
	sort.Slice(elements, func(i, j int) bool {
		return sortFunc(elements[i], elements[j])
	})
}

func Sorted[T any](elements []T, sortFunc func(a, b T) bool) []T {
	copied := make([]T, len(elements))
	copy(copied, elements)

	Sort(copied, sortFunc)
	return copied
}

func SortAsc[T constraints.Ordered](elements []T) {
	sort.Slice(elements, func(i, j int) bool {
		return elements[i] < elements[j]
	})
}

func SortDesc[T constraints.Ordered](elements []T) {
	sort.Slice(elements, func(i, j int) bool {
		return elements[i] > elements[j]
	})
}

func SortedAsc[T constraints.Ordered](elements []T) []T {
	return Sorted(elements, func(a, b T) bool {
		return a < b
	})
}

func SortedDesc[T constraints.Ordered](elements []T) []T {
	return Sorted(elements, func(a, b T) bool {
		return a > b
	})
}
