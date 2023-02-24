package integration_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/internal/httpsrv/treemanager"
	"github.com/infratographer/fertilesoil/storage/crdb/driver"
	"github.com/infratographer/fertilesoil/tests/integration"
	testutils "github.com/infratographer/fertilesoil/tests/utils"
)

func TestListNoRoots(t *testing.T) {
	t.Parallel()

	skt := testutils.NewUnixsocketPath(t)
	srv := newTestServer(t, skt)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, baseServerAddress, nil)

	testutils.WaitForServer(t, cli)

	rl, err := cli.ListRoots(context.Background())
	assert.NoError(t, err, "error getting roots")
	assert.Equal(t, 0, len(rl.Directories), "unexpected number of directories")
}

func TestListOneRoot(t *testing.T) {
	t.Parallel()

	skt := testutils.NewUnixsocketPath(t)
	srv := newTestServer(t, skt)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, baseServerAddress, nil)

	testutils.WaitForServer(t, cli)

	rd, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "root",
	})

	assert.NoError(t, err, "error creating root")

	// Ensure root parent is empty
	assert.Nil(t, rd.Directory.Parent, "unexpected parent")

	// Ensure root is returned in list
	rl, err := cli.ListRoots(context.Background())
	assert.NoError(t, err, "error getting roots")
	assert.Equal(t, 1, len(rl.Directories), "unexpected number of directories")

	// Ensure GetParents call returns empty list
	parents, err := cli.GetParents(context.Background(), rd.Directory.Id)
	assert.NoError(t, err, "error getting parents")
	assert.Equal(t, 0, len(parents.Directories), "unexpected number of parents")
}

func TestListMultipleRoots(t *testing.T) {
	t.Parallel()

	skt := testutils.NewUnixsocketPath(t)
	srv := newTestServer(t, skt)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, baseServerAddress, nil)

	testutils.WaitForServer(t, cli)

	nroots := 10
	for idx := 0; idx < nroots; idx++ {
		_, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
			Version: apiv1.APIVersion,
			Name:    fmt.Sprintf("root%d", idx),
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

	skt := testutils.NewUnixsocketPath(t)
	srv := newTestServer(t, skt)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, baseServerAddress, nil)

	testutils.WaitForServer(t, cli)

	rd, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "root",
	})
	assert.NoError(t, err, "error creating root")

	dir, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "dir",
	},
		rd.Directory.Id,
	)

	assert.NoError(t, err, "error creating directory")
	assert.Equal(t, "dir", dir.Directory.Name, "unexpected directory name")
	// Ensure parent info is set
	assert.NotNil(t, dir.Directory.Parent, "expected parent")
	assert.Equal(t, rd.Directory.Id, *dir.Directory.Parent, "unexpected parent")

	retd, err := cli.GetDirectory(context.Background(), dir.Directory.Id)
	assert.NoError(t, err, "error getting directory")
	assert.Equal(t, "dir", retd.Directory.Name, "unexpected directory name")
	// Ensure parent info is set
	assert.NotNil(t, dir.Directory.Parent, "expected parent")
	assert.Equal(t, rd.Directory.Id, *retd.Directory.Parent, "unexpected parent")

	parents, err := cli.GetParents(context.Background(), dir.Directory.Id)
	assert.NoError(t, err, "error getting parents")
	assert.Equal(t, 1, len(parents.Directories), "unexpected number of parents")
	assert.Equal(t, rd.Directory.Id, parents.Directories[0], "unexpected parent name")
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

	skt := testutils.NewUnixsocketPath(t)
	srv := newTestServer(t, skt)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, baseServerAddress, nil)

	testutils.WaitForServer(t, cli)

	rd, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "root",
	})
	assert.NoError(t, err, "error creating root")

	// dir1
	_, err = cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "dir",
	},
		rd.Directory.Id,
	)

	assert.NoError(t, err, "error creating directory")

	dir2, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "dir",
	},
		rd.Directory.Id,
	)

	assert.NoError(t, err, "error creating directory")

	dir3, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "dir",
	},
		dir2.Directory.Id,
	)

	assert.NoError(t, err, "error creating directory")

	dir4, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "dir",
	},
		dir3.Directory.Id,
	)

	assert.NoError(t, err, "error creating directory")

	dir5, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "dir",
	},
		dir3.Directory.Id,
	)

	assert.NoError(t, err, "error creating directory")

	dir5parents, err := cli.GetParents(context.Background(), dir5.Directory.Id)
	assert.NoError(t, err, "error getting parents")
	assert.Equal(t, 3, len(dir5parents.Directories), "unexpected number of parents")
	assert.Equal(t, dir3.Directory.Id, dir5parents.Directories[0], "unexpected parent name")
	assert.Equal(t, dir2.Directory.Id, dir5parents.Directories[1], "unexpected parent name")
	assert.Equal(t, rd.Directory.Id, dir5parents.Directories[2], "unexpected parent name")

	dir4parents, err := cli.GetParents(context.Background(), dir4.Directory.Id)
	assert.NoError(t, err, "error getting parents")
	// same number of parents as dir5
	assert.Equal(t, 3, len(dir4parents.Directories), "unexpected number of parents")
	assert.Equal(t, dir3.Directory.Id, dir5parents.Directories[0], "unexpected parent name")
	assert.Equal(t, dir2.Directory.Id, dir5parents.Directories[1], "unexpected parent name")
	assert.Equal(t, rd.Directory.Id, dir5parents.Directories[2], "unexpected parent name")

	dir2children, err := cli.GetChildren(context.Background(), dir2.Directory.Id)
	assert.NoError(t, err, "error getting children")
	assert.Equal(t, 3, len(dir2children.Directories), "unexpected number of children")
	assert.Contains(t, dir2children.Directories, dir3.Directory.Id, "dir2 did not contain dir3 as child")
	assert.Contains(t, dir2children.Directories, dir4.Directory.Id, "dir2 did not contain dir4 as child")
	assert.Contains(t, dir2children.Directories, dir5.Directory.Id, "dir2 did not contain dir5 as child")

	parentsUntil, err := cli.GetParentsUntil(context.Background(), dir5.Directory.Id, dir2.Directory.Id)
	assert.NoError(t, err, "error getting parents")
	assert.Equal(t, 2, len(parentsUntil.Directories), "unexpected number of parents")
	assert.Equal(t, dir3.Directory.Id, parentsUntil.Directories[0], "unexpected parent name")
	assert.Equal(t, dir2.Directory.Id, parentsUntil.Directories[1], "unexpected parent name")
}

