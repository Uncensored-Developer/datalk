package ptr

import "time"

func Ptr[T any](v T) *T {
	return &v
}

func CopyPtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	return Ptr(*p)
}

func ErrorPtr(err error) *error {
	return Ptr(err)
}

func StringPtr(s string) *string {
	return Ptr(s)
}

func CopyStringPtr(s *string) *string {
	return CopyPtr(s)
}

func TimePtr(t time.Time) *time.Time {
	return Ptr(t)
}

func CopyTimePtr(t *time.Time) *time.Time {
	return CopyPtr(t)
}

func DurationPtr(t time.Duration) *time.Duration {
	return Ptr(t)
}

func CopyDurationPtr(t *time.Duration) *time.Duration {
	return CopyPtr(t)
}

func UTCDatePtr(year, month, day int) *time.Time {
	d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return TimePtr(d)
}

func Float64Ptr(f float64) *float64 {
	return Ptr(f)
}

func IntPtr(i int) *int {
	return Ptr(i)
}

func Int64Ptr(i int64) *int64 {
	return Ptr(i)
}

func BoolPtr(b bool) *bool {
	return Ptr(b)
}
