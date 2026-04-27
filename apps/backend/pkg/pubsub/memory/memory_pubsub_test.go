package memory

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBus_PublishSubscribe(t *testing.T) {
	t.Parallel()

	bus := NewMemoryBus()
	topic := "memory.delivery"
	receivedCh := make(chan pubsub.Message, 1)

	sub, err := bus.Subscribe(t.Context(), topic, pubsub.HandlerFunc(func(ctx context.Context, msg pubsub.Message) error {
		receivedCh <- msg
		return nil
	}))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sub.Close(context.Background()))
	})

	msg, err := pubsub.NewMessage(topic, "", map[string]string{"status": "ok"})
	require.NoError(t, err)

	require.NoError(t, bus.Publish(t.Context(), topic, *msg))

	select {
	case got := <-receivedCh:
		assert.Equal(t, msg.ID, got.ID)
		assert.Equal(t, topic, got.Topic)
		assert.Equal(t, msg.Body, got.Body)
		assert.False(t, got.PublishedAt.IsZero())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestBus_MultipleSubscribersReceiveMessage(t *testing.T) {
	t.Parallel()

	bus := NewMemoryBus()
	topic := "memory.fanout"
	firstCh := make(chan string, 1)
	secondCh := make(chan string, 1)

	sub1, err := bus.Subscribe(t.Context(), topic, pubsub.HandlerFunc(func(ctx context.Context, msg pubsub.Message) error {
		firstCh <- msg.ID
		return nil
	}))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sub1.Close(context.Background()))
	})

	sub2, err := bus.Subscribe(t.Context(), topic, pubsub.HandlerFunc(func(ctx context.Context, msg pubsub.Message) error {
		secondCh <- msg.ID
		return nil
	}))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sub2.Close(context.Background()))
	})

	msg, err := pubsub.NewMessage(topic, "", map[string]string{"kind": "fanout"})
	require.NoError(t, err)
	require.NoError(t, bus.Publish(t.Context(), topic, *msg))

	assert.Equal(t, msg.ID, waitForString(t, firstCh))
	assert.Equal(t, msg.ID, waitForString(t, secondCh))
}

func TestBus_SubscriptionCloseStopsDelivery(t *testing.T) {
	t.Parallel()

	bus := NewMemoryBus()
	topic := "memory.close.subscription"
	receivedCh := make(chan string, 1)

	sub, err := bus.Subscribe(t.Context(), topic, pubsub.HandlerFunc(func(ctx context.Context, msg pubsub.Message) error {
		receivedCh <- msg.ID
		return nil
	}))
	require.NoError(t, err)

	require.NoError(t, sub.Close(t.Context()))

	msg, err := pubsub.NewMessage(topic, "", map[string]string{"kind": "closed"})
	require.NoError(t, err)
	require.NoError(t, bus.Publish(t.Context(), topic, *msg))

	select {
	case got := <-receivedCh:
		t.Fatalf("unexpected message after subscription close: %s", got)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestBus_Close(t *testing.T) {
	t.Parallel()

	bus := NewMemoryBus()
	topic := "memory.close.bus"

	sub, err := bus.Subscribe(t.Context(), topic, pubsub.HandlerFunc(func(ctx context.Context, msg pubsub.Message) error {
		return nil
	}))
	require.NoError(t, err)

	assert.NoError(t, bus.Close())
	assert.NoError(t, sub.Close(context.Background()))

	msg, err := pubsub.NewMessage(topic, "", map[string]string{"kind": "closed"})
	require.NoError(t, err)

	assert.ErrorIs(t, bus.Publish(t.Context(), topic, *msg), ErrClosed)

	_, err = bus.Subscribe(t.Context(), topic, pubsub.HandlerFunc(func(ctx context.Context, msg pubsub.Message) error {
		return nil
	}))
	assert.ErrorIs(t, err, ErrClosed)
}

func TestBus_RetriesHandler(t *testing.T) {
	t.Parallel()

	bus := NewMemoryBus()
	topic := "memory.retry"
	doneCh := make(chan struct{}, 1)
	var attempts atomic.Int32

	sub, err := bus.Subscribe(
		t.Context(),
		topic,
		pubsub.HandlerFunc(func(ctx context.Context, msg pubsub.Message) error {
			if attempts.Add(1) < 3 {
				return errors.New("transient")
			}
			doneCh <- struct{}{}
			return nil
		}),
		pubsub.WithMaxRetries(2),
		pubsub.WithRetryBackoff(10*time.Millisecond),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sub.Close(context.Background()))
	})

	msg, err := pubsub.NewMessage(topic, "", map[string]string{"kind": "retry"})
	require.NoError(t, err)
	require.NoError(t, bus.Publish(t.Context(), topic, *msg))

	select {
	case <-doneCh:
		assert.Equal(t, int32(3), attempts.Load())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for retried handler")
	}
}

