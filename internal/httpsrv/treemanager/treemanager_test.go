package treemanager_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/metal-toolbox/auditevent/ginaudit"
	"github.com/stretchr/testify/assert"
	"go.hollow.sh/toolbox/ginjwt"
	"go.uber.org/zap"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/internal/httpsrv/common"
	"github.com/infratographer/fertilesoil/internal/httpsrv/treemanager"
	"github.com/infratographer/fertilesoil/storage"
	"github.com/infratographer/fertilesoil/storage/memory"
	integration "github.com/infratographer/fertilesoil/tests/integration"
	testutils "github.com/infratographer/fertilesoil/tests/utils"
)

const (
	srvhost             = "localhost"
	debug               = true
	defaultShutdownTime = 1 * time.Second
)

func newTestServer(t *testing.T,
	skt string,
	store storage.DirectoryAdmin,
	authConfig *ginjwt.AuthConfig,
	w io.Writer,
) *common.Server {
	t.Helper()

	if store == nil {
		store, _ = newMemoryStorage(t)
	}

	tl, err := zap.NewDevelopment()
	assert.NoError(t, err, "error creating logger")

	mdw := ginaudit.NewJSONMiddleware("test", w)

	tm := treemanager.NewServer(
		tl,
		nil, // dbconn is empty.
		treemanager.WithListen(srvhost),
		treemanager.WithDebug(debug),
		treemanager.WithShutdownTimeout(defaultShutdownTime),
		treemanager.WithUnix(skt),
		// this sets up a correct notifier undearneath even if it's nil.
		treemanager.WithNotifier(nil),
		treemanager.WithStorageDriver(store),
		treemanager.WithAuditMiddleware(mdw),
		treemanager.WithAuthConfig(authConfig),
	)

	return tm
}

func newMemoryStorage(t *testing.T) (storage.DirectoryAdmin, *sync.Map) {
	t.Helper()

	dirMap := &sync.Map{}

	store := memory.NewDirectoryDriver(memory.WithDirectoryMap(dirMap))

	return store, dirMap
}

func getStubServerAddress(t *testing.T, skt string) *url.URL {
	t.Helper()

	u, err := url.Parse("http://" + srvhost + "?unix=" + skt)
	assert.NoError(t, err, "error parsing url")

	return u
}

func TestRootOperations(t *testing.T) {
	t.Parallel()

	auditBuf := &strings.Builder{}
	skt := testutils.NewUnixsocketPath(t)

	srv := newTestServer(t, skt, nil, nil, auditBuf)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, getStubServerAddress(t, skt), nil)

	testutils.WaitForServer(t, cli)

	// Create a new root.
	rd, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "root",
	})
	assert.NoError(t, err, "error creating root")
	assert.NotNil(t, rd, "root directory is nil")

	// Check that we have an audit log for this
	assert.Contains(t, auditBuf.String(), "POST:/api/v1/roots")
	auditBuf.Reset()

	// Get the root.
	listroots, err := cli.ListRoots(context.Background())
	assert.NoError(t, err, "error listing roots")

	// Check that we have an audit log for this
	assert.Contains(t, auditBuf.String(), "GET:/api/v1/roots")
	auditBuf.Reset()

	assert.Equal(t, 1, len(listroots.Directories), "expected 1 root, got %d", len(listroots.Directories))

	// Get the root with deleted.
	listroots, err = cli.ListRoots(context.Background(), storage.WithDeletedDirectories)
	assert.NoError(t, err, "error listing roots")

	assert.Equal(t, 1, len(listroots.Directories), "expected 1 root, got %d", len(listroots.Directories))
}

