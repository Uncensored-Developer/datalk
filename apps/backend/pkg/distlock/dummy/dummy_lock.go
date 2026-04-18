package dummy

import (
	"context"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/distlock"
)

type DummyLockedResources struct {
	resources []string
	expiry    time.Time
}

func (l *DummyLockedResources) Unlock() error {
	return nil
}

func (l *DummyLockedResources) UncheckedUnlock() {
	l.Unlock() // nolint #nosec
}

func (l *DummyLockedResources) Resources() []string {
	return l.resources
}

func (l *DummyLockedResources) ApproximateLockExpiry() time.Time {
	return l.expiry
}

type DummyDistributedLocker struct {
	defaultTimeout time.Duration
}

func NewDummyDistributedLocker() *DummyDistributedLocker {
	return &DummyDistributedLocker{
		defaultTimeout: time.Hour,
	}
}

func (dl *DummyDistributedLocker) SetDefaultTimeout(timeout time.Duration) {
	dl.defaultTimeout = timeout
}

func (dl *DummyDistributedLocker) DefaultTimeout() time.Duration { return dl.defaultTimeout }

func (dl *DummyDistributedLocker) Stop() error { return nil }

func (dl *DummyDistributedLocker) Lock(ctx context.Context, resources []string, opts distlock.LockOptions) (distlock.LockedResources, error) {
	return &DummyLockedResources{
		resources: resources,
		expiry:    time.Now().Add(dl.DefaultTimeout()),
	}, nil
}

func (dl *DummyDistributedLocker) SupportsLeases() bool { return false }

func (dl *DummyDistributedLocker) Type() string { return "dummy" }

func (dl *DummyDistributedLocker) Healthy(ctx context.Context) bool {
	return true
}

var _ distlock.DistributedLocker = &DummyDistributedLocker{}
