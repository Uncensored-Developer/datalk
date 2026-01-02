package slices_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/slices"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFirstN(t *testing.T) {
	t.Parallel()

	bar := []string{"b", "a", "r"}
	empty := []string{}

	type args struct {
		in []string
		n  int
	}
	tests := []struct {
		args args
		want []string
	}{
		{args{empty, -2}, empty},
		{args{empty, 0}, empty},
		{args{empty, 5}, empty},
		{args{bar, -1}, empty},
		{args{bar, 0}, empty},
		{args{bar, 1}, []string{"b"}},
		{args{bar, 2}, []string{"b", "a"}},
		{args{bar, 3}, bar},
		{args{bar, 4}, bar},
	}
	for index, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("case_%d", index), func(t *testing.T) {
			t.Parallel()

			got := slices.FirstN(tt.args.in, tt.args.n)
			assert.Equal(t, tt.want, got)
			if len(tt.args.in) > 0 && len(got) > 0 {
				// assert that output is a copy
				assert.True(t, &tt.args.in[0] != &got[0])
			}
		})
	}
}

func TestLastN(t *testing.T) {
	t.Parallel()

	bar := []string{"b", "a", "r"}
	empty := []string{}

	type args struct {
		in []string
		n  int
	}
	tests := []struct {
		args args
		want []string
	}{
		{args{empty, -2}, empty},
		{args{empty, 0}, empty},
		{args{empty, 5}, empty},
		{args{bar, -1}, empty},
		{args{bar, 0}, empty},
		{args{bar, 1}, []string{"r"}},
		{args{bar, 2}, []string{"a", "r"}},
		{args{bar, 3}, bar},
		{args{bar, 4}, bar},
	}
	for index, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("case_%d", index), func(t *testing.T) {
			t.Parallel()

			got := slices.LastN(tt.args.in, tt.args.n)
			assert.Equal(t, tt.want, got)
			if len(tt.args.in) > 0 && len(got) > 0 {
				// assert that output is a copy
				assert.True(t, &tt.args.in[0] != &got[0])
			}
		})
	}
}

func TestToMapNoError(t *testing.T) {
	t.Parallel()

	type keyValue struct {
		key   string
		value string
	}
	getKey := func(kv keyValue) string { return kv.key }
	getValue := func(kv keyValue) string { return kv.value }
	failOnDuplicates := func(_, _ string) string {
		t.Helper()

		t.Error("duplicate key detected")
		return ""
	}

	tests := []struct {
		desc      string
		inputs    []keyValue
		mergeFunc func(string, string) string
		want      map[string]string
	}{
		{
			desc:      "Nil inputs",
			inputs:    nil,
			mergeFunc: failOnDuplicates,
			want:      map[string]string{},
		},
		{
			desc:      "Empty inputs",
			inputs:    []keyValue{},
			mergeFunc: failOnDuplicates,
			want:      map[string]string{},
		},
		{
			desc:      "One input",
			inputs:    []keyValue{{"key", "value"}},
			mergeFunc: failOnDuplicates,
			want:      map[string]string{"key": "value"},
		},
		{
			desc:      "No duplicates",
			inputs:    []keyValue{{"k1", "v1"}, {"k3", "v3"}, {"k2", "v2"}},
			mergeFunc: failOnDuplicates,
			want:      map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"},
		},
		{
			desc:      "Duplicates keep first",
			inputs:    []keyValue{{"k1", "v1"}, {"k2", "v2"}, {"k1", "v3"}, {"k1", "v4"}},
			mergeFunc: slices.KeepFirst[string],
			want:      map[string]string{"k1": "v1", "k2": "v2"},
		},
		{
			desc:      "Duplicates keep last",
			inputs:    []keyValue{{"k1", "v1"}, {"k2", "v2"}, {"k1", "v3"}, {"k1", "v4"}},
			mergeFunc: slices.KeepLast[string],
			want:      map[string]string{"k1": "v4", "k2": "v2"},
		},
	}

	for _, test := range tests {
		tt := test
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			got := slices.ToMapNoError(tt.inputs, getKey, getValue, tt.mergeFunc)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGroupByNoError(t *testing.T) {
	t.Parallel()

	type keyValue struct {
		key   string
		value string
	}
	getKey := func(kv keyValue) string { return kv.key }

	tests := []struct {
		desc   string
		inputs []keyValue
		want   map[string][]keyValue
	}{
		{
			desc:   "Nil inputs",
			inputs: nil,
			want:   map[string][]keyValue{},
		},
		{
			desc:   "Empty inputs",
			inputs: []keyValue{},
			want:   map[string][]keyValue{},
		},
		{
			desc:   "One input",
			inputs: []keyValue{{"key", "value"}},
			want:   map[string][]keyValue{"key": {{"key", "value"}}},
		},
		{
			desc:   "No duplicates",
			inputs: []keyValue{{"k1", "v1"}, {"k3", "v3"}, {"k2", "v2"}},
			want:   map[string][]keyValue{"k1": {{"k1", "v1"}}, "k2": {{"k2", "v2"}}, "k3": {{"k3", "v3"}}},
		},
		{
			desc:   "Duplicates",
			inputs: []keyValue{{"k1", "v1"}, {"k2", "v2"}, {"k1", "v3"}, {"k1", "v4"}},
			want:   map[string][]keyValue{"k1": {{"k1", "v1"}, {"k1", "v3"}, {"k1", "v4"}}, "k2": {{"k2", "v2"}}},
		},
	}

	for _, test := range tests {
		tt := test
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			got := slices.GroupByNoError(tt.inputs, getKey)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConcat(t *testing.T) {
	t.Parallel()

	require.EqualValues(t, []int{}, slices.Concat([]int{}, []int{}))
	require.EqualValues(t, []int{1, 2, 3, 4}, slices.Concat([]int{1, 2}, []int{3, 4}))

	v := []int{1, 2}
	require.EqualValues(t, []int{1, 2, 3, 4}, slices.Concat(v, []int{3, 4}))
	require.EqualValues(t, []int{1, 2}, v) // make sure it's not modified

	require.EqualValues(t, []int{1, 2, 3, 4, 5}, slices.Concat([]int{1}, []int{2}, []int{3}, []int{4}, []int{5}))
}

type intWrapper struct {
	i int64
}

func (f intWrapper) String() string {
	return strconv.FormatInt(f.i, 10)
}

// Test Strings
func TestStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		arg  []intWrapper
		want []string
	}{
		{
			name: "nil returns empty slice",
			arg:  nil,
			want: []string{},
		},
		{
			name: "empty slice returns empty slice",
			arg:  []intWrapper{},
			want: []string{},
		},
		{
			name: "non-empty slice returns values",
			arg:  []intWrapper{{i: 0}, {i: 1}, {i: 2}},
			want: []string{"0", "1", "2"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := slices.Strings(tt.arg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReversed(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []int{}, slices.Reversed([]int{}))
	assert.Equal(t, []int{1}, slices.Reversed([]int{1}))
	assert.Equal(t, []int{2, 1}, slices.Reversed([]int{1, 2}))
	assert.Equal(t, []int{3, 2, 1}, slices.Reversed([]int{1, 2, 3}))

	assert.Equal(t, []string{"a", "b", "c"}, slices.Reversed([]string{"c", "b", "a"}))
}
