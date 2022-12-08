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

	v1 "github.com/JAORMX/fertilesoil/api/v1"
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

	if cfg.unixSocket != "" {
		cfg.client.Transport = &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", cfg.unixSocket)
			},
		}
	}

	return &httpClient{
		c:          cfg.client,
		managerURL: cfg.managerURL,
	}
}

type httpClient struct {
	c          *http.Client
	managerURL *url.URL
}

func (c *httpClient) CreateDirectory(ctx context.Context, cdr *v1.CreateDirectoryRequest, parent v1.DirectoryID) (*v1.DirectoryFetch, error) {
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

func (c *httpClient) CreateRoot(ctx context.Context, cdr *v1.CreateDirectoryRequest) (*v1.DirectoryFetch, error) {
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

func (c *httpClient) ListRoots(ctx context.Context) (*v1.DirectoryList, error) {
	resp, err := c.DoRaw(ctx, http.MethodGet, "/api/v1/roots", nil)
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

func (c *httpClient) GetDirectory(ctx context.Context, id v1.DirectoryID) (*v1.DirectoryFetch, error) {
	path, err := url.JoinPath("/api/v1/directories", id.String())
	if err != nil {
		return nil, fmt.Errorf("error getting directory: %w", err)
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

func (c *httpClient) DoRaw(ctx context.Context, method string, path string, data io.Reader) (*http.Response, error) {
	u := c.managerURL.JoinPath(path)
	req, err := http.NewRequestWithContext(ctx, method, u.String(), data)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	return c.c.Do(req)
}

func (c *httpClient) encode(r any) (io.Reader, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	err := enc.Encode(r)
	if err != nil {
		return nil, fmt.Errorf("error encoding request: %w", err)
	}
	return &buf, nil
}
