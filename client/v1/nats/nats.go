package nats

import (
	"context"
	"fmt"

	natsgo "github.com/nats-io/nats.go"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	clientv1 "github.com/infratographer/fertilesoil/client/v1"
)

// subcriber implements a clientv1.Subscriber interface.
type subscriber struct {
	conn *natsgo.EncodedConn
	subj string
}

// NewSubscriber returns a new clientv1.Subscriber.
func NewSubscriber(conn *natsgo.Conn, subj string) (clientv1.Watcher, error) {
	enc, err := natsgo.NewEncodedConn(conn, natsgo.JSON_ENCODER)
	if err != nil {
		return nil, fmt.Errorf("failed to create encoded connection: %w", err)
	}

	return &subscriber{
		conn: enc,
		subj: subj,
	}, nil
}

// Watch implements clientv1.Subscriber.
// It actively listens for events on the NATS subject.
func (s *subscriber) Watch(ctx context.Context) (eventsChan <-chan *apiv1.DirectoryEvent, errorsChan <-chan error) {
	events := make(chan *apiv1.DirectoryEvent)
	errs := make(chan error)

	go func() {
		defer close(events)
		defer close(errs)

		_, err := s.conn.Subscribe(s.subj, func(e *apiv1.DirectoryEvent) {
			select {
			case <-ctx.Done():
				return
			case events <- e:
			}
		})
		if err != nil {
			errs <- fmt.Errorf("failed to subscribe to subject: %w", err)
			return
		}

		<-ctx.Done()
		s.conn.Close()
	}()

	return events, errs
}
