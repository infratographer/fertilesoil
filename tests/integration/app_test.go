package integration_test

import (
	"context"
	"testing"

	natsgo "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	appv1 "github.com/infratographer/fertilesoil/app/v1"
	clientv1nats "github.com/infratographer/fertilesoil/client/v1/nats"
	"github.com/infratographer/fertilesoil/notifier/nats"
	natsutils "github.com/infratographer/fertilesoil/notifier/nats/utils"
	testutils "github.com/infratographer/fertilesoil/tests/utils"
)

// This scenario tests the following:
//  1. Create a new application
//  2. Create a new directory
//  3. The application should be notified of the new directory
//  4. The application should be notified of new subdirectories
//  5. Even if we'd loose access to the notifications and reconnect, the application
//     should be notified of the new directories via the reconcile loop
func TestAppReconcileAndWatch(t *testing.T) {
	t.Parallel()

	// initialize socket to communicate with the tree manager
	skt := testutils.NewUnixsocketPath(t)

	subject := t.Name()

	// initialize NATS server for notifications
	natss, natserr := natsutils.StartNatsServer(subject)
	assert.NoError(t, natserr, "error starting nats server")

	defer natss.Shutdown()

	conn, err := natsgo.Connect(natss.ClientURL())
	assert.NoError(t, err, "connecting to nats server")

	js, err := conn.JetStream()
	assert.NoError(t, err, "creating JetStream connection")

	clientconn, err := natsgo.Connect(natss.ClientURL())
	assert.NoError(t, err, "connecting to nats server")

	natsutils.WaitConnected(t, conn)
	natsutils.WaitConnected(t, clientconn)

	// build notifier
	ntf := nats.NewNotifier(js, subject)

	// Build tree manager server
	srv := newTestServerWithNotifier(t, skt, ntf)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, baseServerAddress, nil)

	testutils.WaitForServer(t, cli)

	// initialize root. An app needs a root to be initialized
	rd, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "root",
	})
	assert.NoError(t, err, "error creating root")

	// Set up test application
	appstore := setupAppStorage(t)

	watcher, err := clientv1nats.NewSubscriber(clientconn, subject+".*")
	assert.NoError(t, err, "error creating nats subscriber")

	appctrl, apptester := setupTestApp(t, rd.Directory.Id, cli, watcher, appstore)

	cancelCtx, cancel := context.WithCancel(context.Background())

	// Run application controller. This will initialize the application and
	// start watching for changes
	go func() {
		runerr := appctrl.Run(cancelCtx)
		assert.ErrorIs(t, runerr, context.Canceled, "expected context canceled error")
	}()

	apptester.waitForReconcile()

	evts := apptester.popEvents()

	// We should have done one reconcile for the root
	assert.Len(t, evts, 1, "expected 1 event")

	// We should have created the root
	assert.Equal(t, apiv1.EventTypeCreate, evts[0].Type, "expected created event")

	// We should only have one reconcile call
	assert.Equal(t, apptester.getReconcileCalls(), uint32(1), "expected 1 reconcile call")

	// Create a directory
	_, err = cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "test",
	}, rd.Directory.Id)
	assert.NoError(t, err, "error creating directory")

	apptester.waitForReconcile()

	evts = apptester.popEvents()

	// We should have done one reconcile for the new directory
	assert.Len(t, evts, 1, "expected 1 event")

	// We should have created the directory
	assert.Equal(t, apiv1.EventTypeCreate, evts[0].Type, "expected created event")

	// We should only have two reconcile calls
	assert.Equal(t, apptester.getReconcileCalls(), uint32(2), "expected 2 reconcile calls")

	// The app now looses access to notifications

	clientconn.Close()

	// Create a subdirectory
	_, err = cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "subtest",
	}, rd.Directory.Id)
	assert.NoError(t, err, "error creating directory")

	// wait for a full reconcile
	apptester.waitForReconcile()

	// we should have 2 reconcile calls
	// as the directories are up-to-date
	assert.Equal(t, apptester.getReconcileCalls(), uint32(3), "expected 3 reconcile calls")

	evts = apptester.popEvents()

	// We should have done one reconcile for the new directory
	assert.Len(t, evts, 1, "expected 1 event")

	// We should have created the directory
	assert.Equal(t, apiv1.EventTypeCreate, evts[0].Type, "expected created event")

	cancel()
}

