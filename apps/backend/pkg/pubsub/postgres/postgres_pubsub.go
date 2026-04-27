package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	pubsubpkg "github.com/Uncensored-Developer/datalk/apps/backend/pkg/pubsub"
	"github.com/lib/pq"
	"github.com/mdobak/go-xerrors"
)

const (
	channelPrefix        = "datalk_pubsub_"
	maxNotificationBytes = 8000
)

var (
	ErrClosed                  = xerrors.New("pubsub: closed")
	ErrPayloadTooLarge         = xerrors.New("pubsub: payload exceeds postgres NOTIFY limit")
	ErrMissingListenerConnInfo = xerrors.New("pubsub: missing postgres listener connection info")
)

type Bus struct {
	db       *sql.DB
	connInfo string

	mu     sync.Mutex
	subs   map[*subscription]struct{}
	closed bool
}

func NewBus(db *sql.DB, connInfo string) *Bus {
	return &Bus{
		db:       db,
		connInfo: connInfo,
		subs:     make(map[*subscription]struct{}),
	}
}

func (b *Bus) Publish(ctx context.Context, topic string, msg pubsubpkg.Message) error {
	b.mu.Lock()
	closed := b.closed
	b.mu.Unlock()
	if closed {
		return ErrClosed
	}

	msg.Topic = topic
	if msg.PublishedAt.IsZero() {
		msg.PublishedAt = time.Now()
	}

	payload, err := encodeMessage(msg)
	if err != nil {
		return err
	}
	if len(payload) > maxNotificationBytes {
		return ErrPayloadTooLarge
	}

	_, err = b.db.ExecContext(ctx, "SELECT pg_notify($1, $2)", topicChannel(topic), payload)
	return err
}

func (b *Bus) Subscribe(ctx context.Context, topic string, handler pubsubpkg.Handler, opts ...pubsubpkg.SubscribeOption) (pubsubpkg.Subscription, error) {
	if b.connInfo == "" {
		return nil, ErrMissingListenerConnInfo
	}

	cfg := pubsubpkg.DefaultSubscribeOptions()
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
		finalHandler = pubsubpkg.Chain(handler, cfg.Middlewares...)
	}

	listener := pq.NewListener(b.connInfo, 10*time.Second, time.Minute, nil)
	channel := topicChannel(topic)
	if err := listener.Listen(channel); err != nil {
		_ = listener.Close()
		return nil, err
	}

	sub := &subscription{
		bus:      b,
		listener: listener,
		topic:    topic,
		handler:  finalHandler,
		opts:     cfg,
		msgCh:    make(chan pubsubpkg.Message, 256),
		closeCh:  make(chan struct{}),
		closedCh: make(chan struct{}),
	}

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		_ = listener.UnlistenAll()
		_ = listener.Close()
		return nil, ErrClosed
	}
	b.subs[sub] = struct{}{}
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

	subs := make([]*subscription, 0, len(b.subs))
	for sub := range b.subs {
		subs = append(subs, sub)
	}
	b.mu.Unlock()

	for _, sub := range subs {
		_ = sub.Close(context.Background())
	}

	return nil
}

func (b *Bus) removeSubscription(sub *subscription) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.subs, sub)
}

type messageEnvelope struct {
	ID          string            `json:"id"`
	Topic       string            `json:"topic"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        []byte            `json:"body"`
	PublishedAt time.Time         `json:"published_at"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
}

func encodeMessage(msg pubsubpkg.Message) (string, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func decodeMessage(payload string) (pubsubpkg.Message, error) {
	var msg pubsubpkg.Message
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		return pubsubpkg.Message{}, err
	}
	return msg, nil
}

func topicChannel(topic string) string {
	sum := sha256.Sum256([]byte(topic))
	return channelPrefix + hex.EncodeToString(sum[:24])
}

type subscription struct {
	bus      *Bus
	listener *pq.Listener
	topic    string
	handler  pubsubpkg.Handler
	opts     pubsubpkg.SubscribeOptions
	msgCh    chan pubsubpkg.Message
	closeCh  chan struct{}
	closedCh chan struct{}

	once sync.Once
	wg   sync.WaitGroup
}

func (s *subscription) start(ctx context.Context) {
	workerCount := s.opts.Concurrency
	if workerCount <= 0 {
		workerCount = 1
	}

	for range workerCount {
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

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		pingTicker := time.NewTicker(30 * time.Second)
		defer pingTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-s.closeCh:
				return
			case notification, ok := <-s.listener.Notify:
				if !ok {
					return
				}
				if notification == nil {
					continue
				}

				msg, err := decodeMessage(notification.Extra)
				if err != nil {
					continue
				}
				if msg.Topic == "" {
					msg.Topic = s.topic
				}
				if err := s.enqueue(ctx, msg); err != nil {
					return
				}
			case <-pingTicker.C:
				go s.listener.Ping()
			}
		}
	}()

	go func() {
		s.wg.Wait()
		close(s.closedCh)
	}()
}

func (s *subscription) enqueue(ctx context.Context, msg pubsubpkg.Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closeCh:
		return ErrClosed
	case s.msgCh <- msg:
		return nil
	}
}

func (s *subscription) process(ctx context.Context, msg pubsubpkg.Message) {
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

func (s *subscription) Close(ctx context.Context) error {
	s.once.Do(func() {
		close(s.closeCh)
		s.bus.removeSubscription(s)
		_ = s.listener.UnlistenAll()
		_ = s.listener.Close()
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closedCh:
		return nil
	}
}

var (
	_ pubsubpkg.Publisher    = (*Bus)(nil)
	_ pubsubpkg.Subscriber   = (*Bus)(nil)
	_ pubsubpkg.Subscription = (*subscription)(nil)
)
