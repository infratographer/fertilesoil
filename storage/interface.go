package storage

import (
	"context"

	v1 "github.com/infratographer/fertilesoil/api/v1"
)

// Reader is the interface that allows doing basic read operations
// on the directory tree.
type Reader interface {
	GetDirectory(ctx context.Context, id v1.DirectoryID, opts *GetOptions) (*v1.Directory, error)
	GetParents(ctx context.Context, id v1.DirectoryID, opts *ListOptions) ([]v1.DirectoryID, error)
	GetParentsUntilAncestor(
		ctx context.Context,
		child, ancestor v1.DirectoryID,
		opts *ListOptions,
	) ([]v1.DirectoryID, error)
	GetChildren(ctx context.Context, id v1.DirectoryID, opts *ListOptions) ([]v1.DirectoryID, error)
}

// RootReader is the interface that allows doing all read operations
// on the directory tree.
type RootReader interface {
	Reader
	ListRoots(ctx context.Context, opts *ListOptions) ([]v1.DirectoryID, error)
}

// Writer is the interface that allows doing basic write operations.
type Writer interface {
	CreateDirectory(ctx context.Context, d *v1.Directory) (*v1.Directory, error)
	DeleteDirectory(ctx context.Context, id v1.DirectoryID) ([]*v1.Directory, error)
}

// RootWriter is the interface that allows doing all write operations.
type RootWriter interface {
	Writer
	CreateRoot(ctx context.Context, d *v1.Directory) (*v1.Directory, error)
}

// DirectoryAdmin is the interface that allows doing all operations
// on the directory tree.
type DirectoryAdmin interface {
	RootReader
	RootWriter
}
