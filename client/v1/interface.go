package v1

import (
	"context"
	"io"
	"net/http"

	v1 "github.com/infratographer/fertilesoil/api/v1"
)

// ReadOnlyClient allows for instantiating a client
// with read-only access to the API.
type ReadOnlyClient interface {
	GetDirectory(c context.Context, id v1.DirectoryID) (*v1.DirectoryFetch, error)
	GetParents(c context.Context, id v1.DirectoryID) (*v1.DirectoryList, error)
	GetParentsUntil(c context.Context, id, until v1.DirectoryID) (*v1.DirectoryList, error)
	GetChildren(c context.Context, id v1.DirectoryID) (*v1.DirectoryList, error)
}

// Client Allows for instantiating a client
// with read/write access to the API.
type Client interface {
	ReadOnlyClient
	CreateDirectory(c context.Context, r *v1.CreateDirectoryRequest, parent v1.DirectoryID) (*v1.DirectoryFetch, error)
}

// RootClient allows for instantiating a client
// with read/write and root node access to the API.
type RootClient interface {
	Client // Embed the Client interface
	CreateRoot(c context.Context, r *v1.CreateDirectoryRequest) (*v1.DirectoryFetch, error)
	ListRoots(c context.Context) (*v1.DirectoryList, error)
}

// RawHTTP allows for instantiating a client
// with raw HTTP access to the API.
type RawHTTP interface {
	DoRaw(c context.Context, method, path string, data io.Reader) (*http.Response, error)
}

// HTTPClient allows for instantiating a client that can
// do general HTTP requests.
type HTTPClient interface {
	Client
	RawHTTP
}

type HTTPRootClient interface {
	RootClient
	RawHTTP
}

func NewHTTPClient(cfg *ClientConfig) HTTPClient {
	return newFullHTTPClient(cfg)
}

func NewHTTPRootClient(cfg *ClientConfig) HTTPRootClient {
	return newFullHTTPClient(cfg)
}

type Watcher interface {
	// Watch starts watching for events.
	// It returns a channel for events and a channel for errors.
	Watch(ctx context.Context) (<-chan *v1.DirectoryEvent, <-chan error)
}
