package treemanager

import (
	"time"

	"github.com/infratographer/fertilesoil/notifier/noop"
)

const (
	// DefaultTreeManagerListen is the default listen address for the TreeManager.
	DefaultTreeManagerListen = ":8080"
	// DefaultTreeManagerUnix is the default unix socket for the TreeManager.
	DefaultTreeManagerUnix = ""
	// DefaultTreeManagerDebug is the default debug flag for the TreeManager.
	DefaultTreeManagerDebug = false
	// DefaultTreeManagerReadOnly is the default read-only flag for the TreeManager.
	DefaultTreeManagerReadOnly = false
	// DefaultTreeManagerFastReads is the default fast reads flag for the TreeManager.
	DefaultTreeManagerFastReads = false
	// DefaultTreeManagerShutdownTimeout is the default shutdown timeout for the TreeManager.
	DefaultTreeManagerShutdownTimeout = 5 * time.Second
)

// DefaultTreeManagerNotifier is the default notifier for the TreeManager.
var DefaultTreeManagerNotifier = noop.NewNotifier()
