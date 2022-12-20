package v1

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	clientv1 "github.com/infratographer/fertilesoil/client/v1"
)

type controller struct {
	c       clientv1.Client
	w       clientv1.Watcher
	baseDir apiv1.DirectoryID
	store   AppStorage
	r       Reconciler
}

// NewController creates a new controller.
// It takes a base directory and a list of options.
// The base directory can be any directory in the directory tree.
func newController(baseDir apiv1.DirectoryID, opts ...Option) (Controller, error) {
	c := &controller{
		baseDir: baseDir,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.r == nil {
		return nil, ErrNoReconciler
	}

	return c, nil
}

func withReconciler(r Reconciler) Option {
	return func(c *controller) {
		c.r = r
	}
}

func withClient(cli clientv1.Client) Option {
	return func(c *controller) {
		c.c = cli
	}
}

func withStorage(store AppStorage) Option {
	return func(c *controller) {
		c.store = store
	}
}

func withWatcher(w clientv1.Watcher) Option {
	return func(c *controller) {
		c.w = w
	}
}

func (c *controller) Run(ctx context.Context) error {
	// initialize ticker to check for updates at a random interval
	ticker := time.NewTicker(getRandomTickerDuration())

	// initialize directories
	err := c.initializeDirectories(ctx)
	if err != nil {
		return err
	}

	// start watching for events
	evCh, errCh := c.w.Watch(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			err := c.initializeDirectories(ctx)
			if err != nil {
				return err
			}
			// reset ticker to check for updates at a random interval
			ticker.Reset(getRandomTickerDuration())
		case err := <-errCh:
			return err
		case ev := <-evCh:
			err := c.processIncomingEvent(ctx, ev)
			if err != nil {
				return err
			}
		}
	}
}

func (c *controller) initializeDirectories(ctx context.Context) error {
	err := c.persistIfUpToDate(ctx, c.baseDir)
	if err != nil {
		return err
	}

	// check if all subdirs are tracked and up-to-date, else, persist them.
	subdirs, err := c.c.GetChildren(ctx, c.baseDir)
	if err != nil {
		return err
	}

	for _, subdir := range subdirs.Directories {
		err := c.persistIfUpToDate(ctx, subdir)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *controller) persistIfUpToDate(ctx context.Context, dir apiv1.DirectoryID) error {
	fd, err := c.c.GetDirectory(ctx, dir)
	if err != nil {
		return err
	}

	d := &fd.Directory
	upToDate, err := c.store.IsDirectoryInfoUpdated(ctx, d)
	if err != nil {
		return err
	}

	if upToDate {
		return nil
	}

	return c.persistDirectory(ctx, d)
}

// persistDirectory persists the directory in the store.
// If the directory is not up-to-date on the store, the reconciler is called.
func (c *controller) persistDirectory(ctx context.Context, d *apiv1.Directory) error {
	// handle deletion
	if d.IsDeleted() {
		err := c.store.DeleteDirectory(ctx, d.ID)
		if err != nil {
			return err
		}
		return c.r.Reconcile(ctx, apiv1.DirectoryEvent{
			DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
				Version: apiv1.APIVersion,
			},
			Time:      time.Now().UTC(),
			Type:      apiv1.EventTypeDelete,
			Directory: *d,
		})
	}

	_, err := c.store.CreateDirectory(ctx, d)
	if err != nil {
		return err
	}

	return c.r.Reconcile(ctx, apiv1.DirectoryEvent{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Time:      time.Now().UTC(),
		Type:      apiv1.EventTypeCreate,
		Directory: *d,
	})
}

func (c *controller) processIncomingEvent(ctx context.Context, ev *apiv1.DirectoryEvent) error {
	isRelevant, err := c.isRelevantEvent(ctx, ev)
	if err != nil {
		return fmt.Errorf("error checking if directory is tracked: %w", err)
	}

	if !isRelevant {
		return nil
	}

	if err = c.persistDirectory(ctx, &ev.Directory); err != nil {
		return fmt.Errorf("error persisting directory: %w", err)
	}

	return c.r.Reconcile(ctx, *ev)
}

func (c *controller) isRelevantEvent(ctx context.Context, ev *apiv1.DirectoryEvent) (bool, error) {
	d := &ev.Directory

	tracking, err := c.store.IsDirectoryTracked(ctx, d.ID)
	if err != nil {
		return false, fmt.Errorf("error checking if directory is tracked: %w", err)
	}

	// If we're tracking this directory, we can react to it.
	if tracking {
		return true, nil
	}

	// Otherwise, we only care if it's a create event and
	// if it's a subdirectory of a directory we're already tracking.
	if ev.Type != apiv1.EventTypeCreate {
		return false, nil
	}

	// New root directory, we can skip this as we're not tracking it.
	if d.Parent == nil {
		return false, nil
	}

	trackingParent, err := c.store.IsDirectoryTracked(ctx, *d.Parent)
	if err != nil {
		return false, fmt.Errorf("error checking if parent directory is tracked: %w", err)
	}

	// If we're tracking the parent, we can react to it.
	return trackingParent, nil
}

// getRandomTickerDuration returns a random duration between 5 and 30 minutes.
func getRandomTickerDuration() time.Duration {
	const minimumInterval = 5
	const maximumInterval = 30
	//nolint:gosec // not used for crypto. just a random number to set an interval.
	return time.Duration(minimumInterval+rand.Intn(maximumInterval)) * time.Minute
}