func TestDirectoryOperations(t *testing.T) {
	t.Parallel()

	auditBuf := &strings.Builder{}
	skt := testutils.NewUnixsocketPath(t)

	srv := newTestServer(t, skt, nil, nil, auditBuf)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, getStubServerAddress(t, skt), nil)

	testutils.WaitForServer(t, cli)

	// Create a new root.
	rd, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "root",
	})
	assert.NoError(t, err, "error creating root")
	assert.NotNil(t, rd, "root directory is nil")

	// Check that we have an audit log for this
	assert.Contains(t, auditBuf.String(), "POST:/api/v1/roots")
	auditBuf.Reset()

	// Create a new directory.
	d, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "test",
	}, rd.Directory.Id)
	assert.NoError(t, err, "error creating directory")
	assert.NotNil(t, d, "directory is nil")

	// Check that we have an audit log for this
	assert.Contains(t, auditBuf.String(), "POST:/api/v1/directories")
	auditBuf.Reset()

	// Get the directory.
	retd, err := cli.GetDirectory(context.Background(), d.Directory.Id)
	assert.NoError(t, err, "error getting directory")
	assert.NotNil(t, retd, "directory is nil")

	// Check that we have an audit log for this
	assert.Contains(t, auditBuf.String(), "GET:/api/v1/directories")
	auditBuf.Reset()

	// directory should be the same as the one we created.
	assert.Equal(t, d.Directory.Id, retd.Directory.Id, "directory is not the same")
	assert.Equal(t, d.Directory.Name, retd.Directory.Name, "directory is not the same")

	// List the directory.
	listd, err := cli.GetChildren(context.Background(), rd.Directory.Id)
	assert.NoError(t, err, "error listing directories")
	assert.NotNil(t, listd, "directory is nil")

	// There should be 1 directory.
	assert.Equal(t, 1, len(listd.Directories), "expected 1 directory, got %d", len(listd.Directories))

	// Get the root as parent.
	listrd, err := cli.GetParents(context.Background(), d.Directory.Id)
	assert.NoError(t, err, "error listing directories")
	assert.NotNil(t, listrd, "directory is nil")

	// There should be 1 directory.
	assert.Equal(t, 1, len(listrd.Directories), "expected 1 directory, got %d", len(listrd.Directories))

	// The directory should be the parent.
	assert.Equal(t, rd.Directory.Id, listrd.Directories[0], "directory is not the same")

	// Get the root as parent with GetParentsUntil function
	listrd, err = cli.GetParentsUntil(context.Background(), d.Directory.Id, rd.Directory.Id)
	assert.NoError(t, err, "error listing directories")
	assert.NotNil(t, listrd, "directory is nil")

	// There should be 1 directory.
	assert.Equal(t, 1, len(listrd.Directories), "expected 1 directory, got %d", len(listrd.Directories))

	// The directory should be the parent.
	assert.Equal(t, rd.Directory.Id, listrd.Directories[0], "directory is not the same")

	d2, err := cli.UpdateDirectory(context.Background(), d.Directory.Id, &apiv1.UpdateDirectoryRequest{
		Name: ptr("test2"),
		Metadata: &apiv1.DirectoryMetadata{
			"item1": "value1",
		},
	})
	assert.NoError(t, err, "expected no error updating directory")
	assert.Equal(t, "test2", d2.Directory.Name, "expected name to be updated")
	assert.Contains(t, map[string]string(*d2.Directory.Metadata), "item1", "expected metadata to be updated")

	d2, err = cli.UpdateDirectory(context.Background(), d.Directory.Id, &apiv1.UpdateDirectoryRequest{
		Name: nil,
		Metadata: &apiv1.DirectoryMetadata{
			"item2": "value2",
		},
	})
	assert.NoError(t, err, "expected no error updating directory")
	assert.Equal(t, "test2", d2.Directory.Name, "expected name to be not change")
	assert.Contains(t, map[string]string(*d2.Directory.Metadata), "item2", "expected metadata to be updated")

	d2, err = cli.UpdateDirectory(context.Background(), d.Directory.Id, &apiv1.UpdateDirectoryRequest{
		Name: ptr(""),
		Metadata: &apiv1.DirectoryMetadata{
			"item3": "value3",
		},
	})
	assert.NoError(t, err, "expected no error updating directory")
	assert.Equal(t, "test2", d2.Directory.Name, "expected name to be not change")
	assert.Contains(t, map[string]string(*d2.Directory.Metadata), "item3", "expected metadata to be updated")

	d2, err = cli.UpdateDirectory(context.Background(), d.Directory.Id, &apiv1.UpdateDirectoryRequest{
		Name: ptr("test3"),
	})
	assert.NoError(t, err, "expected no error updating directory")
	assert.Equal(t, "test3", d2.Directory.Name, "expected name to be updated")
	assert.Contains(t, map[string]string(*d2.Directory.Metadata), "item3", "expected metadata to not be updated")

	// Delete directory
	affected, err := cli.DeleteDirectory(context.Background(), d.Directory.Id)
	assert.NoError(t, err, "error deleting child directory")
	assert.Equal(t, 1, len(affected.Directories), "expected 1 deleted directory, got %d", len(affected.Directories))

	// Get deleted directory
	dd, err := cli.GetDirectory(context.Background(), d.Directory.Id, storage.WithDeletedDirectories)
	assert.NoError(t, err, "error retrieving deleted directory")
	assert.Equal(t, d.Directory.Id, dd.Directory.Id, "unexpected response directory")

	// Get directory deleted children
	dl, err := cli.GetChildren(context.Background(), rd.Directory.Id, storage.WithDeletedDirectories)
	assert.NoError(t, err, "error retrieving directory deleted children")
	assert.Equal(t, 1, len(dl.Directories), "unexpected children count")
	assert.Equal(t, d.Directory.Id, dl.Directories[0], "unexpected response child id")

	// Get the root as parent with GetParentsUntil function
	dl, err = cli.GetParentsUntil(
		context.Background(),
		d.Directory.Id,
		rd.Directory.Id,
		storage.WithDeletedDirectories,
	)
	assert.NoError(t, err, "error retrieving parent directories")
	assert.Equal(t, 1, len(dl.Directories), "unexpected parent count")
	assert.Contains(t, dl.Directories, rd.Directory.Id, "expected root directory in returned directories")

	// Test errors are returned for bad params
	resp, err := cli.DoRaw(context.Background(), http.MethodGet, "/api/v1/roots?with_deleted=bad", nil)
	assert.NoError(t, err, "no error expected for http request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "expected bad requests status code")
	resp.Body.Close()

	resp, err = cli.DoRaw(context.Background(), http.MethodGet, fmt.Sprintf(
		"/api/v1/directories/%s?with_deleted=bad",
		d.Directory.Id.String(),
	), nil)
	assert.NoError(t, err, "no error expected for http request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "expected bad requests status code")
	resp.Body.Close()

	resp, err = cli.DoRaw(context.Background(), http.MethodGet, fmt.Sprintf(
		"/api/v1/directories/%s/children?with_deleted=bad",
		d.Directory.Id.String(),
	), nil)
	assert.NoError(t, err, "no error expected for http request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "expected bad requests status code")
	resp.Body.Close()

	resp, err = cli.DoRaw(context.Background(), http.MethodGet, fmt.Sprintf(
		"/api/v1/directories/%s/parents?with_deleted=bad",
		d.Directory.Id.String(),
	), nil)
	assert.NoError(t, err, "no error expected for http request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "expected bad requests status code")
	resp.Body.Close()

	resp, err = cli.DoRaw(context.Background(), http.MethodGet, fmt.Sprintf(
		"/api/v1/directories/%s/parents/%s?with_deleted=bad",
		d.Directory.Id.String(),
		rd.Directory.Id.String(),
	), nil)
	assert.NoError(t, err, "no error expected for http request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "expected bad requests status code")
	resp.Body.Close()
}

type mockDriver struct {
	storage.DirectoryAdmin
	UpdateError error
}

// UpdateDirectory returns UpdateError if not nil.
func (m *mockDriver) UpdateDirectory(ctx context.Context, d *apiv1.Directory) error {
	if m.UpdateError != nil {
		return m.UpdateError
	}

	return m.DirectoryAdmin.UpdateDirectory(ctx, d)
}

func ptr[T any](v T) *T {
	return &v
}

func TestUpdateDirectoryError(t *testing.T) {
	t.Parallel()

	auditBuf := &strings.Builder{}
	skt := testutils.NewUnixsocketPath(t)

	memDriver, _ := newMemoryStorage(t)

	// no simple way to make the driver return an error, so force an error to fully test.
	updateErrorStore := &mockDriver{
		DirectoryAdmin: memDriver,
		UpdateError:    fmt.Errorf("update error"),
	}

	srv := newTestServer(t, skt, updateErrorStore, nil, auditBuf)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, getStubServerAddress(t, skt), nil)

	testutils.WaitForServer(t, cli)

	// Create a new root.
	rd, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "root",
	})
	assert.NoError(t, err, "error creating root")
	assert.NotNil(t, rd, "root directory is nil")

	// Check that we have an audit log for this
	assert.Contains(t, auditBuf.String(), "POST:/api/v1/roots")
	auditBuf.Reset()

	// Create a new directory.
	d, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "test",
	}, rd.Directory.Id)
	assert.NoError(t, err, "error creating directory")
	assert.NotNil(t, d, "directory is nil")

	_, err = cli.UpdateDirectory(context.Background(), d.Directory.Id, &apiv1.UpdateDirectoryRequest{
		Name: ptr("test2"),
		Metadata: &apiv1.DirectoryMetadata{
			"item1": "value1",
		},
	})
	assert.Error(t, err, "expected error updating directory")
}

