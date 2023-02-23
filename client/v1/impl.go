package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	v1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/storage"
)

func newFullHTTPClient(cfg *ClientConfig) HTTPRootClient {
	if cfg == nil {
		cfg = &ClientConfig{}
	}

	if cfg.managerURL == nil {
		cfg.managerURL = &url.URL{
			Scheme: "http",
			Host:   "localhost:8080",
		}
	}

	if cfg.client == nil {
		cfg.client = &http.Client{
			Transport: http.DefaultTransport,
		}
	}

	return &httpClient{
		c:          cfg.client,
		managerURL: cfg.managerURL,
	}
}

func UnixClient(socket string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socket)
			},
		},
	}
}

type httpClient struct {
	c          *http.Client
	managerURL *url.URL
}

func (c *httpClient) CreateDirectory(
	ctx context.Context,
	cdr *v1.CreateDirectoryRequest,
	parent v1.DirectoryID,
) (*v1.DirectoryFetch, error) {
	r, err := c.encode(cdr)
	if err != nil {
		return nil, err
	}

	path, err := url.JoinPath("/api/v1/directories", parent.String())
	if err != nil {
		return nil, fmt.Errorf("error creating directory: %w", err)
	}

	resp, err := c.DoRaw(ctx, http.MethodPost, path, r)
	if err != nil {
		return nil, fmt.Errorf("error creating directory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("error creating directory: %s", resp.Status)
	}

	var dir v1.DirectoryFetch
	err = dir.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &dir, nil
}

func (c *httpClient) CreateRoot(
	ctx context.Context,
	cdr *v1.CreateDirectoryRequest,
) (*v1.DirectoryFetch, error) {
	r, err := c.encode(cdr)
	if err != nil {
		return nil, err
	}

	resp, err := c.DoRaw(ctx, http.MethodPost, "/api/v1/roots", r)
	if err != nil {
		return nil, fmt.Errorf("error creating root: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("error creating root: %s", resp.Status)
	}

	var dir v1.DirectoryFetch
	err = dir.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &dir, nil
}

func (c *httpClient) ListRoots(ctx context.Context, opts *storage.ListOptions) (*v1.DirectoryList, error) {
	path, err := addListOptionsToURL("/api/v1/roots", opts)
	if err != nil {
		return nil, fmt.Errorf("error adding options to url: %w", err)
	}

	resp, err := c.DoRaw(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error listing roots: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error listing roots: %s", resp.Status)
	}

	var dirList v1.DirectoryList
	err = dirList.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &dirList, nil
}

func (c *httpClient) DeleteDirectory(ctx context.Context, id v1.DirectoryID) (*v1.DirectoryList, error) {
	path, err := url.JoinPath("/api/v1/directories", id.String())
	if err != nil {
		return nil, fmt.Errorf("error getting directory: %w", err)
	}

	resp, err := c.DoRaw(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting directory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting directory: %s", resp.Status)
	}

	var dirList v1.DirectoryList
	err = dirList.Parse(resp.Body)

	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &dirList, nil
}

func (c *httpClient) GetDirectory(
	ctx context.Context,
	id v1.DirectoryID,
	opts *storage.GetOptions,
) (*v1.DirectoryFetch, error) {
	path, err := url.JoinPath("/api/v1/directories", id.String())
	if err != nil {
		return nil, fmt.Errorf("error getting directory: %w", err)
	}

	path, err = addGetOptionsToURL(path, opts)
	if err != nil {
		return nil, fmt.Errorf("error adding options to url: %w", err)
	}

	resp, err := c.DoRaw(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting directory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting directory: %s", resp.Status)
	}

	var dir v1.DirectoryFetch
	err = dir.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &dir, nil
}

func (c *httpClient) GetParents(
	ctx context.Context,
	id v1.DirectoryID,
	opts *storage.ListOptions,
) (*v1.DirectoryList, error) {
	return c.doGetParents(ctx, id, nil, opts)
}

func (c *httpClient) GetParentsUntil(
	ctx context.Context,
	id v1.DirectoryID,
	until v1.DirectoryID,
	opts *storage.ListOptions,
) (*v1.DirectoryList, error) {
	return c.doGetParents(ctx, id, &until, opts)
}

func (c *httpClient) doGetParents(
	ctx context.Context,
	id v1.DirectoryID,
	until *v1.DirectoryID,
	opts *storage.ListOptions,
) (*v1.DirectoryList, error) {
	path, err := url.JoinPath("/api/v1/directories", id.String(), "parents")
	if err != nil {
		return nil, fmt.Errorf("error getting parents: %w", err)
	}

	path, err = addListOptionsToURL(path, opts)
	if err != nil {
		return nil, fmt.Errorf("error adding options to url: %w", err)
	}

	if until != nil {
		path, err = url.JoinPath(path, until.String())
		if err != nil {
			return nil, fmt.Errorf("error getting parents: %w", err)
		}
	}

	resp, err := c.DoRaw(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting parents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting parents: %s", resp.Status)
	}

	var dirList v1.DirectoryList
	err = dirList.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &dirList, nil
}

func (c *httpClient) GetChildren(
	ctx context.Context,
	id v1.DirectoryID,
	opts *storage.ListOptions,
) (*v1.DirectoryList, error) {
	path, err := url.JoinPath("/api/v1/directories", id.String(), "children")
	if err != nil {
		return nil, fmt.Errorf("error getting children: %w", err)
	}

	path, err = addListOptionsToURL(path, opts)
	if err != nil {
		return nil, fmt.Errorf("error adding options to url: %w", err)
	}

	resp, err := c.DoRaw(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting children: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting children: %s", resp.Status)
	}

	var dirList v1.DirectoryList
	err = dirList.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &dirList, nil
}

func (c *httpClient) DoRaw(
	ctx context.Context,
	method string,
	path string,
	data io.Reader,
) (*http.Response, error) {
	uPath, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("error handling path: %w", err)
	}

	u := c.managerURL.JoinPath(uPath.Path)

	// Merge any query values managerURL and path may have.
	values := u.Query()

	for k, v := range uPath.Query() {
		values[k] = v
	}

	u.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, method, u.String(), data)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	return c.c.Do(req)
}

func (c *httpClient) encode(r any) (io.Reader, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	if err := enc.Encode(r); err != nil {
		return nil, fmt.Errorf("error encoding request: %w", err)
	}
	return &buf, nil
}

// addGetOptionsToURL adds defined options to the provided url string.
func addGetOptionsToURL(urlstr string, opts *storage.GetOptions) (string, error) {
	u, err := url.Parse(urlstr)
	if err != nil {
		return "", err
	}

	values := u.Query()

	if opts.IsWithDeleted() {
		values.Set("with_deleted", "true")
	}

	u.RawQuery = values.Encode()

	return u.String(), nil
}

// addListOptionsToURL adds defined options to the provided url string.
func addListOptionsToURL(urlstr string, opts *storage.ListOptions) (string, error) {
	u, err := url.Parse(urlstr)
	if err != nil {
		return "", err
	}

	values := u.Query()

	if opts.IsWithDeleted() {
		values.Set("with_deleted", "true")
	}

	u.RawQuery = values.Encode()

	return u.String(), nil
}
