// Package memory implements a memory storage backend for the
// fertilesoil storage interface.
// This is not meant to be the most useful nor the most performant
// storage backend, but rather a reference implementation which
// is useful for testing and development.
package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	v1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/storage"
)

type Options func(*Driver)

type Driver struct {
	// dirMap is a thread-safe map of directories.
	dirMap *sync.Map
}

// WithDirectoryMap allows to set a custom directory map.
// This is useful for testing, since it allows to inject a custom
// map and further modify it in the test.
func WithDirectoryMap(dirMap *sync.Map) Options {
	return func(d *Driver) {
		d.dirMap = dirMap
	}
}

func NewDirectoryDriver(opts ...Options) *Driver {
	d := &Driver{
		dirMap: &sync.Map{},
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
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
		d.Metadata = &v1.DirectoryMetadata{}
	}

	d.Id = v1.DirectoryID(uuid.New())

	rawdir, _ := t.dirMap.LoadOrStore(d.Id, d)

	dir, ok := rawdir.(*v1.Directory)
	if !ok {
		return nil, fmt.Errorf("directory %s is not of type *v1.Directory", d.Id)
	}

	return dir, nil
}

// ListRoots lists all root directories.
func (t *Driver) ListRoots(ctx context.Context, opts *storage.ListOptions) ([]v1.DirectoryID, error) {
	var roots []v1.DirectoryID

	var iterationErr error

	t.dirMap.Range(func(key, value interface{}) bool {
		dir, ok := value.(*v1.Directory)
		if !ok {
			iterationErr = fmt.Errorf("found directory that is not of type *v1.Directory")
			return false
		}

		if (dir.DeletedAt == nil || opts.IsWithDeleted()) && dir.Parent == nil {
			roots = append(roots, dir.Id)
		}

		return true
	})

	if iterationErr != nil {
		return nil, iterationErr
	}

	return roots, nil
}

// CreateDirectory creates a directory.
// ID is generated by the database, it will be ignored if given.
func (t *Driver) CreateDirectory(ctx context.Context, d *v1.Directory) (*v1.Directory, error) {
	if d.Parent == nil {
		return nil, storage.ErrDirectoryWithoutParent
	}

	if d.Metadata == nil {
		d.Metadata = &v1.DirectoryMetadata{}
	}

	d.Id = v1.DirectoryID(uuid.New())

	rawdir, _ := t.dirMap.LoadOrStore(d.Id, d)

	dir, ok := rawdir.(*v1.Directory)
	if !ok {
		return nil, fmt.Errorf("directory %s is not of type *v1.Directory", d.Id)
	}

	return dir, nil
}

// DeleteDirectory deletes a directory.
func (t *Driver) DeleteDirectory(ctx context.Context, id v1.DirectoryID) ([]*v1.Directory, error) {
	dir, err := t.GetDirectory(ctx, id, nil)
	if err != nil {
		return nil, err
	}

	if dir.Parent == nil {
		return nil, storage.ErrDirectoryNotFound
	}

	children, err := t.GetChildren(ctx, id, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting children: %w", err)
	}

	deletedTime := time.Now()

	// length is the requested directory plus the count of children
	affected := make([]*v1.Directory, 1+len(children))

	dir.DeletedAt = &deletedTime

	affected[0] = dir

	for i, childID := range children {
		child, err := t.GetDirectory(ctx, childID, nil)
		if err != nil {
			return nil, fmt.Errorf("error getting child: %s: %w", childID, err)
		}

		child.DeletedAt = &deletedTime

		affected[i+1] = child
	}

	return affected, nil
}

// GetDirectory gets a directory by ID.
func (t *Driver) GetDirectory(ctx context.Context, id v1.DirectoryID, opts *storage.GetOptions) (*v1.Directory, error) {
	rawdir, ok := t.dirMap.Load(id)
	if !ok {
		return nil, storage.ErrDirectoryNotFound
	}

	dir, ok := rawdir.(*v1.Directory)
	if !ok {
		return nil, fmt.Errorf("directory %s is not of type *v1.Directory", id)
	}

	if dir.DeletedAt != nil && !opts.IsWithDeleted() {
		return nil, storage.ErrDirectoryNotFound
	}

	return dir, nil
}

// GetParents gets all parent directories of a directory.
func (t *Driver) GetParents(
	ctx context.Context,
	id v1.DirectoryID,
	opts *storage.ListOptions,
) ([]v1.DirectoryID, error) {
	var parents []v1.DirectoryID

	for {
		dir, err := t.GetDirectory(ctx, id, opts.ToGetOptions())
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
	opts *storage.ListOptions,
) ([]v1.DirectoryID, error) {
	var parents []v1.DirectoryID

	// verify that ancestor indeed exists
	_, err := t.GetDirectory(ctx, ancestor, opts.ToGetOptions())
	if err != nil {
		return nil, err
	}

	for {
		dir, err := t.GetDirectory(ctx, child, opts.ToGetOptions())
		if err != nil {
			return nil, err
		}

		if dir.Parent == nil && dir.Id != ancestor {
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
func (t *Driver) GetChildren(
	ctx context.Context,
	id v1.DirectoryID,
	opts *storage.ListOptions,
) ([]v1.DirectoryID, error) {
	var children []v1.DirectoryID

	var iterationErr error

	t.dirMap.Range(func(key, value interface{}) bool {
		dir, ok := value.(*v1.Directory)
		if !ok {
			iterationErr = fmt.Errorf("found directory that is not of type *v1.Directory")
			return false
		}

		if (dir.DeletedAt == nil || opts.IsWithDeleted()) && dir.Parent != nil && *dir.Parent == id {
			children = append(children, dir.Id)
		}

		return true
	})

	if iterationErr != nil {
		return nil, iterationErr
	}

	if len(children) == 0 {
		return children, nil
	}

	// append the children's children
	for _, child := range children {
		c, err := t.GetChildren(ctx, child, opts)
		if err != nil {
			return nil, err
		}

		children = append(children, c...)
	}

	return children, nil
}
