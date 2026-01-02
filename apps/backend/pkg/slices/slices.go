package slices

import (
	"fmt"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/common"
)

func GetUniqueValues[T any, U comparable](values []T, f func(T) (U, bool)) []U {
	if f == nil {
		return nil
	}

	uniqueKeys := make([]U, 0, len(values))
	keysMap := map[U]struct{}{}

	for _, value := range values {
		key, ok := f(value)
		if !ok {
			continue
		}
		if _, ok := keysMap[key]; !ok {
			uniqueKeys = append(uniqueKeys, key)
			keysMap[key] = struct{}{}
		}
	}

	return uniqueKeys
}

func Map[T any, U any](in []T, f func(T) (U, error)) ([]U, error) {
	if f == nil {
		return nil, common.ErrNilFunction
	}

	result := make([]U, len(in))
	for index, input := range in {
		out, err := f(input)
		if err != nil {
			return nil, err
		}
		result[index] = out
	}
	return result, nil
}

func MapDereference[T any, U any](in []T, f func(T) (*U, error)) ([]U, error) {
	if f == nil {
		return nil, common.ErrNilFunction
	}

	result := make([]U, len(in))
	for index, input := range in {
		out, err := f(input)
		if err != nil {
			return nil, err
		}
		result[index] = *out
	}
	return result, nil
}

func Filter[T any](in []T, f func(T) bool) []T {
	if f == nil {
		return nil
	}

	result := make([]T, 0, len(in))
	for _, input := range in {
		if f(input) {
			result = append(result, input)
		}
	}
	return result
}

func Contains[T comparable](in []T, value T) bool {
	for _, item := range in {
		if item == value {
			return true
		}
	}
	return false
}

func MapNoError[T any, U any](in []T, f func(T) U) []U {
	if f == nil {
		return nil
	}

	result := make([]U, len(in))
	for index, input := range in {
		result[index] = f(input)
	}
	return result
}

// FirstN returns a slice copy with the first n elements of `in`, or `in` if its length is less than n, or empty slice if n is negative.
func FirstN[T any](in []T, n int) []T {
	if n <= 0 {
		return []T{}
	}
	resultSize := n
	if len(in) < n {
		resultSize = len(in)
	}

	result := make([]T, resultSize)
	copy(result, in[:resultSize])
	return result
}

// LastN returns a slice copy with the last n elements of `in`, or `in` if its length is less than n, or empty slice if n is negative.
func LastN[T any](in []T, n int) []T {
	if n <= 0 {
		return []T{}
	}
	resultSize := n
	if len(in) < n {
		resultSize = len(in)
	}

	result := make([]T, resultSize)
	startIndex := len(in) - resultSize
	copy(result, in[startIndex:])
	return result
}

func KeepLast[T any](_, replacement T) T {
	return replacement
}

func KeepFirst[T any](existing, _ T) T {
	return existing
}

// ToMapNoError creates a map with the input slice elements indexed by applying the given key and value functions.
// If duplicate keys are found, mergeFunc will be applied to decide which value to keep.
func ToMapNoError[T, V any, K comparable](elements []T, keyFunc func(a T) K, valueFunc func(a T) V, mergeFunc func(existing, replacement V) V) map[K]V {
	res := make(map[K]V, len(elements))

	for _, element := range elements {
		key := keyFunc(element)
		val := valueFunc(element)

		if prev, ok := res[key]; ok {
			val = mergeFunc(prev, val)
		}

		res[key] = val
	}

	return res
}

// GroupByNoError creates a map of the input slice elements grouped by applying the key function.
func GroupByNoError[T any, K comparable](elements []T, keyFunc func(a T) K) map[K][]T {
	res := make(map[K][]T)

	for _, element := range elements {
		key := keyFunc(element)
		res[key] = append(res[key], element)
	}

	return res
}

func Concat[T any](slices ...[]T) []T {
	var totalLen int
	for _, slice := range slices {
		totalLen += len(slice)
	}

	result := make([]T, totalLen)
	var offset int
	for _, slice := range slices {
		copy(result[offset:], slice)
		offset += len(slice)
	}

	return result
}

// Strings maps a slice of fmt.Stringer to a slice of strings.
func Strings[T fmt.Stringer](items []T) []string {
	return MapNoError(items, func(i T) string { return i.String() })
}

// Reversed returns a new slice with the elements in reverse order.
func Reversed[T any](in []T) []T {
	out := make([]T, len(in))
	for i := range in {
		out[i] = in[len(in)-1-i]
	}
	return out
}
