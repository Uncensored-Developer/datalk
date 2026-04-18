package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/distlock"
	distlockerrors "github.com/Uncensored-Developer/datalk/apps/backend/pkg/distlock/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistributedLocker_Lock(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	t.Run("No wait lock and unlock", func(t *testing.T) {
		t.Parallel()

		resources := testResources(t, "b", "a", "a")

		locked, err := locker.Lock(ctx, resources, distlock.LockOptions{Wait: false})
		require.NoError(t, err)
		assert.Equal(t, []string{resources[1], resources[0]}, locked.Resources())

		_, err = locker.Lock(ctx, []string{resources[0]}, distlock.LockOptions{Wait: false})
		assert.ErrorIs(t, err, distlockerrors.ErrFailedToAcquireLock)

		assert.NoError(t, locked.Unlock())

		locked2, err := locker.Lock(ctx, []string{resources[0]}, distlock.LockOptions{Wait: false})
		require.NoError(t, err)
		assert.NoError(t, locked2.Unlock())
	})

	t.Run("Wait lock success", func(t *testing.T) {
		t.Parallel()

		resource := testResource(t, "wait-release")

		otherLocker := NewDistributedLocker(runner.Conn)
		locked, err := otherLocker.Lock(ctx, []string{resource}, distlock.LockOptions{Wait: false})
		require.NoError(t, err)

		go func() {
			time.Sleep(100 * time.Millisecond)
			_ = locked.Unlock()
		}()

		ctxTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		locked2, err := locker.Lock(ctxTimeout, []string{resource}, distlock.LockOptions{Wait: true})
		require.NoError(t, err)
		assert.NoError(t, locked2.Unlock())
	})

	t.Run("Partial lock failure releases earlier locks", func(t *testing.T) {
		t.Parallel()

		firstResource := testResource(t, "partial-a")
		secondResource := testResource(t, "partial-b")

		otherLocker := NewDistributedLocker(runner.Conn)
		blockingLock, err := otherLocker.Lock(ctx, []string{secondResource}, distlock.LockOptions{Wait: false})
		require.NoError(t, err)
		defer blockingLock.Unlock()

		_, err = locker.Lock(ctx, []string{firstResource, secondResource}, distlock.LockOptions{Wait: false})
		require.ErrorIs(t, err, distlockerrors.ErrFailedToAcquireLock)

		retryLock, err := otherLocker.Lock(ctx, []string{firstResource}, distlock.LockOptions{Wait: false})
		require.NoError(t, err)
		assert.NoError(t, retryLock.Unlock())
	})

	t.Run("Unlock idempotency", func(t *testing.T) {
		t.Parallel()

		resource := testResource(t, "idempotent")

		locked, err := locker.Lock(ctx, []string{resource}, distlock.LockOptions{Wait: false})
		require.NoError(t, err)
		assert.NoError(t, locked.Unlock())
		assert.NoError(t, locked.Unlock())
	})

	t.Run("Unlock ownership check", func(t *testing.T) {
		t.Parallel()

		resource := testResource(t, "ownership")

		locked, err := locker.Lock(ctx, []string{resource}, distlock.LockOptions{Wait: false})
		require.NoError(t, err)

		sqlLocked, ok := locked.(*LockedResources)
		require.True(t, ok)

		otherConn, err := runner.Conn.Conn(ctx)
		require.NoError(t, err)

		originalConn := sqlLocked.conn
		sqlLocked.conn = otherConn

		err = sqlLocked.Unlock()
		require.Error(t, err)
		assert.ErrorContains(t, err, `is not locked by this session`)

		released, releaseErr := advisoryUnlock(context.Background(), originalConn, advisoryKey(resource))
		require.NoError(t, releaseErr)
		assert.True(t, released)
		assert.NoError(t, originalConn.Close())
	})
}

func TestDistributedLocker_TypeHealthyStop(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	typeStr := locker.Type()
	assert.NotEmpty(t, typeStr)
	assert.Equal(t, "postgres", typeStr)

	assert.False(t, locker.SupportsLeases())
	assert.True(t, locker.Healthy(ctx))
	assert.NoError(t, locker.Stop())
}

func testResource(t *testing.T, suffix string) string {
	t.Helper()

	return fmt.Sprintf("distlock-postgres/%s/%s", t.Name(), suffix)
}

func testResources(t *testing.T, suffixes ...string) []string {
	t.Helper()

	resources := make([]string, 0, len(suffixes))
	for _, suffix := range suffixes {
		resources = append(resources, testResource(t, suffix))
	}

	return resources
}
