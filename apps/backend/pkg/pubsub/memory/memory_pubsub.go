package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pubsub"
)

var ErrClosed = errors.New("pubsub: closed")

type Bus struct {
	mu     sync.RWMutex
	topics map[string][]*memorySubscription
	closed bool
}

func NewMemoryBus() *Bus {
	return &Bus{
		topics: make(map[string][]*memorySubscription),
	}
}

func (b *Bus) Publish(ctx context.Context, topic string, msg pubsub.Message) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return ErrClosed
	}
	subs := append([]*memorySubscription(nil), b.topics[topic]...)
	b.mu.RUnlock()

	msg.Topic = topic
	if msg.PublishedAt.IsZero() {
		msg.PublishedAt = time.Now()
	}

	for _, sub := range subs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := sub.enqueue(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bus) Subscribe(ctx context.Context, topic string, handler pubsub.Handler, opts ...pubsub.SubscribeOption) (pubsub.Subscription, error) {
	cfg := pubsub.DefaultSubscribeOptions()
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.Ordered {
		cfg.Concurrency = 1
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}

	finalHandler := handler
	if len(cfg.Middlewares) > 0 {
		finalHandler = pubsub.Chain(handler, cfg.Middlewares...)
	}

	sub := &memorySubscription{
		bus:      b,
		topic:    topic,
		handler:  finalHandler,
		opts:     cfg,
		msgCh:    make(chan pubsub.Message, 256),
		closeCh:  make(chan struct{}),
		closedCh: make(chan struct{}),
	}

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil, ErrClosed
	}
	b.topics[topic] = append(b.topics[topic], sub)
	b.mu.Unlock()

	sub.start(ctx)
	return sub, nil
}

func (b *Bus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true

	var subs []*memorySubscription
	for _, group := range b.topics {
		subs = append(subs, group...)
	}
	b.topics = map[string][]*memorySubscription{}
	b.mu.Unlock()

	for _, s := range subs {
		_ = s.Close(context.Background())
	}
	return nil
}

type memorySubscription struct {
	bus      *Bus
	topic    string
	handler  pubsub.Handler
	opts     pubsub.SubscribeOptions
	msgCh    chan pubsub.Message
	closeCh  chan struct{}
	closedCh chan struct{}

	once sync.Once
	wg   sync.WaitGroup
}

func (s *memorySubscription) start(ctx context.Context) {
	workerCount := s.opts.Concurrency
	if workerCount <= 0 {
		workerCount = 1
	}

	for i := 0; i < workerCount; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-s.closeCh:
					return
				case msg := <-s.msgCh:
					s.process(ctx, msg)
				}
			}
		}()
	}

	go func() {
		s.wg.Wait()
		close(s.closedCh)
	}()
}

func (s *memorySubscription) enqueue(ctx context.Context, msg pubsub.Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closeCh:
		return ErrClosed
	case s.msgCh <- msg:
		return nil
	}
}

func (s *memorySubscription) process(ctx context.Context, msg pubsub.Message) {
	var err error

	for attempt := 0; attempt <= s.opts.MaxRetries; attempt++ {
		err = s.handler.Handle(ctx, msg)
		if err == nil {
			return
		}

		if attempt < s.opts.MaxRetries && s.opts.RetryBackoff > 0 {
			select {
			case <-time.After(s.opts.RetryBackoff):
			case <-ctx.Done():
				return
			case <-s.closeCh:
				return
			}
		}
	}
}

func (s *memorySubscription) Close(ctx context.Context) error {
	s.once.Do(func() {
		close(s.closeCh)

		s.bus.mu.Lock()
		defer s.bus.mu.Unlock()

		subs := s.bus.topics[s.topic]
		filtered := subs[:0]
		for _, sub := range subs {
			if sub != s {
				filtered = append(filtered, sub)
			}
		}
		if len(filtered) == 0 {
			delete(s.bus.topics, s.topic)
		} else {
			s.bus.topics[s.topic] = filtered
		}
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closedCh:
		return nil
	}
}
