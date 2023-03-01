package nats

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	nats "github.com/nats-io/nats.go"
	"go.uber.org/zap"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
)

// NewNotifier creates a new NATS notifier using JetStream.
// The subject is the NATS subject to publish events to.
func NewNotifier(js nats.JetStreamContext, subjectPrefix string, options ...Option) *Notifier {
	n := &Notifier{
		logger:        zap.NewNop(),
		js:            js,
		subjectPrefix: subjectPrefix,
	}

	for _, opt := range options {
		opt(n)
	}

	return n
}

// Option is a functional configuration option for governor eventing.
type Option func(c *Notifier)

// WithLogger sets the logger.
func WithLogger(l *zap.Logger) Option {
	return func(n *Notifier) {
		n.logger = l
	}
}

// WithPublishOptions sets the nats publish options.
func WithPublishOptions(options ...nats.PubOpt) Option {
	return func(n *Notifier) {
		n.publishOptions = options
	}
}

// Notifier implements NATS notification handling.
type Notifier struct {
	logger         *zap.Logger
	js             nats.JetStreamContext
	subjectPrefix  string
	publishOptions []nats.PubOpt
}

// AddStream checks if a stream exists and attempts to create it if it doesn't.
// Currently we don't check that the stream is configured identically to the desired configuration.
func (n *Notifier) AddStream(stream *nats.StreamConfig) (*nats.StreamInfo, error) {
	info, err := n.js.StreamInfo(stream.Name)
	if err == nil {
		n.logger.Debug("got info for stream, assuming stream exists", zap.Any("nats.stream.info", info.Config))

		return info, nil
	} else if !errors.Is(err, nats.ErrStreamNotFound) {
		n.logger.Error("failed to get stream info", zap.Error(err))

		return nil, err
	}

	n.logger.Debug("nats stream not found, attempting to create it", zap.String("nats.stream.name", stream.Name))

	// Ensure we're capturing each action.
	stream.Subjects = append(stream.Subjects, n.subjectPrefix+".>")

	return n.js.AddStream(stream)
}

// NotifyCreate publishes a create event for the provided directory.
func (n *Notifier) NotifyCreate(ctx context.Context, d *apiv1.Directory) error {
	return n.publish(getEvent(apiv1.EventTypeCreate, d))
}

// NotifyUpdate publishes an update event for the provided directory.
func (n *Notifier) NotifyUpdate(ctx context.Context, d *apiv1.Directory) error {
	return n.publish(getEvent(apiv1.EventTypeUpdate, d))
}

// NotifyDelete publishes a delete event for the provided directory.
func (n *Notifier) NotifyDelete(ctx context.Context, d *apiv1.Directory) error {
	return n.publish(getEvent(apiv1.EventTypeDelete, d))
}

// NotifyDeleteHard publishes a hard delete event for the provided directory.
func (n *Notifier) NotifyDeleteHard(ctx context.Context, d *apiv1.Directory) error {
	return n.publish(getEvent(apiv1.EventTypeDeleteHard, d))
}

func (n *Notifier) publish(evt *apiv1.DirectoryEvent) error {
	var buff bytes.Buffer

	if err := json.NewEncoder(&buff).Encode(evt); err != nil {
		return fmt.Errorf("failed encoding event: %w", err)
	}

	subject := n.subjectPrefix + "." + string(evt.Type)

	n.logger.Debug("Sending event", zap.String("nats.publish.subject", subject), zap.Any("nats.publish.body", evt))

	if _, err := n.js.Publish(subject, buff.Bytes(), n.publishOptions...); err != nil {
		n.logger.Debug("Failed to send event",
			zap.String("nats.publish.subject", subject),
			zap.Any("nats.publish.body", evt),
			zap.Error(err),
		)

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