func TestCreateRootWithMalformedData(t *testing.T) {
	t.Parallel()

	skt := testutils.NewUnixsocketPath(t)
	srv := newTestServer(t, skt)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, baseServerAddress, nil)

	testutils.WaitForServer(t, cli)

	integration.MalformedDataTest(t, skt, baseServerAddress)
}

func TestServerWithBadDB(t *testing.T) {
	t.Parallel()

	skt := testutils.NewUnixsocketPath(t)

	tl, err := zap.NewDevelopment()
	assert.NoError(t, err, "error creating logger")

	// We're opening a valid database connection, but there's not database set.
	dbconn, err := sql.Open("postgres", baseDBURL.String())
	assert.NoError(t, err, "error creating db connection")

	store := driver.NewDirectoryDriver(dbconn)

	srv := treemanager.NewServer(
		tl,
		dbconn,
		treemanager.WithListen(srvhost),
		treemanager.WithDebug(debug),
		treemanager.WithShutdownTimeout(defaultShutdownTime),
		treemanager.WithUnix(skt),
		treemanager.WithStorageDriver(store),
	)

	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, baseServerAddress, nil)

	t.Log("waiting for server to start. This uses a timer as the database is not set up.")
	time.Sleep(1 * time.Second)

	_, err = cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "root",
	})
	assert.Error(t, err, "expected error creating root")

	_, err = cli.ListRoots(context.Background())
	assert.Error(t, err, "expected error getting roots")

	_, err = cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "dir",
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

	skt := testutils.NewUnixsocketPath(t)
	srv := newTestServer(t, skt)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, baseServerAddress, nil)

	testutils.WaitForServer(t, cli)

	integration.InvalidDirectoryIDsTest(t, cli)
}

func TestCreateErroneousDirectory(t *testing.T) {
	t.Parallel()

	skt := testutils.NewUnixsocketPath(t)
	srv := newTestServer(t, skt)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, baseServerAddress, nil)

	testutils.WaitForServer(t, cli)

	integration.ErroneousDirectoryTest(t, cli)
}

func TestDirectoryNotFound(t *testing.T) {
	t.Parallel()

	skt := testutils.NewUnixsocketPath(t)
	srv := newTestServer(t, skt)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, baseServerAddress, nil)

	testutils.WaitForServer(t, cli)

	integration.DirectoryNotFoundTest(t, cli)
}

func TestDeleteDirectory(t *testing.T) {
	t.Parallel()

	skt := testutils.NewUnixsocketPath(t)
	srv := newTestServer(t, skt)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, baseServerAddress, nil)

	testutils.WaitForServer(t, cli)

	integration.DeleteDirectoryTest(t, cli)
}
