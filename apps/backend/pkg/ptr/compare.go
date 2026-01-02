package ptr

import "time"

func StringPtrEquals(a, b *string) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil && b != nil, a != nil && b == nil:
		return false
	default:
		return *a == *b
	}
}

func TimePtrEquals(a, b *time.Time) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil && b != nil, a != nil && b == nil:
		return false
	default:
		return *a == *b
	}
}
