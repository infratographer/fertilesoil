package nats

import (
	"context"
	"fmt"
	"time"

	natsgo "github.com/nats-io/nats.go"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/notifier"
)

// NewNotifier creates a new NATS notifier.
// The subject is the NATS subject to publish events to.
// The `*nats.Conn` object is passed directly so the caller can configure the
// connection as they see fit. e.g. to use TLS, JWT tokens, set a custom timeout, etc.
func NewNotifier(nc *natsgo.Conn, subject string) (notifier.Notifier, error) {
	c, err := natsgo.NewEncodedConn(nc, natsgo.JSON_ENCODER)
	if err != nil {
		return nil, fmt.Errorf("failed to create encoded connection: %w", err)
	}

	return &natsnotifier{
		c:       c,
		subject: subject,
	}, nil
}

type natsnotifier struct {
	c       *natsgo.EncodedConn
	subject string
}

func (n *natsnotifier) NotifyCreate(ctx context.Context, d *apiv1.Directory) error {
	return n.publish(getEvent(apiv1.EventTypeCreate, d))
}

func (n *natsnotifier) NotifyUpdate(ctx context.Context, d *apiv1.Directory) error {
	return n.publish(getEvent(apiv1.EventTypeUpdate, d))
}

func (n *natsnotifier) NotifyDelete(ctx context.Context, d *apiv1.Directory) error {
	return n.publish(getEvent(apiv1.EventTypeDelete, d))
}

func (n *natsnotifier) NotifyDeleteHard(ctx context.Context, d *apiv1.Directory) error {
	return n.publish(getEvent(apiv1.EventTypeDeleteHard, d))
}

func (n *natsnotifier) publish(evt *apiv1.DirectoryEvent) error {
	if err := n.c.Publish(n.subject, evt); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

func getEvent(evtType apiv1.EventType, d *apiv1.Directory) *apiv1.DirectoryEvent {
	return &apiv1.DirectoryEvent{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Type:      evtType,
		Time:      time.Now().UTC(),
		Directory: *d,
	}
}
