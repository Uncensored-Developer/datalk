package distlock

import (
	"context"
	"time"
)

type LockOptions struct {
	Wait bool
	TTL  *time.Duration
}

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
	// Lock blocks until acquiring all locks or context expires.
	Lock(ctx context.Context, resources []string, opts LockOptions) (LockedResources, error)

	// SupportsLeases returns true if the locker supports lease-based locks.
	SupportsLeases() bool

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
