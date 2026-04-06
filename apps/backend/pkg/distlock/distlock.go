package distlock

import (
	"context"
	"time"
)

//go:generate go tool with-modfile mockery --name LockedResources --outpkg testing --output ./testing --filename generated__distlock_locked_resources_mocks.go
type LockedResources interface {
	// UncheckedUnlock releases locks without verifying ownership. Use only in emergencies.
	UncheckedUnlock()

	// Unlock safely releases locks, returning an error if the ownership check fails.
	Unlock() error

	// Resources returns the sorted list of locked resource keys.
	Resources() []string
}

//go:generate go tool with-modfile mockery --name DistributedLocker --outpkg testing --output ./testing --filename generated__distlock_mocks.go
type DistributedLocker interface {
	// WaitLock blocks until acquiring all locks or context expires. expiration is the TTL for each lock.
	WaitLock(ctx context.Context, resources []string, expiration time.Duration) (LockedResources, error)

	// TryLock attempts to acquire locks without blocking. expiration is the TTL for each lock.
	TryLock(ctx context.Context, resources []string, expiration time.Duration) (LockedResources, error)

	// Type returns a unique identifier for the locker implementation.
	Type() string

	// Stop gracefully closes underlying resources (e.g., Redis connections).
	Stop() error

	// Healthy returns true if the locker is healthy, false otherwise
	Healthy(ctx context.Context) bool
}

var globalLocker DistributedLocker

func SetGlobalLocker(l DistributedLocker) {
	globalLocker = l
}

func GlobalLocker() DistributedLocker {
	return globalLocker
}
