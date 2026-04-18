package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"sort"
	"sync"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/distlock"
	distlockerrors "github.com/Uncensored-Developer/datalk/apps/backend/pkg/distlock/errors"
	mapset "github.com/deckarep/golang-set/v2"
)

const lockerType = "postgres"

type DistributedLocker struct {
	db *sql.DB
}

func NewDistributedLocker(db *sql.DB) *DistributedLocker {
	return &DistributedLocker{db: db}
}

func (l *DistributedLocker) Lock(ctx context.Context, resources []string, opts distlock.LockOptions) (distlock.LockedResources, error) {
	// Session-based advisory locks do not support TTL-style leases.
	normalized := normalizeResources(resources)
	if len(normalized) == 0 {
		return &LockedResources{
			resources: normalized,
			released:  true,
		}, nil
	}

	conn, err := l.db.Conn(ctx)
	if err != nil {
		return nil, err
	}

	keys := make([]resourceLock, 0, len(normalized))
	for _, resource := range normalized {
		lock := resourceLock{
			resource: resource,
			key:      advisoryKey(resource),
		}

		if opts.Wait {
			err = advisoryLock(ctx, conn, lock.key)
		} else {
			var acquired bool
			acquired, err = tryAdvisoryLock(ctx, conn, lock.key)
			if err == nil && !acquired {
				err = distlockerrors.ErrFailedToAcquireLock
			}
		}
		if err != nil {
			_ = conn.Close()
			return nil, err
		}

		keys = append(keys, lock)
	}

	return &LockedResources{
		conn:      conn,
		resources: normalized,
		locks:     keys,
	}, nil
}

func (l *DistributedLocker) SupportsLeases() bool { return false }

func (l *DistributedLocker) Type() string { return lockerType }

func (l *DistributedLocker) Stop() error { return nil }

func (l *DistributedLocker) Healthy(ctx context.Context) bool {
	return l.db.PingContext(ctx) == nil
}

type resourceLock struct {
	resource string
	key      int64
}

type LockedResources struct {
	mu        sync.Mutex
	conn      *sql.Conn
	resources []string
	locks     []resourceLock
	released  bool
}

func (l *LockedResources) UncheckedUnlock() {
	_ = l.unlock(false)
}

func (l *LockedResources) Unlock() error {
	return l.unlock(true)
}

func (l *LockedResources) Resources() []string {
	return append([]string(nil), l.resources...)
}

func (l *LockedResources) unlock(verifyOwnership bool) error {
	l.mu.Lock()
	if l.released {
		l.mu.Unlock()
		return nil
	}

	conn := l.conn
	locks := append([]resourceLock(nil), l.locks...)
	l.conn = nil
	l.released = true
	l.mu.Unlock()

	if conn == nil {
		return nil
	}

	var firstErr error
	if verifyOwnership {
		for i := len(locks) - 1; i >= 0; i-- {
			unlocked, err := advisoryUnlock(context.Background(), conn, locks[i].key)
			if err != nil && firstErr == nil {
				firstErr = err
			}
			if !unlocked && err == nil && firstErr == nil {
				firstErr = fmt.Errorf("resource %q is not locked by this session", locks[i].resource)
			}
		}
	}

	if err := conn.Close(); err != nil && firstErr == nil {
		firstErr = err
	}

	return firstErr
}

func normalizeResources(resources []string) []string {
	// Sort resources lexicographically to enforce lock ordering and prevent deadlocks.
	sorted := mapset.NewSet[string](resources...).ToSlice()
	copy(sorted, resources)
	sort.Strings(sorted)
	return sorted
}

func advisoryKey(resource string) int64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(resource))
	return int64(hasher.Sum64())
}

func advisoryLock(ctx context.Context, conn *sql.Conn, key int64) error {
	_, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", key)
	return err
}

func tryAdvisoryLock(ctx context.Context, conn *sql.Conn, key int64) (bool, error) {
	var locked bool
	err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&locked)
	return locked, err
}

func advisoryUnlock(ctx context.Context, conn *sql.Conn, key int64) (bool, error) {
	var unlocked bool
	err := conn.QueryRowContext(ctx, "SELECT pg_advisory_unlock($1)", key).Scan(&unlocked)
	return unlocked, err
}

var (
	_ distlock.DistributedLocker = &DistributedLocker{}
	_ distlock.LockedResources   = &LockedResources{}
)
