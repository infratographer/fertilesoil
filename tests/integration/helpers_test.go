package integration_test

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
	"go.uber.org/zap"

	clientv1 "github.com/infratographer/fertilesoil/client/v1"
	"github.com/infratographer/fertilesoil/internal/httpsrv/common"
	"github.com/infratographer/fertilesoil/internal/httpsrv/treemanager"
	dbutils "github.com/infratographer/fertilesoil/storage/crdb/utils"
)

const (
	srvhost             = "localhost"
	debug               = true
	defaultShutdownTime = 1 * time.Second
)

var (
	baseDBURL         *url.URL
	baseServerAddress = mustParseURL("http://" + srvhost)
)

func mustParseURL(u string) *url.URL {
	parsed, err := url.Parse(u)
	if err != nil {
		panic(err)
	}
	return parsed
}

func TestMain(m *testing.M) {
	var stop func()
	baseDBURL, stop = dbutils.NewTestDBServerOrDie()
	defer stop()

	m.Run()
}

func newUnixsocketPath(t *testing.T) string {
	t.Helper()
	tmpdir := t.TempDir()
	skt := filepath.Join(tmpdir, "skt")
	return skt
}

func newTestServer(t *testing.T, skt string) *common.Server {
	t.Helper()

	tl, err := zap.NewDevelopment()
	assert.NoError(t, err, "error creating logger")

	dbconn := dbutils.GetNewTestDB(t, baseDBURL)
	tm := treemanager.NewServer(tl, srvhost, dbconn, debug, defaultShutdownTime, skt, nil)

	return tm
}

func newTestClient(t *testing.T, skt string) clientv1.HTTPRootClient {
	t.Helper()

	cfg := clientv1.NewClientConfig().WithManagerURL(baseServerAddress).WithUnixSocket(skt)
	httpc := clientv1.NewHTTPRootClient(cfg)
	return httpc
}

func runTestServer(t *testing.T, srv *common.Server) {
	t.Helper()

	err := srv.Run(context.Background())
	assert.ErrorIs(t, err, http.ErrServerClosed, "unexpected error running server")
}

func waitForServer(t *testing.T, cli clientv1.HTTPClient) {
	t.Helper()

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
	}, backoff.WithMaxRetries(backoff.NewConstantBackOff(5*time.Millisecond), 10))
	assert.NoError(t, err, "error waiting for server to be ready")
}
