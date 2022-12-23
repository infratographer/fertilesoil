// Package memory implements a memory storage backend for the
// fertilesoil storage interface.
// This is not meant to be the most useful nor the most performant
// storage backend, but rather a reference implementation which
// is useful for testing and development.
package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	v1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/storage"
)

type Driver struct {
	// dirMap is a thread-safe map of directories.
	dirMap sync.Map
}

func NewDirectoryDriver() *Driver {
	return &Driver{}
}

var _ storage.DirectoryAdmin = (*Driver)(nil)

// CreateRoot creates a root directory.
// Root directories are directories that have no parent directory.
// ID is generated by the database, it will be ignored if given.
func (t *Driver) CreateRoot(ctx context.Context, d *v1.Directory) (*v1.Directory, error) {
	if d.Parent != nil {
		return nil, storage.ErrRootWithParentDirectory
	}

	if d.Metadata == nil {
		d.Metadata = v1.DirectoryMetadata{}
	}

	d.ID = v1.DirectoryID(uuid.New())

	rawdir, _ := t.dirMap.LoadOrStore(d.ID, d)

	dir, ok := rawdir.(*v1.Directory)
	if !ok {
		panic("invalid type in dirMap")
	}

	return dir, nil
}

// ListRoots lists all root directories.
func (t *Driver) ListRoots(ctx context.Context) ([]v1.DirectoryID, error) {
	var roots []v1.DirectoryID

	t.dirMap.Range(func(key, value interface{}) bool {
		dir, ok := value.(*v1.Directory)
		if !ok {
			panic("invalid type in dirMap")
		}

		if dir.Parent == nil {
			roots = append(roots, dir.ID)
		}

		return true
	})

	return roots, nil
}

// CreateDirectory creates a directory.
// ID is generated by the database, it will be ignored if given.
func (t *Driver) CreateDirectory(ctx context.Context, d *v1.Directory) (*v1.Directory, error) {
	if d.Parent == nil {
		return nil, storage.ErrDirectoryWithoutParent
	}

	if d.Metadata == nil {
		d.Metadata = v1.DirectoryMetadata{}
	}

	d.ID = v1.DirectoryID(uuid.New())

	rawdir, _ := t.dirMap.LoadOrStore(d.ID, d)

	dir, ok := rawdir.(*v1.Directory)
	if !ok {
		panic("invalid type in dirMap")
	}

	return dir, nil
}

// DeleteDirectory deletes a directory.
func (t *Driver) DeleteDirectory(ctx context.Context, id v1.DirectoryID) error {
	t.dirMap.Delete(id)
	return nil
}

// GetDirectory gets a directory by ID.
func (t *Driver) GetDirectory(ctx context.Context, id v1.DirectoryID) (*v1.Directory, error) {
	rawdir, ok := t.dirMap.Load(id)
	if !ok {
		return nil, storage.ErrDirectoryNotFound
	}

	dir, ok := rawdir.(*v1.Directory)
	if !ok {
		panic("invalid type in dirMap")
	}

	return dir, nil
}

// GetParents gets all parent directories of a directory.
func (t *Driver) GetParents(ctx context.Context, id v1.DirectoryID) ([]v1.DirectoryID, error) {
	var parents []v1.DirectoryID

	for {
		dir, err := t.GetDirectory(ctx, id)
		if err != nil {
			return nil, err
		}

		if dir.Parent == nil {
			break
		}

		parents = append(parents, *dir.Parent)
		id = *dir.Parent
	}

	return parents, nil
}

// GetParentsUntilAncestor gets all parent directories of a directory
// until the ancestor directory is reached.
func (t *Driver) GetParentsUntilAncestor(
	ctx context.Context,
	child,
	ancestor v1.DirectoryID,
) ([]v1.DirectoryID, error) {
	var parents []v1.DirectoryID

	// verify that ancestor indeed exists
	_, err := t.GetDirectory(ctx, ancestor)
	if err != nil {
		return nil, err
	}

	for {
		dir, err := t.GetDirectory(ctx, child)
		if err != nil {
			return nil, err
		}

		if dir.Parent == nil && dir.ID != ancestor {
			return nil, storage.ErrDirectoryNotFound
		}

		parents = append(parents, *dir.Parent)
		child = *dir.Parent

		if child == ancestor {
			break
		}
	}

	return parents, nil
}

// GetChildren gets all child directories of a directory.
func (t *Driver) GetChildren(ctx context.Context, id v1.DirectoryID) ([]v1.DirectoryID, error) {
	var children []v1.DirectoryID

	t.dirMap.Range(func(key, value interface{}) bool {
		dir, ok := value.(*v1.Directory)
		if !ok {
			panic("invalid type in dirMap")
		}

		if dir.Parent != nil && *dir.Parent == id {
			children = append(children, dir.ID)
		}

		return true
	})

	if len(children) == 0 {
		return children, nil
	}

	// append the children's children
	for _, child := range children {
		c, err := t.GetChildren(ctx, child)
		if err != nil {
			return nil, err
		}

		children = append(children, c...)
	}

	return children, nil
}
