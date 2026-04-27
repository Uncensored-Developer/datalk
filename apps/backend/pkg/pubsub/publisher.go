package pubsub

import (
	"context"
	"sync"

	"github.com/mdobak/go-xerrors"
)

var (
	ErrNoPublisher = xerrors.New("pubsub: no publisher")
	publisherMutex sync.Mutex
	publisher      Publisher
)

func RegisterPublisher(pub Publisher) {
	publisherMutex.Lock()
	defer publisherMutex.Unlock()
	publisher = pub
}

func GetPublisher() (Publisher, error) {
	if publisher == nil {
		return nil, ErrNoPublisher
	}
	return publisher, nil
}

func Send(ctx context.Context, msg *Message) error {
	publisher, err := GetPublisher()
	if err != nil {
		return err
	}
	return publisher.Publish(ctx, msg.Topic, *msg)
}
