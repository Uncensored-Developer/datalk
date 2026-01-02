package slices_test

import (
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/slices"
	"github.com/stretchr/testify/require"
)

func TestSort(t *testing.T) {
	t.Parallel()

	type A struct {
		Value int
		Name  string
	}

	elements := []A{
		{Value: 1, Name: "nico"},
		{Value: 2, Name: "seb"},
		{Value: 3, Name: "luis"},
	}

	// Asc sort
	slices.Sort(elements, func(a, b A) bool {
		return a.Value < b.Value
	})
	require.Equal(t, []A{
		{Value: 1, Name: "nico"},
		{Value: 2, Name: "seb"},
		{Value: 3, Name: "luis"},
	}, elements)

	// Desc sort
	slices.Sort(elements, func(a, b A) bool {
		return a.Value > b.Value
	})
	require.Equal(t, []A{
		{Value: 3, Name: "luis"},
		{Value: 2, Name: "seb"},
		{Value: 1, Name: "nico"},
	}, elements)

	// Asc sort by name
	slices.Sort(elements, func(a, b A) bool {
		return a.Name < b.Name
	})
	require.Equal(t, []A{
		{Value: 3, Name: "luis"},
		{Value: 1, Name: "nico"},
		{Value: 2, Name: "seb"},
	}, elements)
}

func TestSortedAsc(t *testing.T) {
	t.Parallel()

	require.EqualValues(t, []int{}, slices.SortedAsc([]int{}))

	v := []int{1, 3, 2, 4}
	require.EqualValues(t, []int{1, 2, 3, 4}, slices.SortedAsc(v))
	require.EqualValues(t, []int{1, 3, 2, 4}, v) // make sure it's not sorted in place
}

func TestSortedDesc(t *testing.T) {
	t.Parallel()

	require.EqualValues(t, []int{}, slices.SortedDesc([]int{}))

	v := []int{1, 3, 2, 4}
	require.EqualValues(t, []int{4, 3, 2, 1}, slices.SortedDesc(v))
	require.EqualValues(t, []int{1, 3, 2, 4}, v) // make sure it's not sorted in place
}

func TestSortAsc(t *testing.T) {
	t.Parallel()

	v := []int{1, 3, 2, 4}
	slices.SortAsc(v)
	require.EqualValues(t, []int{1, 2, 3, 4}, v)
}

func TestSortDesc(t *testing.T) {
	t.Parallel()

	v := []int{1, 3, 2, 4}
	slices.SortDesc(v)
	require.EqualValues(t, []int{4, 3, 2, 1}, v)
}
