package v1

import (
	"context"
	"io"
	"net/http"

	v1 "github.com/infratographer/fertilesoil/api/v1"
)

type Client interface {
	CreateDirectory(c context.Context, r *v1.CreateDirectoryRequest, parent v1.DirectoryID) (*v1.DirectoryFetch, error)
	GetDirectory(c context.Context, id v1.DirectoryID) (*v1.DirectoryFetch, error)
	GetParents(c context.Context, id v1.DirectoryID) (*v1.DirectoryList, error)
	GetParentsUntil(c context.Context, id, until v1.DirectoryID) (*v1.DirectoryList, error)
	GetChildren(c context.Context, id v1.DirectoryID) (*v1.DirectoryList, error)
}

type RootClient interface {
	Client // Embed the Client interface
	CreateRoot(c context.Context, r *v1.CreateDirectoryRequest) (*v1.DirectoryFetch, error)
	ListRoots(c context.Context) (*v1.DirectoryList, error)
}

type RawHTTP interface {
	DoRaw(c context.Context, method, path string, data io.Reader) (*http.Response, error)
}

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
