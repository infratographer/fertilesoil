package treemanager_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
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
	t.Parallel()

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
	t.Parallel()

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

	// Ensure root parent is empty
	assert.Nil(t, rd.Directory.Parent, "unexpected parent")

	// Ensure root is returned in list
	rl, err := cli.ListRoots(context.Background())
	assert.NoError(t, err, "error getting roots")
	assert.Equal(t, 1, len(rl.Directories), "unexpected number of directories")

	// Ensure GetParents call returns empty list
	parents, err := cli.GetParents(context.Background(), rd.Directory.ID)
	assert.NoError(t, err, "error getting parents")
	assert.Equal(t, 0, len(parents.Directories), "unexpected number of parents")
}

func TestListMultipleRoots(t *testing.T) {
	t.Parallel()

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

	// Ensure ListRoots returns all roots
	rl, err := cli.ListRoots(context.Background())
	assert.NoError(t, err, "error getting roots")
	assert.Equal(t, nroots, len(rl.Directories), "unexpected number of directories")
}

func TestOneDirectory(t *testing.T) {
	t.Parallel()

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
	// Ensure parent info is set
	assert.NotNil(t, dir.Directory.Parent, "expected parent")
	assert.Equal(t, rd.Directory.ID, *dir.Directory.Parent, "unexpected parent")

	retd, err := cli.GetDirectory(context.Background(), dir.Directory.ID)
	assert.NoError(t, err, "error getting directory")
	assert.Equal(t, "dir", retd.Directory.Name, "unexpected directory name")
	// Ensure parent info is set
	assert.NotNil(t, dir.Directory.Parent, "expected parent")
	assert.Equal(t, rd.Directory.ID, *retd.Directory.Parent, "unexpected parent")

	parents, err := cli.GetParents(context.Background(), dir.Directory.ID)
	assert.NoError(t, err, "error getting parents")
	assert.Equal(t, 1, len(parents.Directories), "unexpected number of parents")
	assert.Equal(t, rd.Directory.ID, parents.Directories[0], "unexpected parent name")
}

/*
Tree structure to test:

	root
	dir1 dir2
			dir3
		dir4	dir5
*/
func TestFullTree(t *testing.T) {
	t.Parallel()

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

	// dir1
	_, err = cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "dir",
	},
		rd.Directory.ID,
	)

	assert.NoError(t, err, "error creating directory")

	dir2, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "dir",
	},
		rd.Directory.ID,
	)

	assert.NoError(t, err, "error creating directory")

	dir3, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "dir",
	},
		dir2.Directory.ID,
	)

	assert.NoError(t, err, "error creating directory")

	dir4, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "dir",
	},
		dir3.Directory.ID,
	)

	assert.NoError(t, err, "error creating directory")

	dir5, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "dir",
	},
		dir3.Directory.ID,
	)

	assert.NoError(t, err, "error creating directory")

	dir5parents, err := cli.GetParents(context.Background(), dir5.Directory.ID)
	assert.NoError(t, err, "error getting parents")
	assert.Equal(t, 3, len(dir5parents.Directories), "unexpected number of parents")
	assert.Equal(t, dir3.Directory.ID, dir5parents.Directories[0], "unexpected parent name")
	assert.Equal(t, dir2.Directory.ID, dir5parents.Directories[1], "unexpected parent name")
	assert.Equal(t, rd.Directory.ID, dir5parents.Directories[2], "unexpected parent name")

	dir4parents, err := cli.GetParents(context.Background(), dir4.Directory.ID)
	assert.NoError(t, err, "error getting parents")
	// same number of parents as dir5
	assert.Equal(t, 3, len(dir4parents.Directories), "unexpected number of parents")
	assert.Equal(t, dir3.Directory.ID, dir5parents.Directories[0], "unexpected parent name")
	assert.Equal(t, dir2.Directory.ID, dir5parents.Directories[1], "unexpected parent name")
	assert.Equal(t, rd.Directory.ID, dir5parents.Directories[2], "unexpected parent name")

	dir2children, err := cli.GetChildren(context.Background(), dir2.Directory.ID)
	assert.NoError(t, err, "error getting children")
	assert.Equal(t, 3, len(dir2children.Directories), "unexpected number of children")
	assert.Contains(t, dir2children.Directories, dir3.Directory.ID, "dir2 did not contain dir3 as child")
	assert.Contains(t, dir2children.Directories, dir4.Directory.ID, "dir2 did not contain dir4 as child")
	assert.Contains(t, dir2children.Directories, dir5.Directory.ID, "dir2 did not contain dir5 as child")

	parentsUntil, err := cli.GetParentsUntil(context.Background(), dir5.Directory.ID, dir2.Directory.ID)
	assert.NoError(t, err, "error getting parents")
	assert.Equal(t, 2, len(parentsUntil.Directories), "unexpected number of parents")
	assert.Equal(t, dir3.Directory.ID, parentsUntil.Directories[0], "unexpected parent name")
	assert.Equal(t, dir2.Directory.ID, parentsUntil.Directories[1], "unexpected parent name")
}

func TestCreateRootWithMalformedData(t *testing.T) {
	t.Parallel()

	skt := newUnixsocketPath(t)
	srv := newTestServer(t, skt)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go runTestServer(t, srv)

	cli := newTestClient(t, skt)

	waitForServer(t, cli)

	c := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", skt)
			},
		},
	}

	u := baseServerAddress.JoinPath("/api/v1/roots")

	// Invalid request with valid JSON
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	err := enc.Encode(map[string]string{"foo": "bar"})
	assert.NoError(t, err, "error encoding data")

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, u.String(), &buf)
	assert.NoError(t, err, "error creating request")

	resp, err := c.Do(req)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// invalid request with invalid JSON
	buf.Reset()
	buf.WriteString("{\"foo")

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, u.String(), &buf)
	assert.NoError(t, err, "error creating request")

	resp, err = c.Do(req)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()
}

