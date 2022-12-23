package treemanager_test

import (
	"context"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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

func newTestServer(t *testing.T, skt string, store storage.DirectoryAdmin) *common.Server {
	t.Helper()

	if store == nil {
		store, _ = newMemoryStorage(t)
	}

	tl, err := zap.NewDevelopment()
	assert.NoError(t, err, "error creating logger")

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

	skt := testutils.NewUnixsocketPath(t)
	srv := newTestServer(t, skt, nil)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, getStubServerAddress(t, skt))

	testutils.WaitForServer(t, cli)

	// Create a new root.
	rd, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "root",
	})
	assert.NoError(t, err, "error creating root")
	assert.NotNil(t, rd, "root directory is nil")

	// Get the root.
	listroots, err := cli.ListRoots(context.Background())
	assert.NoError(t, err, "error listing roots")

	assert.Equal(t, 1, len(listroots.Directories), "expected 1 root, got %d", len(listroots.Directories))
}

func TestDirectoryOperations(t *testing.T) {
	t.Parallel()

	skt := testutils.NewUnixsocketPath(t)
	srv := newTestServer(t, skt, nil)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, getStubServerAddress(t, skt))

	testutils.WaitForServer(t, cli)

	// Create a new root.
	rd, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "root",
	})
	assert.NoError(t, err, "error creating root")
	assert.NotNil(t, rd, "root directory is nil")

	// Create a new directory.
	d, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "test",
	}, rd.Directory.ID)
	assert.NoError(t, err, "error creating directory")
	assert.NotNil(t, d, "directory is nil")

	// Get the directory.
	retd, err := cli.GetDirectory(context.Background(), d.Directory.ID)
	assert.NoError(t, err, "error getting directory")
	assert.NotNil(t, retd, "directory is nil")

	// directory should be the same as the one we created.
	assert.Equal(t, d.Directory.ID, retd.Directory.ID, "directory is not the same")
	assert.Equal(t, d.Directory.Name, retd.Directory.Name, "directory is not the same")

	// List the directory.
	listd, err := cli.GetChildren(context.Background(), rd.Directory.ID)
	assert.NoError(t, err, "error listing directories")
	assert.NotNil(t, listd, "directory is nil")

	// There should be 1 directory.
	assert.Equal(t, 1, len(listd.Directories), "expected 1 directory, got %d", len(listd.Directories))

	// Get the root as parent.
	listrd, err := cli.GetParents(context.Background(), d.Directory.ID)
	assert.NoError(t, err, "error listing directories")
	assert.NotNil(t, listrd, "directory is nil")

	// There should be 1 directory.
	assert.Equal(t, 1, len(listrd.Directories), "expected 1 directory, got %d", len(listrd.Directories))

	// The directory should be the parent.
	assert.Equal(t, rd.Directory.ID, listrd.Directories[0], "directory is not the same")

	// Get the root as parent with GetParentsUntil function
	listrd, err = cli.GetParentsUntil(context.Background(), d.Directory.ID, rd.Directory.ID)
	assert.NoError(t, err, "error listing directories")
	assert.NotNil(t, listrd, "directory is nil")

	// There should be 1 directory.
	assert.Equal(t, 1, len(listrd.Directories), "expected 1 directory, got %d", len(listrd.Directories))

	// The directory should be the parent.
	assert.Equal(t, rd.Directory.ID, listrd.Directories[0], "directory is not the same")
}

func TestErroneousCalls(t *testing.T) {
	t.Parallel()

	skt := testutils.NewUnixsocketPath(t)
	srv := newTestServer(t, skt, nil)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	srvAddr := getStubServerAddress(t, skt)
	cli := testutils.NewTestClient(t, skt, srvAddr)

	testutils.WaitForServer(t, cli)

	integration.MalformedDataTest(t, skt, srvAddr)
}

func TestInvalidDirectoryIDs(t *testing.T) {
	t.Parallel()

	skt := testutils.NewUnixsocketPath(t)
	srv := newTestServer(t, skt, nil)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	srvAddr := getStubServerAddress(t, skt)
	cli := testutils.NewTestClient(t, skt, srvAddr)

	testutils.WaitForServer(t, cli)

	integration.InvalidDirectoryIDsTest(t, cli)
}

func TestErroneousDirectories(t *testing.T) {
	t.Parallel()

	skt := testutils.NewUnixsocketPath(t)
	srv := newTestServer(t, skt, nil)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	srvAddr := getStubServerAddress(t, skt)
	cli := testutils.NewTestClient(t, skt, srvAddr)

	testutils.WaitForServer(t, cli)

	integration.ErroneousDirectoryTest(t, cli)
}

func TestDirectoryNotFound(t *testing.T) {
	t.Parallel()

	skt := testutils.NewUnixsocketPath(t)
	srv := newTestServer(t, skt, nil)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	srvAddr := getStubServerAddress(t, skt)
	cli := testutils.NewTestClient(t, skt, srvAddr)

	testutils.WaitForServer(t, cli)

	integration.DirectoryNotFoundTest(t, cli)
}

func TestErrorDoesntLeakInfo(t *testing.T) {
	t.Parallel()

	skt := testutils.NewUnixsocketPath(t)

	store, dirMap := newMemoryStorage(t)

	srv := newTestServer(t, skt, store)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	srvAddr := getStubServerAddress(t, skt)
	cli := testutils.NewTestClient(t, skt, srvAddr)

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

	// We shouldn't reveal to the user the error.
	// Instead, it should only be logged and viewed by admins.
	assert.NotContains(t, err.Error(), "is not of type", "error contains directory ID")

	// try to get root by ID
	respget, err := cli.GetDirectory(context.Background(), id)
	assert.Error(t, err, "expected error getting directory")
	assert.Nil(t, respget, "expected nil response")

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
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "test",
	}, id)
	assert.Error(t, err, "expected error creating directory")
	assert.Nil(t, nodir, "expected nil response")

	// We shouldn't reveal to the user the error.
	// Instead, it should only be logged and viewed by admins.
	assert.NotContains(t, err.Error(), "is not of type", "error contains directory ID")

	// force create a new directory with the bad root as parent.
	dirID := apiv1.DirectoryID(uuid.New())
	dirMap.Store(dirID, &apiv1.Directory{
		ID:       dirID,
		Name:     "test",
		Metadata: map[string]string{},
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
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "test",
	}, dirID)
	assert.NoError(t, err, "error creating directory")

	subdir2, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		DirectoryRequestMeta: apiv1.DirectoryRequestMeta{
			Version: apiv1.APIVersion,
		},
		Name: "test",
	}, subdir1.Directory.ID)
	assert.NoError(t, err, "error creating directory")

	// replace subdir1 for erroneous data
	dirMap.Store(subdir1.Directory.ID, 0)

	// list parents until test
	// The idea is that both the given directories are valid, but there are directories in
	// between that are not.
	resp, err = cli.GetParentsUntil(context.Background(), subdir2.Directory.ID, dirID)
	assert.Error(t, err, "expected error listing parents")
	assert.Nil(t, resp, "expected nil response")

	// We shouldn't reveal to the user the error.
	// Instead, it should only be logged and viewed by admins.
	assert.NotContains(t, err.Error(), "is not of type", "error contains directory ID")
}