func TestErroneousCalls(t *testing.T) {
	t.Parallel()

	auditBuf := &strings.Builder{}
	skt := testutils.NewUnixsocketPath(t)

	srv := newTestServer(t, skt, nil, nil, auditBuf)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	srvAddr := getStubServerAddress(t, skt)
	cli := testutils.NewTestClient(t, skt, srvAddr, nil)

	testutils.WaitForServer(t, cli)

	integration.MalformedDataTest(t, skt, srvAddr)
}

func TestInvalidDirectoryIDs(t *testing.T) {
	t.Parallel()

	auditBuf := &strings.Builder{}
	skt := testutils.NewUnixsocketPath(t)

	srv := newTestServer(t, skt, nil, nil, auditBuf)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	srvAddr := getStubServerAddress(t, skt)
	cli := testutils.NewTestClient(t, skt, srvAddr, nil)

	testutils.WaitForServer(t, cli)

	integration.InvalidDirectoryIDsTest(t, cli)
}

func TestErroneousDirectories(t *testing.T) {
	t.Parallel()

	auditBuf := &strings.Builder{}
	skt := testutils.NewUnixsocketPath(t)

	srv := newTestServer(t, skt, nil, nil, auditBuf)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	srvAddr := getStubServerAddress(t, skt)
	cli := testutils.NewTestClient(t, skt, srvAddr, nil)

	testutils.WaitForServer(t, cli)

	integration.ErroneousDirectoryTest(t, cli)
}