func TestServerWithBadDB(t *testing.T) {
	t.Parallel()

	skt := newUnixsocketPath(t)

	tl, err := zap.NewDevelopment()
	assert.NoError(t, err, "error creating logger")

	// We're opening a valid database connection, but there's not database set.
	dbconn, err := sql.Open("postgres", baseDBURL.String())
	assert.NoError(t, err, "error creating db connection")

	srv := treemanager.NewServer(tl, srvhost, dbconn, debug, defaultShutdownTime, skt)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go runTestServer(t, srv)

	cli := newTestClient(t, skt)

	t.Log("waiting for server to start. This uses a timer as the database is not set up.")
	time.Sleep(1 * time.Second)

	_, err = cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "root",
	})
	assert.Error(t, err, "expected error creating root")

	_, err = cli.ListRoots(context.Background())
	assert.Error(t, err, "expected error getting roots")

	_, err = cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "dir",
	}, apiv1.DirectoryID(uuid.New()))
	assert.Error(t, err, "expected error creating directory")

	_, err = cli.GetDirectory(context.Background(), apiv1.DirectoryID(uuid.New()))
	assert.Error(t, err, "expected error getting directory")

	_, err = cli.GetChildren(context.Background(), apiv1.DirectoryID(uuid.New()))
	assert.Error(t, err, "expected error getting children")

	_, err = cli.GetParents(context.Background(), apiv1.DirectoryID(uuid.New()))
	assert.Error(t, err, "expected error getting parents")

	_, err = cli.GetParentsUntil(context.Background(),
		apiv1.DirectoryID(uuid.New()), apiv1.DirectoryID(uuid.New()))
	assert.Error(t, err, "expected error getting parents until")
}

func TestInvalidDirectoryIDs(t *testing.T) {
	t.Parallel()

	skt := newUnixsocketPath(t)
	srv := newTestServer(t, skt)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go runTestServer(t, srv)

	cli := newTestClient(t, skt)

	waitForServer(t, cli)

	// Create a root directory
	root, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "root",
	})
	assert.NoError(t, err, "error creating root directory")

	// Create a child directory
	child, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "child",
	}, root.Directory.ID)
	assert.NoError(t, err, "error creating child directory")

	// some string
	resp, err := cli.DoRaw(context.Background(), http.MethodGet, "/api/v1/directories/invalid", nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// some string getting children
	resp, err = cli.DoRaw(context.Background(), http.MethodGet, "/api/v1/directories/invalid/children", nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// some string getting parents
	resp, err = cli.DoRaw(context.Background(), http.MethodGet, "/api/v1/directories/invalid/parents", nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// some string getting parents until
	resp, err = cli.DoRaw(context.Background(), http.MethodGet,
		"/api/v1/directories/invalid/parents/00000000-0000-0000-0000-000000000000", nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// a valid child with an invalid parent
	resp, err = cli.DoRaw(context.Background(), http.MethodGet,
		fmt.Sprintf("/api/v1/directories/%s/parents/invalid", child.Directory.ID),
		nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// almost valid UUID
	resp, err = cli.DoRaw(context.Background(), http.MethodGet,
		"/api/v1/directories/00000000-0000-0000-0000-00000000XXX/children", nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// crazy long string
	resp, err = cli.DoRaw(context.Background(), http.MethodGet,
		fmt.Sprintf("/api/v1/directories/%s/children", strings.Repeat("a", 1000)), nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// SQL injection through DirectoryID
	resp, err = cli.DoRaw(context.Background(), http.MethodGet,
		"/api/v1/directories/00000000-0000-0000-0000-000000000000; DROP TABLE directories", nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// SQL injection through DirectoryID (using valid root ID)
	resp, err = cli.DoRaw(context.Background(), http.MethodGet,
		fmt.Sprintf("/api/v1/directories/%s; DROP TABLE directories", root.Directory.ID),
		nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()
}

func TestCreateErroneousDirectory(t *testing.T) {
	t.Parallel()

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

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	err = enc.Encode(&apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "child",
	})
	assert.NoError(t, err, "error encoding data")

	// Creating directory with invalid parent
	resp, err := cli.DoRaw(context.Background(), http.MethodPost,
		"/api/v1/directories/invalid", &buf)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// creating directory with null request
	resp, err = cli.DoRaw(context.Background(), http.MethodPost,
		fmt.Sprintf("/api/v1/directories/%s", rd.Directory.ID), nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// creating directory with invalid request
	resp, err = cli.DoRaw(context.Background(), http.MethodPost,
		fmt.Sprintf("/api/v1/directories/%s", rd.Directory.ID), bytes.NewBufferString("invalid"))
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// creating directory with invalid request (no name)
	buf.Reset()
	err = enc.Encode(&apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
	})
	assert.NoError(t, err, "error encoding data")
	resp, err = cli.DoRaw(context.Background(), http.MethodPost,
		fmt.Sprintf("/api/v1/directories/%s", rd.Directory.ID), &buf)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// creating directory with parent that doesn't exist
	nodir, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "nodir",
	}, apiv1.DirectoryID(uuid.New()))
	assert.Error(t, err, "should have errored creating directory")
	assert.Nil(t, nodir, "directory should be nil")

}