func TestBus_WithOrderedForcesSequentialProcessing(t *testing.T) {
	t.Parallel()

	bus := NewMemoryBus()
	topic := "memory.ordered"
	startedCh := make(chan string, 2)
	releaseCh := make(chan struct{})
	finishedCh := make(chan string, 2)

	sub, err := bus.Subscribe(
		t.Context(),
		topic,
		pubsub.HandlerFunc(func(ctx context.Context, msg pubsub.Message) error {
			startedCh <- msg.ID
			<-releaseCh
			finishedCh <- msg.ID
			return nil
		}),
		pubsub.WithConcurrency(4),
		pubsub.WithOrdered(),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sub.Close(context.Background()))
	})

	firstMsg, err := pubsub.NewMessage(topic, "", map[string]int{"n": 1})
	require.NoError(t, err)
	secondMsg, err := pubsub.NewMessage(topic, "", map[string]int{"n": 2})
	require.NoError(t, err)

	require.NoError(t, bus.Publish(t.Context(), topic, *firstMsg))
	require.NoError(t, bus.Publish(t.Context(), topic, *secondMsg))

	assert.Equal(t, firstMsg.ID, waitForString(t, startedCh))

	select {
	case got := <-startedCh:
		t.Fatalf("second message started before first was released: %s", got)
	case <-time.After(100 * time.Millisecond):
	}

	releaseCh <- struct{}{}
	assert.Equal(t, firstMsg.ID, waitForString(t, finishedCh))

	assert.Equal(t, secondMsg.ID, waitForString(t, startedCh))
	releaseCh <- struct{}{}
	assert.Equal(t, secondMsg.ID, waitForString(t, finishedCh))
}

func TestBus_WithMiddlewareAppliesWrappedHandler(t *testing.T) {
	t.Parallel()

	bus := NewMemoryBus()
	topic := "memory.middleware"
	receivedCh := make(chan string, 1)

	middleware := func(next pubsub.Handler) pubsub.Handler {
		return pubsub.HandlerFunc(func(ctx context.Context, msg pubsub.Message) error {
			msg.Headers = map[string]string{"wrapped": "true"}
			return next.Handle(ctx, msg)
		})
	}

	sub, err := bus.Subscribe(
		t.Context(),
		topic,
		pubsub.HandlerFunc(func(ctx context.Context, msg pubsub.Message) error {
			receivedCh <- msg.Headers["wrapped"]
			return nil
		}),
		pubsub.WithMiddleware(middleware),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sub.Close(context.Background()))
	})

	msg, err := pubsub.NewMessage(topic, "", map[string]string{"kind": "middleware"})
	require.NoError(t, err)
	require.NoError(t, bus.Publish(t.Context(), topic, *msg))

	assert.Equal(t, "true", waitForString(t, receivedCh))
}

func waitForString(t *testing.T, ch <-chan string) string {
	t.Helper()

	select {
	case value := <-ch:
		return value
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for value")
		return ""
	}
}
