package notifier

import (
	"context"
	"errors"
	"fmt"

	"github.com/cenkalti/backoff/v4"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	nif "github.com/infratographer/fertilesoil/notifier"
	"github.com/infratographer/fertilesoil/storage"
)

var ErrNotifyFailed = errors.New("notification failed")

type Option func(*notifierWithStorage)

// StorageWithNotifier is a meta-storage driver that wraps a storage.DirectoryAdmin
// and notifies a notifier.Notifier on every operation.
// The notifier is called after the operation has been completed.
func StorageWithNotifier(s storage.DirectoryAdmin, n nif.Notifier, opts ...Option) storage.DirectoryAdmin {
	nws := &notifierWithStorage{
		DirectoryAdmin: s,
		notifier:       n,
	}

	for _, opt := range opts {
		opt(nws)
	}

	return nws
}

// WithNotifyRetrier wraps the notifier with a retry mechanism.
func WithNotifyRetrier() Option {
	return func(n *notifierWithStorage) {
		n.addWrapper(func(ctx context.Context, h handler) error {
			return backoff.Retry(func() error {
				return h(ctx)
			}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
		})
	}
}

type notifierWithStorage struct {
	storage.DirectoryAdmin
	notifier      nif.Notifier
	notifyWrapper wrapper
}

// ensure notifier implements storage.DirectoryAdmin.
var _ storage.DirectoryAdmin = &notifierWithStorage{}

// handles a notification.
type handler func(context.Context) error

// wraps a handler.
type wrapper func(context.Context, handler) error

func (n *notifierWithStorage) CreateDirectory(ctx context.Context, d *apiv1.Directory) (*apiv1.Directory, error) {
	d, err := n.DirectoryAdmin.CreateDirectory(ctx, d)
	if err != nil {
		return nil, err
	}

	err = n.notifyWrapper(ctx, func(ctx context.Context) error {
		if err := n.notifier.NotifyCreate(ctx, d); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return d, fmt.Errorf("%w: %v", ErrNotifyFailed, err)
	}
	return d, nil
}

func (n *notifierWithStorage) DeleteDirectory(ctx context.Context, id apiv1.DirectoryID) error {
	// TODO(jaosorior): Implement soft-delete and notify for all children.
	return nil
}

func (n *notifierWithStorage) CreateRoot(ctx context.Context, d *apiv1.Directory) (*apiv1.Directory, error) {
	d, err := n.DirectoryAdmin.CreateRoot(ctx, d)
	if err != nil {
		return nil, err
	}

	err = n.notifyWrapper(ctx, func(ctx context.Context) error {
		if err := n.notifier.NotifyCreate(ctx, d); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return d, fmt.Errorf("%w: %v", ErrNotifyFailed, err)
	}
	return d, nil
}

// passthrough functions.
func (n *notifierWithStorage) GetDirectory(ctx context.Context, id apiv1.DirectoryID) (*apiv1.Directory, error) {
	return n.DirectoryAdmin.GetDirectory(ctx, id)
}

func (n *notifierWithStorage) GetParents(ctx context.Context, id apiv1.DirectoryID) ([]apiv1.DirectoryID, error) {
	return n.DirectoryAdmin.GetParents(ctx, id)
}

func (n *notifierWithStorage) GetParentsUntilAncestor(ctx context.Context,
	child apiv1.DirectoryID,
	ancestor apiv1.DirectoryID,
) ([]apiv1.DirectoryID, error) {
	return n.DirectoryAdmin.GetParentsUntilAncestor(ctx, child, ancestor)
}

func (n *notifierWithStorage) GetChildren(ctx context.Context, id apiv1.DirectoryID) ([]apiv1.DirectoryID, error) {
	return n.DirectoryAdmin.GetChildren(ctx, id)
}

func (n *notifierWithStorage) addWrapper(w wrapper) {
	wrap := n.notifyWrapper
	if wrap == nil {
		n.notifyWrapper = w
		return
	}
	n.notifyWrapper = func(ctx context.Context, h handler) error {
		return wrap(ctx, func(ctx context.Context) error {
			return w(ctx, h)
		})
	}
}
