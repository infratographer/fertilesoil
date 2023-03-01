package treemanager

import (
	"net"
	"time"

	"github.com/metal-toolbox/auditevent/ginaudit"
	"go.hollow.sh/toolbox/ginjwt"

	"github.com/infratographer/fertilesoil/notifier"
	"github.com/infratographer/fertilesoil/notifier/noop"
	"github.com/infratographer/fertilesoil/storage"
)

type treeManagerConfig struct {
	listen          string
	unix            string
	listener        net.Listener
	debug           bool
	shutdownTimeout time.Duration
	notif           notifier.Notifier
	storageDriver   storage.DirectoryAdmin
	auditMdw        *ginaudit.Middleware
	authConfig      *ginjwt.AuthConfig
	trustedProxies  []string
}

type Option func(*treeManagerConfig)

// WithListen sets the listen address for the server.
func WithListen(listen string) Option {
	return func(c *treeManagerConfig) {
		c.listen = listen
	}
}

// WithListener sets the listener for the server.
func WithListener(listener net.Listener) Option {
	return func(c *treeManagerConfig) {
		c.listener = listener
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

func WithAuditMiddleware(auditMiddleware *ginaudit.Middleware) Option {
	return func(c *treeManagerConfig) {
		c.auditMdw = auditMiddleware
	}
}

// WithAuthConfig sets the jwt auth config.
func WithAuthConfig(config *ginjwt.AuthConfig) Option {
	return func(c *treeManagerConfig) {
		c.authConfig = config
	}
}

// WithTrustedProxies sets the trusted proxies for handling proxied requests properly.
func WithTrustedProxies(proxies []string) Option {
	return func(c *treeManagerConfig) {
		c.trustedProxies = proxies
	}
}

func (c *treeManagerConfig) apply(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}
