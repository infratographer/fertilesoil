package utils

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/assert"

	clientv1 "github.com/infratographer/fertilesoil/client/v1"
	"github.com/infratographer/fertilesoil/internal/httpsrv/common"
)

// NewUnixsocketPath returns a new unix socket path for testing.
func NewUnixsocketPath(t *testing.T) string {
	t.Helper()
	tmpdir := t.TempDir()
	skt := filepath.Join(tmpdir, "skt")
	return skt
}

func RunTestServer(t *testing.T, srv *common.Server) {
	t.Helper()

	err := srv.Run(context.Background())
	assert.ErrorIs(t, err, http.ErrServerClosed, "unexpected error running server")
}

func NewTestClient(t *testing.T, skt string, srvURL *url.URL) clientv1.HTTPRootClient {
	t.Helper()

	cfg := clientv1.NewClientConfig().WithManagerURL(srvURL).WithUnixSocket(skt)
	httpc := clientv1.NewHTTPRootClient(cfg)
	return httpc
}

func WaitForServer(t *testing.T, cli clientv1.HTTPClient) {
	t.Helper()

	const (
		maxRetries      = 10
		backoffDuration = 5 * time.Millisecond
	)

	err := backoff.Retry(func() error {
		readyz, err := cli.DoRaw(context.Background(), http.MethodGet, "/readyz", nil)
		if err != nil {
			return err
		}
		defer readyz.Body.Close()
		if readyz.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", readyz.StatusCode)
		}
		return nil
	}, backoff.WithMaxRetries(backoff.NewConstantBackOff(backoffDuration), maxRetries))
	assert.NoError(t, err, "error waiting for server to be ready")
}
