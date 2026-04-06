package redis

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistributedLocker_TryLockAndUnlock(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	resources := []string{"resource1", "resource2"}

	locked, err := distLocker.TryLock(ctx, resources, 2*time.Second)
	require.NoError(t, err)
	assert.ElementsMatch(t, resources, locked.Resources())

	// Locking again should fail
	_, err = distLocker.TryLock(ctx, resources, 2*time.Second)
	assert.Error(t, err)

	// Unlock
	assert.NoError(t, locked.Unlock())

	// Now TryLock should succeed again
	locked2, err := distLocker.TryLock(ctx, resources, 2*time.Second)
	assert.NoError(t, err)
	assert.NoError(t, locked2.Unlock())
}

func TestDistributedLocker_WaitLockTimeout(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	resources := []string{"resource3"}

	// Lock resource in another locker
	locker2, cleanup2, err := setupTestRedis(cfg)
	require.NoError(t, err)
	defer cleanup2()

	locked, err := locker2.TryLock(ctx, resources, 5*time.Second)
	require.NoError(t, err)
	defer locked.Unlock()

	ctxTimeout, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	_, err = distLocker.WaitLock(ctxTimeout, resources, 2*time.Second)
	assert.Error(t, err)
}

func TestDistributedLocker_WaitLockSuccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	resources := []string{"resource4"}

	// Lock and unlock quickly in another goroutine
	locker2, cleanup2, err := setupTestRedis(cfg)
	require.NoError(t, err)
	defer cleanup2()
	locked, err := locker2.TryLock(ctx, resources, 1*time.Second)
	require.NoError(t, err)
	go func() {
		time.Sleep(300 * time.Millisecond)
		locked.Unlock()
	}()

	ctxTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	locked2, err := distLocker.WaitLock(ctxTimeout, resources, 2*time.Second)
	assert.NoError(t, err)
	assert.NoError(t, locked2.Unlock())
}

func TestDistributedLocker_UnlockIdempotency(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	resources := []string{"resource5"}

	locked, err := distLocker.TryLock(ctx, resources, 2*time.Second)
	require.NoError(t, err)
	assert.NoError(t, locked.Unlock())
	// Unlock again should not fail
	assert.NoError(t, locked.Unlock())
}

func TestDistributedLocker_TypeHealthyStop(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	typeStr := distLocker.Type()
	assert.NotEmpty(t, typeStr)
	assert.Equal(t, "redis", typeStr)

	assert.True(t, distLocker.Healthy(ctx))
	assert.NoError(t, distLocker.Stop())
}
