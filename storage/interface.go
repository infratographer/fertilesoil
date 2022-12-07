package storage

import (
	"context"

	v1 "github.com/JAORMX/fertilesoil/api/v1"
)

type Storage interface {
	CreateRoot(ctx context.Context, d *v1.Directory) (*v1.Directory, error)
	ListRoots(ctx context.Context) ([]v1.DirectoryID, error)
	CreateDirectory(ctx context.Context, d *v1.Directory) (*v1.Directory, error)
	GetDirectory(ctx context.Context, id v1.DirectoryID) (*v1.Directory, error)
	GetParents(ctx context.Context, id v1.DirectoryID) ([]v1.DirectoryID, error)
	GetChildren(ctx context.Context, id v1.DirectoryID) ([]v1.DirectoryID, error)
}