func TestDirectoryNotFound(t *testing.T) {
	t.Parallel()

	auditBuf := &strings.Builder{}
	skt := testutils.NewUnixsocketPath(t)

	srv := newTestServer(t, skt, nil, nil, auditBuf)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	srvAddr := getStubServerAddress(t, skt)
	cli := testutils.NewTestClient(t, skt, srvAddr, nil)

	testutils.WaitForServer(t, cli)

	integration.DirectoryNotFoundTest(t, cli)
}

func TestDeleteDirectory(t *testing.T) {
	t.Parallel()

	auditBuf := &strings.Builder{}
	skt := testutils.NewUnixsocketPath(t)

	srv := newTestServer(t, skt, nil, nil, auditBuf)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	srvAddr := getStubServerAddress(t, skt)
	cli := testutils.NewTestClient(t, skt, srvAddr, nil)

	testutils.WaitForServer(t, cli)

	integration.DeleteDirectoryTest(t, cli)
}

func TestErrorDoesntLeakInfo(t *testing.T) {
	t.Parallel()

	auditBuf := &strings.Builder{}
	skt := testutils.NewUnixsocketPath(t)

	store, dirMap := newMemoryStorage(t)

	srv := newTestServer(t, skt, store, nil, auditBuf)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	srvAddr := getStubServerAddress(t, skt)
	cli := testutils.NewTestClient(t, skt, srvAddr, nil)

	testutils.WaitForServer(t, cli)

	// Create erroneous instance of the root directory.
	id := apiv1.DirectoryID(uuid.New())
	dirMap.Store(id, 0)

	// print all dirMap
	dirMap.Range(func(key, value interface{}) bool {
		return true
	})

	// try to list the roots.
	resp, err := cli.ListRoots(context.Background())
	assert.Error(t, err, "expected error listing roots")
	assert.Nil(t, resp, "expected nil response")

	// Check that we have an audit log for this
	assert.Contains(t, auditBuf.String(), "GET:/api/v1/roots")
	auditBuf.Reset()

	// We shouldn't reveal to the user the error.
	// Instead, it should only be logged and viewed by admins.
	assert.NotContains(t, err.Error(), "is not of type", "error contains directory ID")

	// try to get root by ID
	respget, err := cli.GetDirectory(context.Background(), id)
	assert.Error(t, err, "expected error getting directory")
	assert.Nil(t, respget, "expected nil response")

	// Check that we have an audit log for this
	assert.Contains(t, auditBuf.String(), "GET:/api/v1/directories")
	assert.Contains(t, auditBuf.String(), "failed")
	auditBuf.Reset()

	// We shouldn't reveal to the user the error.
	// Instead, it should only be logged and viewed by admins.
	assert.NotContains(t, err.Error(), "is not of type", "error contains directory ID")

	// try to list the children of the root.
	resp, err = cli.GetChildren(context.Background(), id)
	assert.Error(t, err, "expected error listing children")
	assert.Nil(t, resp, "expected nil response")

	// We shouldn't reveal to the user the error.
	// Instead, it should only be logged and viewed by admins.
	assert.NotContains(t, err.Error(), "is not of type", "error contains directory ID")

	// try to list the parents of the root.
	resp, err = cli.GetParents(context.Background(), id)
	assert.Error(t, err, "expected error listing parents")
	assert.Nil(t, resp, "expected nil response")

	// We shouldn't reveal to the user the error.
	// Instead, it should only be logged and viewed by admins.
	assert.NotContains(t, err.Error(), "is not of type", "error contains directory ID")

	// try to create a new directory with the bad root as parent.
	nodir, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "test",
	}, id)
	assert.Error(t, err, "expected error creating directory")
	assert.Nil(t, nodir, "expected nil response")

	// We shouldn't reveal to the user the error.
	// Instead, it should only be logged and viewed by admins.
	assert.NotContains(t, err.Error(), "is not of type", "error contains directory ID")

	// force create a new directory with the bad root as parent.
	dirID := apiv1.DirectoryID(uuid.New())
	meta := &apiv1.DirectoryMetadata{}
	dirMap.Store(dirID, &apiv1.Directory{
		Id:       dirID,
		Name:     "test",
		Metadata: meta,
		Parent:   &id,
	})

	// try to get the directories parents.
	resp, err = cli.GetParents(context.Background(), dirID)
	assert.Error(t, err, "expected error listing parents")
	assert.Nil(t, resp, "expected nil response")

	// We shouldn't reveal to the user the error.
	// Instead, it should only be logged and viewed by admins.
	assert.NotContains(t, err.Error(), "is not of type", "error contains directory ID")

	// try to get the directories children.
	// Note that this will fail for the simple fact that we already
	// have erroneous entries in the directory map.
	resp, err = cli.GetChildren(context.Background(), dirID)
	assert.Error(t, err, "expected error listing children")
	assert.Nil(t, resp, "expected nil response")

	// We shouldn't reveal to the user the error.
	// Instead, it should only be logged and viewed by admins.
	assert.NotContains(t, err.Error(), "is not of type", "error contains directory ID")

	// create yet another subdirectory
	subdir1, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "test",
	}, dirID)
	assert.NoError(t, err, "error creating directory")

	subdir2, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "test",
	}, subdir1.Directory.Id)
	assert.NoError(t, err, "error creating directory")

	// replace subdir1 for erroneous data
	dirMap.Store(subdir1.Directory.Id, 0)

	// list parents until test
	// The idea is that both the given directories are valid, but there are directories in
	// between that are not.
	resp, err = cli.GetParentsUntil(context.Background(), subdir2.Directory.Id, dirID)
	assert.Error(t, err, "expected error listing parents")
	assert.Nil(t, resp, "expected nil response")

	// We shouldn't reveal to the user the error.
	// Instead, it should only be logged and viewed by admins.
	assert.NotContains(t, err.Error(), "is not of type", "error contains directory ID")
}

