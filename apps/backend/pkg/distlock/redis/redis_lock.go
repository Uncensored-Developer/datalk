package redis

import (
	"context"
	"log/slog"
	"math"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/distlock"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/distlock/errors"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/logger"
	"github.com/go-redis/redis/v8"
)

// LockedResources holds information about acquired locks
// and provides methods to release them.
type LockedResources struct {
	ctx       context.Context
	client    *redis.Client
	logger    *slog.Logger
	resources []string
	createdAt time.Time
	expiresAt time.Time
}

// UncheckedUnlock releases all locks without returning errors.
func (l *LockedResources) UncheckedUnlock() {
	l.unlock(false)
}

// Unlock releases all locks and returns the first error encountered, if any.
func (l *LockedResources) Unlock() error {
	return l.unlock(true)
}

// Resources returns the list of locked resource names.
func (l *LockedResources) Resources() []string {
	return l.resources
}

// unlock executes a Lua script to safely delete keys only if they match the lockID.
func (l *LockedResources) unlock(checkErrors bool) error {
	for _, res := range l.resources {
		err := l.client.Del(context.Background(), res).Err()
		if err != nil && checkErrors {
			l.logger.Error("failed to unlock resource", slog.String("resource", res), logger.Err(err))
			return err
		}
	}
	return nil
}

type DistributedLocker struct {
	client *redis.Client
}

func NewDistributedLocker(opt *redis.Options) *DistributedLocker {
	client := redis.NewClient(opt)
	return &DistributedLocker{
		client: client,
	}
}

func (r *DistributedLocker) Type() string {
	return "redis"
}

func (r *DistributedLocker) Stop() error {
	return nil
}

func (r *DistributedLocker) Healthy(ctx context.Context) bool {
	return r.client.Ping(ctx).Err() == nil
}

// TryLock attempts to acquire locks immediately for the given resources.
// Returns an error if any resource is already locked.
func (r *DistributedLocker) TryLock(ctx context.Context, resources []string, expiration time.Duration) (distlock.LockedResources, error) {
	sorted := r.sortedKeys(resources)
	locked := make([]string, 0, len(sorted))

	now := time.Now()
	for _, res := range sorted {
		select {
		case <-ctx.Done():
			// Release any acquired locks on context cancellation.
			r.releasePartial(locked)
			return nil, ctx.Err()
		default:
		}

		// SET NX with EX to assign a TTL, ensuring stale locks eventually expire.
		ok, err := r.client.SetNX(ctx, res, res, expiration).Result()
		if err != nil {

			r.releasePartial(locked)
			return nil, err
		}
		if !ok {
			r.releasePartial(locked)
			return nil, errors.ErrFailedToAcquireLock
		}

		locked = append(locked, res)
	}

	return &LockedResources{
		ctx:       ctx,
		resources: resources,
		client:    r.client,
		createdAt: now,
		expiresAt: now.Add(expiration),
	}, nil
}

// WaitLock blocks until all locks on the given resources are acquired or timeout expires.
func (r *DistributedLocker) WaitLock(ctx context.Context, resources []string, expiration time.Duration) (distlock.LockedResources, error) {
	sorted := r.sortedKeys(resources)

	// Use backoff with jitter to avoid thundering herd.
	baseDelay := 50 * time.Millisecond
	maxDelay := 500 * time.Millisecond

	now := time.Now()
	for attempt := 0; ; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		// Attempt to acquire all locks in order.
		var locked []string
		success := true
		for _, res := range sorted {
			ok, err := r.client.SetNX(ctx, res, res, expiration).Result()
			if err != nil || !ok {
				success = false
				break
			}
			locked = append(locked, res)
		}
		if success {
			return &LockedResources{
				ctx:       ctx,
				resources: resources,
				client:    r.client,
				createdAt: now,
				expiresAt: now.Add(expiration),
			}, nil
		}
		// Release partial locks before retrying.
		r.releasePartial(locked)
		// Exponential backoff with jitter.
		delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
		if delay > maxDelay {
			delay = maxDelay
		}
		// Add up to ±25% jitter to spread retries.
		jitter := time.Duration((rand.Float64()*0.5 + 0.75) * float64(delay))
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(jitter):
		}
	}
}

// releasePartial deletes keys for resources in the slice
func (r *DistributedLocker) releasePartial(resources []string) {
	for _, res := range resources {
		r.client.Del(context.Background(), res)
	}
}

func (r *DistributedLocker) sortedKeys(resources []string) []string {
	// Sort resources lexicographically to enforce lock ordering and prevent deadlocks.
	sorted := make([]string, len(resources))
	copy(sorted, resources)
	sort.Strings(sorted)
	return sorted
}

// Ensure DistributedLocker implements distlock.DistributedLocker
var _ distlock.DistributedLocker = &DistributedLocker{}
