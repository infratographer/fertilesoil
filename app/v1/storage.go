package v1

import (
	"context"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/storage"
)

// AppStorage is a commodity interface for applications of the tree
// to be able to track directories.
type AppStorage interface {
	storage.Writer
	IsDirectoryTracked(ctx context.Context, id apiv1.DirectoryID) (bool, error)
	IsDirectoryInfoUpdated(ctx context.Context, dir *apiv1.Directory) (bool, error)
}
