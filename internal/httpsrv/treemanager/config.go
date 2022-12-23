package treemanager

import (
	"time"

	"github.com/infratographer/fertilesoil/notifier"
	"github.com/infratographer/fertilesoil/notifier/noop"
	"github.com/infratographer/fertilesoil/storage/crdb/driver"
)

type treeManagerConfig struct {
	listen          string
	unix            string
	debug           bool
	readonly        bool
	fastReads       bool
	shutdownTimeout time.Duration
	notif           notifier.Notifier
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

// WithReadOnly sets the readonly flag for the server.
// If true, the server will use the readonly flag when
// reading from the database.
func WithReadOnly(readonly bool) Option {
	return func(c *treeManagerConfig) {
		c.readonly = readonly
	}
}

// WithFastReads sets the fastReads flag for the server.
// If true, the server will use the fastReads flag when
// reading from the database. This will cause the server
// to read from the database without waiting for the
// database to commit the transaction. This is useful
// for read-heavy workloads, but can cause data to be
// out of sync with the database.
func WithFastReads(fastReads bool) Option {
	return func(c *treeManagerConfig) {
		c.fastReads = fastReads
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

func (c *treeManagerConfig) apply(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}

func (c *treeManagerConfig) withStorageDriverOptions() []driver.Options {
	opts := []driver.Options{}
	if c.readonly {
		opts = append(opts, driver.WithReadOnly())
	}
	if c.fastReads {
		opts = append(opts, driver.WithFastReads())
	}
	return opts
}
