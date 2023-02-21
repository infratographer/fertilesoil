package v1

import (
	"context"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
)

// NoopReconciler is a Reconciler that does nothing.
// This is useful in cases where we just want to persist the directory tree
// without doing any reconciliation.
type NoopReconciler struct{}

//nolint:gocritic // we want to keep the signature of the Reconciler interface
func (r *NoopReconciler) Reconcile(ctx context.Context, de apiv1.DirectoryEvent) error {
	return nil
}
