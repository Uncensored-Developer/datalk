package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	pubsubpkg "github.com/Uncensored-Developer/datalk/apps/backend/pkg/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresBus_PublishSubscribe(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	topic := testTopic(t, "delivery")
	receivedCh := make(chan pubsubpkg.Message, 1)

	sub, err := bus.Subscribe(ctx, topic, pubsubpkg.HandlerFunc(func(ctx context.Context, msg pubsubpkg.Message) error {
		receivedCh <- msg
		return nil
	}))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sub.Close(context.Background()))
	})

	msg, err := pubsubpkg.NewMessage(topic, "ignored", map[string]any{"value": "ok"})
	require.NoError(t, err)
	msg.Headers = map[string]string{"x-test": "1"}
	msg.Metadata = map[string]any{"attempt": float64(1)}

	require.NoError(t, bus.Publish(ctx, topic, *msg))

	select {
	case got := <-receivedCh:
		assert.Equal(t, msg.ID, got.ID)
		assert.Equal(t, topic, got.Topic)
		assert.Equal(t, msg.Headers, got.Headers)
		assert.Equal(t, msg.Body, got.Body)
		assert.WithinDuration(t, msg.PublishedAt, got.PublishedAt, time.Second)

		var body map[string]string
		require.NoError(t, json.Unmarshal(got.Body, &body))
		assert.Equal(t, "ok", body["value"])
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for notification")
	}
}

func TestPostgresBus_MultipleSubscribersReceiveSameMessage(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	topic := testTopic(t, "fanout")
	firstCh := make(chan string, 1)
	secondCh := make(chan string, 1)

	sub1, err := bus.Subscribe(ctx, topic, pubsubpkg.HandlerFunc(func(ctx context.Context, msg pubsubpkg.Message) error {
		firstCh <- msg.ID
		return nil
	}))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sub1.Close(context.Background()))
	})

	sub2, err := bus.Subscribe(ctx, topic, pubsubpkg.HandlerFunc(func(ctx context.Context, msg pubsubpkg.Message) error {
		secondCh <- msg.ID
		return nil
	}))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sub2.Close(context.Background()))
	})

	msg, err := pubsubpkg.NewMessage(topic, "", map[string]string{"kind": "fanout"})
	require.NoError(t, err)
	require.NoError(t, bus.Publish(ctx, topic, *msg))

	assert.Equal(t, msg.ID, waitForString(t, firstCh))
	assert.Equal(t, msg.ID, waitForString(t, secondCh))
}

func TestPostgresBus_SubscriptionCloseStopsDelivery(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	topic := testTopic(t, "close")
	receivedCh := make(chan string, 1)

	sub, err := bus.Subscribe(ctx, topic, pubsubpkg.HandlerFunc(func(ctx context.Context, msg pubsubpkg.Message) error {
		receivedCh <- msg.ID
		return nil
	}))
	require.NoError(t, err)

	require.NoError(t, sub.Close(ctx))

	msg, err := pubsubpkg.NewMessage(topic, "", map[string]string{"kind": "closed"})
	require.NoError(t, err)
	require.NoError(t, bus.Publish(ctx, topic, *msg))

	select {
	case got := <-receivedCh:
		t.Fatalf("unexpected notification after close: %s", got)
	case <-time.After(250 * time.Millisecond):
	}
}

func TestPostgresBus_RetriesHandler(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	topic := testTopic(t, "retry")
	doneCh := make(chan struct{}, 1)
	var attempts atomic.Int32

	sub, err := bus.Subscribe(
		ctx,
		topic,
		pubsubpkg.HandlerFunc(func(ctx context.Context, msg pubsubpkg.Message) error {
			if attempts.Add(1) < 3 {
				return errors.New("transient")
			}
			doneCh <- struct{}{}
			return nil
		}),
		pubsubpkg.WithMaxRetries(2),
		pubsubpkg.WithRetryBackoff(25*time.Millisecond),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sub.Close(context.Background()))
	})

	msg, err := pubsubpkg.NewMessage(topic, "", map[string]string{"kind": "retry"})
	require.NoError(t, err)
	require.NoError(t, bus.Publish(ctx, topic, *msg))

	select {
	case <-doneCh:
		assert.Equal(t, int32(3), attempts.Load())
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for retried handler")
	}
}

func TestPostgresBus_PayloadTooLarge(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	topic := testTopic(t, "too-large")
	msg := pubsubpkg.Message{
		ID:          "too-large",
		Topic:       topic,
		Body:        []byte(string(make([]byte, maxNotificationBytes))),
		PublishedAt: time.Now(),
	}

	err := bus.Publish(ctx, topic, msg)
	require.ErrorIs(t, err, ErrPayloadTooLarge)
}

func TestPostgresBus_Close(t *testing.T) {
	localBus := NewBus(runner.Conn, connInfo)

	sub, err := localBus.Subscribe(t.Context(), testTopic(t, "bus-close"), pubsubpkg.HandlerFunc(func(ctx context.Context, msg pubsubpkg.Message) error {
		return nil
	}))
	require.NoError(t, err)

	assert.NoError(t, localBus.Close())
	assert.NoError(t, sub.Close(context.Background()))

	msg, err := pubsubpkg.NewMessage(testTopic(t, "bus-close"), "", map[string]string{"kind": "closed"})
	require.NoError(t, err)
	assert.ErrorIs(t, localBus.Publish(t.Context(), msg.Topic, *msg), ErrClosed)
}

func testTopic(t *testing.T, suffix string) string {
	t.Helper()
	return fmt.Sprintf("pubsub.%s.%s", t.Name(), suffix)
}

func waitForString(t *testing.T, ch <-chan string) string {
	t.Helper()

	select {
	case value := <-ch:
		return value
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for message")
		return ""
	}
}
