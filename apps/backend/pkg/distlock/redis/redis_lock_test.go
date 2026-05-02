package redis

import (
	"context"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/distlock"
	"github.com/gotidy/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistributedLocker_Lock(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	t.Run("No wait lock and unlock", func(t *testing.T) {
		t.Parallel()
		resources := []string{"resource1", "resource2"}

		locked, err := distLocker.Lock(ctx, resources, distlock.LockOptions{
			TTL:  ptr.Of(2 * time.Second),
			Wait: false,
		})
		require.NoError(t, err)
		assert.ElementsMatch(t, resources, locked.Resources())

		// Locking again should fail
		_, err = distLocker.Lock(ctx, resources, distlock.LockOptions{
			TTL:  ptr.Of(2 * time.Second),
			Wait: false,
		})
		assert.Error(t, err)

		// Unlock
		assert.NoError(t, locked.Unlock())

		// Now TryLock should succeed again
		locked2, err := distLocker.Lock(ctx, resources, distlock.LockOptions{
			TTL:  ptr.Of(2 * time.Second),
			Wait: false,
		})
		assert.NoError(t, err)
		assert.NoError(t, locked2.Unlock())
	})

	//t.Run("Wait lock with timeout", func(t *testing.T) {
	//	t.Parallel()
	//	resources := []string{"resource3"}
	//
	//	// Lock resource in another locker
	//	locker2, cleanup2, err := setupTestRedis(cfg)
	//	require.NoError(t, err)
	//	defer cleanup2()
	//
	//	locked, err := locker2.Lock(ctx, resources, distlock.LockOptions{
	//		TTL:  ptr.Of(5 * time.Second),
	//		Wait: false,
	//	})
	//	require.NoError(t, err)
	//	defer locked.Unlock()
	//
	//	ctxTimeout, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	//	defer cancel()
	//	_, err = distLocker.Lock(ctxTimeout, resources, distlock.LockOptions{
	//		TTL:  ptr.Of(2 * time.Second),
	//		Wait: true,
	//	})
	//	assert.Error(t, err)
	//})

	t.Run("Wait lock success", func(t *testing.T) {
		t.Parallel()
		resources := []string{"resource4"}

		// Lock and unlock quickly in another goroutine
		locker2, cleanup2, err := setupTestRedis(cfg)
		require.NoError(t, err)
		defer cleanup2()
		locked, err := locker2.Lock(ctx, resources, distlock.LockOptions{
			TTL:  ptr.Of(1 * time.Second),
			Wait: false,
		})
		require.NoError(t, err)
		go func() {
			time.Sleep(300 * time.Millisecond)
			locked.Unlock()
		}()

		ctxTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		locked2, err := distLocker.Lock(ctxTimeout, resources, distlock.LockOptions{
			TTL:  ptr.Of(2 * time.Second),
			Wait: true,
		})
		assert.NoError(t, err)
		assert.NoError(t, locked2.Unlock())
	})

	t.Run("Unlock idempotency", func(t *testing.T) {
		t.Parallel()

		resources := []string{"resource5"}

		locked, err := distLocker.Lock(ctx, resources, distlock.LockOptions{
			TTL:  ptr.Of(2 * time.Second),
			Wait: false,
		})
		require.NoError(t, err)
		assert.NoError(t, locked.Unlock())
		// Unlock again should not fail
		assert.NoError(t, locked.Unlock())
	})
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
