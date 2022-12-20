package v1

import (
	"context"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
)

// Reconciler is the interface that allows the controller to reconcile
// the directory tree with the application.
type Reconciler interface {
	Reconcile(context.Context, apiv1.DirectoryEvent) error
}

// Controller is the main controller for a fertilesoil application.
// It is responsible for watching for changes in the directory tree and
// reconciling the changes with the application.
// It also periodically checks for updates to the directory tree.
// The controller is responsible for persisting the directory tree in the
// provided storage.
// Note that while you could run multiple controllers for the same application,
// you need to make sure that any changes the application reacts to are
// idempotent. Else, you might end up with multiple instances of the same
// resource. Instead, you should run a single controller and replicas on
// stand-by by using leader election.
type Controller interface {
	// Run starts the controller.
	Run(context.Context) error
}

// ControllerBuilder is a function that builds a controller.
type ControllerBuilder func(baseDir apiv1.DirectoryID, opts ...Option) (Controller, error)

// NewController is the default implementation of ControllerBuilder.
var NewController ControllerBuilder = newController

// Option is a function that configures the controller.
type Option func(*controller)

// WithReconciler is an Option that sets the reconciler for the controller.
// Make sure that the reconciler is idempotent.
var WithReconciler = withReconciler

// WithClient is an Option that sets the client for the controller.
var WithClient = withClient

// WithStorage is an Option that sets the storage for the tracked directories of
// the application.
var WithStorage = withStorage

// WithWatcher is an Option that sets the watcher for the controller.
var WithWatcher = withWatcher
