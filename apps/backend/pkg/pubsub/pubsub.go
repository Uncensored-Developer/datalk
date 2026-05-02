package pubsub

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID          string            `json:"id"`
	Topic       string            `json:"topic"`
	key         string            `json:"key"`
	Headers     map[string]string `json:"headers"`
	Body        []byte            `json:"body"`
	PublishedAt time.Time         `json:"published_at"`
	Metadata    map[string]any    `json:"metadata"`
}

func NewMessage(topic, key string, content any) (*Message, error) {
	c, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}

	return &Message{
		ID:          uuid.NewString(),
		Topic:       topic,
		key:         key,
		Body:        c,
		PublishedAt: time.Now(),
	}, nil
}

func (m *Message) DecodeBody(output any) error {
	return json.Unmarshal(m.Body, output)
}

//go:generate go tool with-modfile mockery --name Publisher --outpkg testing --output ./testing --filename generated__publisher_mocks.go
type Publisher interface {
	Publish(ctx context.Context, topic string, msg Message) error
}

//go:generate go tool with-modfile mockery --name Subscriber --outpkg testing --output ./testing --filename generated__subscriber_mocks.go
type Subscriber interface {
	Subscribe(ctx context.Context, topic string, handler Handler, opts ...SubscribeOption) (Subscription, error)
}

//go:generate go tool with-modfile mockery --name Handler --outpkg testing --output ./testing --filename generated__handler_mocks.go
type Handler interface {
	Handle(ctx context.Context, msg Message) error
}

//go:generate go tool with-modfile mockery --name Subscription --outpkg testing --output ./testing --filename generated__subscription_mocks.go
type Subscription interface {
	Close(ctx context.Context) error
}

type HandlerFunc func(ctx context.Context, msg Message) error

func (f HandlerFunc) Handle(ctx context.Context, msg Message) error {
	return f(ctx, msg)
}

type StartPosition int

const (
	StartLatest StartPosition = iota
	StartEarliest
)

type SubscribeOptions struct {
	Group        string
	Concurrency  int
	MaxRetries   int
	RetryBackoff time.Duration
	Ordered      bool
	StartAt      StartPosition
	Middlewares  []Middleware
}

type SubscribeOption func(*SubscribeOptions)

func DefaultSubscribeOptions() SubscribeOptions {
	return SubscribeOptions{
		Concurrency:  1,
		MaxRetries:   0,
		RetryBackoff: 0,
		StartAt:      StartLatest,
	}
}

func WithGroup(group string) SubscribeOption {
	return func(o *SubscribeOptions) { o.Group = group }
}

func WithConcurrency(n int) SubscribeOption {
	return func(o *SubscribeOptions) {
		if n > 0 {
			o.Concurrency = n
		}
	}
}

func WithMaxRetries(n int) SubscribeOption {
	return func(o *SubscribeOptions) {
		if n >= 0 {
			o.MaxRetries = n
		}
	}
}

func WithRetryBackoff(d time.Duration) SubscribeOption {
	return func(o *SubscribeOptions) { o.RetryBackoff = d }
}

func WithOrdered() SubscribeOption {
	return func(o *SubscribeOptions) {
		o.Ordered = true
		o.Concurrency = 1
	}
}

func WithMiddleware(mw ...Middleware) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.Middlewares = append(o.Middlewares, mw...)
	}
}
