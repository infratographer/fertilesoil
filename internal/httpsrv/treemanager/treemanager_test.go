package treemanager_test

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

	apiv1 "github.com/JAORMX/fertilesoil/api/v1"
	clientv1 "github.com/JAORMX/fertilesoil/client/v1"
	"github.com/JAORMX/fertilesoil/internal/httpsrv/common"
	"github.com/JAORMX/fertilesoil/internal/httpsrv/treemanager"
	dbutils "github.com/JAORMX/fertilesoil/storage/crdb/utils"
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
	tm := treemanager.NewServer(tl, srvhost, dbconn, debug, defaultShutdownTime, skt)

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

func TestListNoRoots(t *testing.T) {
	skt := newUnixsocketPath(t)
	srv := newTestServer(t, skt)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go runTestServer(t, srv)

	cli := newTestClient(t, skt)

	waitForServer(t, cli)

	rl, err := cli.ListRoots(context.Background())
	assert.NoError(t, err, "error getting roots")
	assert.Equal(t, 0, len(rl.Directories), "unexpected number of directories")
}

func TestListOneRoot(t *testing.T) {
	skt := newUnixsocketPath(t)
	srv := newTestServer(t, skt)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go runTestServer(t, srv)

	cli := newTestClient(t, skt)

	waitForServer(t, cli)

	_, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "root",
	})

	assert.NoError(t, err, "error creating root")

	rl, err := cli.ListRoots(context.Background())
	assert.NoError(t, err, "error getting roots")
	assert.Equal(t, 1, len(rl.Directories), "unexpected number of directories")
}

func TestListMultipleRoots(t *testing.T) {
	skt := newUnixsocketPath(t)
	srv := newTestServer(t, skt)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go runTestServer(t, srv)

	cli := newTestClient(t, skt)

	waitForServer(t, cli)

	nroots := 10
	for idx := 0; idx < nroots; idx++ {
		_, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
			DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
				Version: apiv1.APIVersion,
			},
			Name: fmt.Sprintf("root%d", idx),
		})
		assert.NoError(t, err, "error creating root")
	}

	rl, err := cli.ListRoots(context.Background())
	assert.NoError(t, err, "error getting roots")
	assert.Equal(t, nroots, len(rl.Directories), "unexpected number of directories")
}

func TestOneDirectory(t *testing.T) {
	skt := newUnixsocketPath(t)
	srv := newTestServer(t, skt)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go runTestServer(t, srv)

	cli := newTestClient(t, skt)

	waitForServer(t, cli)

	rd, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "root",
	})
	assert.NoError(t, err, "error creating root")

	dir, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "dir",
	},
		rd.Directory.ID,
	)

	assert.NoError(t, err, "error creating directory")
	assert.Equal(t, "dir", dir.Directory.Name, "unexpected directory name")

	retd, err := cli.GetDirectory(context.Background(), dir.Directory.ID)
	assert.NoError(t, err, "error getting directory")
	assert.Equal(t, "dir", retd.Directory.Name, "unexpected directory name")
}