// This scenario tests the following:
//  1. Create a new application
//  2. Create a new directory
//  3. The application should be notified of the new directory
//  4. The application should be notified of new subdirectories
func TestAppWatchWithoutClient(t *testing.T) {
	t.Parallel()

	// initialize socket to communicate with the tree manager
	skt := testutils.NewUnixsocketPath(t)

	subject := t.Name()

	// initialize NATS server for notifications
	natss, natserr := natsutils.StartNatsServer(subject)
	assert.NoError(t, natserr, "error starting nats server")

	defer natss.Shutdown()

	conn, err := natsgo.Connect(natss.ClientURL())
	assert.NoError(t, err, "connecting to nats server")

	js, err := conn.JetStream()
	assert.NoError(t, err, "creating JetStream connection")

	clientconn, err := natsgo.Connect(natss.ClientURL())
	assert.NoError(t, err, "connecting to nats server")

	natsutils.WaitConnected(t, conn)
	natsutils.WaitConnected(t, clientconn)

	// build notifier
	ntf := nats.NewNotifier(js, subject)

	// Build tree manager server
	srv := newTestServerWithNotifier(t, skt, ntf)
	defer func() {
		err := srv.Shutdown()
		assert.NoError(t, err, "error shutting down server")
	}()

	go testutils.RunTestServer(t, srv)

	cli := testutils.NewTestClient(t, skt, baseServerAddress, nil)

	testutils.WaitForServer(t, cli)

	// initialize root. An app needs a root to be initialized
	rd, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "root",
	})
	assert.NoError(t, err, "error creating root")

	// Set up test application
	appstore := setupAppStorage(t)

	watcher, err := clientv1nats.NewSubscriber(clientconn, subject+".*")
	assert.NoError(t, err, "error creating nats subscriber")

	fullrec, err := appv1.NewSeeder(rd.Directory.Id, cli, appstore)
	assert.NoError(t, err, "error creating full subtree reconciler")

	// Trigger a full reconcile
	err = fullrec.InitializeDirectories(context.Background())
	assert.NoError(t, err, "error initializing directories")

	// We don't pass a client to the application controller
	appctrl, apptester := setupTestApp(t, rd.Directory.Id, nil, watcher, appstore)

	cancelCtx, cancel := context.WithCancel(context.Background())

	// Run application controller. This will initialize the application and
	// start watching for changes
	go func() {
		runerr := appctrl.Run(cancelCtx)
		assert.ErrorIs(t, runerr, context.Canceled, "expected context canceled error")
	}()

	// At this point we don't have any events yet. Let's trigger some!

	// Create a directory
	_, err = cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "test",
	}, rd.Directory.Id)
	assert.NoError(t, err, "error creating directory")

	apptester.waitForReconcile()

	evts := apptester.popEvents()

	// We should have done one reconcile for the new directory
	assert.Len(t, evts, 1, "expected 1 event")

	// We should have created the directory
	assert.Equal(t, apiv1.EventTypeCreate, evts[0].Type, "expected created event")

	// We should only have two reconcile calls
	assert.Equal(t, apptester.getReconcileCalls(), uint32(1), "expected 1 reconcile calls")

	// Create a subdirectory
	_, err = cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "subtest",
	}, rd.Directory.Id)
	assert.NoError(t, err, "error creating directory")

	// wait for a full reconcile
	apptester.waitForReconcile()

	// we should have 2 reconcile calls
	// as the directories are up-to-date
	assert.Equal(t, apptester.getReconcileCalls(), uint32(2), "expected 2 reconcile calls")

	evts = apptester.popEvents()

	// We should have done one reconcile for the new directory
	assert.Len(t, evts, 1, "expected 1 event")

	// We should have created the directory
	assert.Equal(t, apiv1.EventTypeCreate, evts[0].Type, "expected created event")

	cancel()
}
