package treemanager

import (
	"time"

	"github.com/infratographer/fertilesoil/notifier"
	"github.com/infratographer/fertilesoil/notifier/noop"
	"github.com/infratographer/fertilesoil/storage"
)

type treeManagerConfig struct {
	listen          string
	unix            string
	debug           bool
	shutdownTimeout time.Duration
	notif           notifier.Notifier
	storageDriver   storage.DirectoryAdmin
}

type Option func(*treeManagerConfig)

// WithListen sets the listen address for the server.
func WithListen(listen string) Option {
	return func(c *treeManagerConfig) {
		c.listen = listen
	}
}

// WithUnix sets the unix socket for the server.
// If set, the server will listen on the unix socket
// instead of the listen address.
func WithUnix(unix string) Option {
	return func(c *treeManagerConfig) {
		c.unix = unix
	}
}

// WithDebug sets the debug flag for the server.
func WithDebug(debug bool) Option {
	return func(c *treeManagerConfig) {
		c.debug = debug
	}
}

// WithShutdownTimeout sets the shutdown timeout for the server.
func WithShutdownTimeout(t time.Duration) Option {
	return func(c *treeManagerConfig) {
		c.shutdownTimeout = t
	}
}

// WithNotifier sets the notifier for the server.
func WithNotifier(n notifier.Notifier) Option {
	return func(c *treeManagerConfig) {
		if n == nil {
			n = noop.NewNotifier()
		}
		c.notif = n
	}
}

// WithStorageDriver sets the storage driver for the server.
func WithStorageDriver(d storage.DirectoryAdmin) Option {
	return func(c *treeManagerConfig) {
		c.storageDriver = d
	}
}

func (c *treeManagerConfig) apply(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}
