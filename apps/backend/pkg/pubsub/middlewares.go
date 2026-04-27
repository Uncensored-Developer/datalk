package pubsub

import (
	"context"
	"log"
	"time"
)

type Middleware func(Handler) Handler

func Chain(h Handler, mws ...Middleware) Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func RecoverPanic() Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, msg Message) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = &PanicError{Value: r}
				}
			}()
			return next.Handle(ctx, msg)
		})
	}
}

func Retry(maxRetries int, backoff time.Duration) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, msg Message) error {
			var err error
			for attempt := 0; attempt <= maxRetries; attempt++ {
				err = next.Handle(ctx, msg)
				if err == nil {
					return nil
				}
				if attempt == maxRetries {
					return err
				}
				if backoff > 0 {
					select {
					case <-time.After(backoff):
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
			return err
		})
	}
}

func Timeout(d time.Duration) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, msg Message) error {
			cctx, cancel := context.WithTimeout(ctx, d)
			defer cancel()
			return next.Handle(cctx, msg)
		})
	}
}

func Logging(logger *log.Logger) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, msg Message) error {
			start := time.Now()
			err := next.Handle(ctx, msg)
			if logger != nil {
				if err != nil {
					logger.Printf("topic=%s msg_id=%s status=error dur=%s err=%v", msg.Topic, msg.ID, time.Since(start), err)
				} else {
					logger.Printf("topic=%s msg_id=%s status=ok dur=%s", msg.Topic, msg.ID, time.Since(start))
				}
			}
			return err
		})
	}
}

type PanicError struct {
	Value any
}

func (e *PanicError) Error() string {
	return "pubsub: handler panic"
}
