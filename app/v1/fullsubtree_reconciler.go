package v1

import (
	"context"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	clientv1 "github.com/infratographer/fertilesoil/client/v1"
)

func newSeeder(
	baseDir apiv1.DirectoryID,
	cli clientv1.ReadOnlyClient,
	store AppStorage,
) (Seeder, error) {
	return newController(baseDir,
		WithClient(cli),
		withStorage(store),
		WithReconciler(&NoopReconciler{}))
}

// InitializeDirectories initializes the directories in the store.
// It checks if the base directory is up-to-date on the store.
// If it is not, it is persisted.
func (c *controller) InitializeDirectories(ctx context.Context) error {
	// not having the client is not an error, it just means that the controller
	// solely relies on events to update the store.
	if c.c == nil {
		return nil
	}

	err := c.persistIfUpToDate(ctx, c.baseDir)
	if err != nil {
		return err
	}

	// check if all subdirs are tracked and up-to-date, else, persist them.
	subdirs, err := c.c.GetChildren(ctx, c.baseDir, nil)
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

// persistIfUpToDate checks if the directory is up-to-date on the store.
// If it is not, it is persisted.
func (c *controller) persistIfUpToDate(ctx context.Context, dir apiv1.DirectoryID) error {
	// not having the client is not an error, it just means that the controller
	// solely relies on events to update the store.
	if c.c == nil {
		return nil
	}

	fd, err := c.c.GetDirectory(ctx, dir, nil)
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
