package notifier

import (
	"context"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
)

type Notifier interface {
	NotifyCreate(ctx context.Context, d *apiv1.Directory) error
	NotifyUpdate(ctx context.Context, d *apiv1.Directory) error
	NotifyDeleteSoft(ctx context.Context, d *apiv1.Directory) error
	NotifyDeleteHard(ctx context.Context, d *apiv1.Directory) error
}
