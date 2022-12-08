package v1

import (
	"net/http"
	"net/url"
)

type ClientConfig struct {
	managerURL *url.URL
	client     *http.Client
	unixSocket string
}

func NewClientConfig() *ClientConfig {
	return &ClientConfig{}
}

func (c *ClientConfig) WithManagerURL(u *url.URL) *ClientConfig {
	c.managerURL = u
	return c
}

func (c *ClientConfig) WithClient(client *http.Client) *ClientConfig {
	c.client = client
	return c
}

func (c *ClientConfig) WithUnixSocket(socket string) *ClientConfig {
	c.unixSocket = socket
	return c
}

func (c *ClientConfig) WithManagerURLFromString(s string) (*ClientConfig, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	c.managerURL = u
	return c, nil
}
