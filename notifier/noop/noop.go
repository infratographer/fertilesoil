package noop

import (
	"context"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/notifier"
)

func NewNotifier() notifier.Notifier {
	return &noopNotifier{}
}

type noopNotifier struct{}

// ensure noopNotifier implements notifier.Notifier.
var _ notifier.Notifier = &noopNotifier{}

func (n *noopNotifier) NotifyCreate(ctx context.Context, d *apiv1.Directory) error {
	return nil
}

func (n *noopNotifier) NotifyUpdate(ctx context.Context, d *apiv1.Directory) error {
	return nil
}

func (n *noopNotifier) NotifyDeleteSoft(ctx context.Context, d *apiv1.Directory) error {
	return nil
}

func (n *noopNotifier) NotifyDeleteHard(ctx context.Context, d *apiv1.Directory) error {
	return nil
}
