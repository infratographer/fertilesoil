package callback

import (
	"context"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	appv1 "github.com/infratographer/fertilesoil/app/v1"
)

// CallbackConfig allows for configuring callbacks for when a directory
// is created or deleted. This allows applications to react to these
// events.
type Config struct {
	CreateDirectory func(context.Context, *apiv1.Directory) error
	DeleteDirectory func(context.Context, apiv1.DirectoryID) error
}

// AppStorageWithCallback is an implementation of AppStorage that
// allows for callbacks to be configured for when a directory is
// created or deleted.
type AppStorageWithCallback struct {
	impl appv1.AppStorage
	cfg  Config
}

func NewAppStorageWithCallback(impl appv1.AppStorage, cfg Config) *AppStorageWithCallback {
	return &AppStorageWithCallback{
		impl: impl,
		cfg:  cfg,
	}
}

var _ appv1.AppStorage = (*AppStorageWithCallback)(nil)

func (s *AppStorageWithCallback) CreateDirectory(ctx context.Context, d *apiv1.Directory) (*apiv1.Directory, error) {
	d, err := s.impl.CreateDirectory(ctx, d)
	if err != nil {
		return nil, err
	}
	if err := s.cfg.CreateDirectory(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *AppStorageWithCallback) DeleteDirectory(ctx context.Context, id apiv1.DirectoryID) error {
	if err := s.cfg.DeleteDirectory(ctx, id); err != nil {
		return err
	}
	return s.impl.DeleteDirectory(ctx, id)
}

func (s *AppStorageWithCallback) IsDirectoryTracked(ctx context.Context, id apiv1.DirectoryID) (bool, error) {
	return s.impl.IsDirectoryTracked(ctx, id)
}

func (s *AppStorageWithCallback) IsDirectoryInfoUpdated(ctx context.Context, dir *apiv1.Directory) (bool, error) {
	return s.impl.IsDirectoryInfoUpdated(ctx, dir)
}
