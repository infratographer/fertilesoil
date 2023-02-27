package nats

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	natsgo "github.com/nats-io/nats.go"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/notifier"
)

// NewNotifier creates a new NATS notifier using JetStream.
// The subject is the NATS subject to publish events to.
func NewNotifier(js natsgo.JetStreamContext, subject string) notifier.Notifier {
	return &natsnotifier{
		js:      js,
		subject: subject,
	}
}

type natsnotifier struct {
	js      natsgo.JetStreamContext
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
	var buff bytes.Buffer

	if err := json.NewEncoder(&buff).Encode(evt); err != nil {
		return fmt.Errorf("failed encoding event: %w", err)
	}

	if _, err := n.js.Publish(n.subject, buff.Bytes()); err != nil {
		return fmt.Errorf("failed to publish event to: %w", err)
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
