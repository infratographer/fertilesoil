package v1

import "errors"

// ErrNoReconciler is returned when no reconciler is provided to the controller.
var ErrNoReconciler = errors.New("no reconciler provided")