func TestAuthRequired(t *testing.T) {
	t.Parallel()

	jwksURI := ginjwt.TestHelperJWKSProvider(ginjwt.TestPrivRSAKey1ID, ginjwt.TestPrivRSAKey2ID)

	authConfig := &ginjwt.AuthConfig{
		Enabled:  true,
		Audience: "ginjwt.test",
		Issuer:   "ginjwt.test.issuer",
		JWKSURI:  jwksURI,
	}

	skt := testutils.NewUnixsocketPath(t)
	srv := newTestServer(t, skt, nil, authConfig, nil)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	// Test without auth code

	cli := testutils.NewTestClient(t, skt, getStubServerAddress(t, skt), nil)

	testutils.WaitForServer(t, cli)

	// Create root without authentication
	_, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "root",
	})
	assert.Error(t, err, "expected auth error")

	// List roots without authentication
	_, err = cli.ListRoots(context.Background())
	assert.Error(t, err, "expected auth error")

	// Tests with auth

	cli = testutils.NewTestClient(t, skt, getStubServerAddress(t, skt), authConfig)

	testutils.WaitForServer(t, cli)

	// Create a new root.
	rd, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "root",
	})
	assert.NoError(t, err, "error creating root")
	assert.NotNil(t, rd, "root directory is nil")

	// Get the root.
	listroots, err := cli.ListRoots(context.Background())
	assert.NoError(t, err, "error listing roots")

	assert.Equal(t, 1, len(listroots.Directories), "expected 1 root, got %d", len(listroots.Directories))
}
