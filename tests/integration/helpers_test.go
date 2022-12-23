package integration_test

import (
	"context"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	appv1 "github.com/infratographer/fertilesoil/app/v1"
	appv1sql "github.com/infratographer/fertilesoil/app/v1/sql"
	appsqlmig "github.com/infratographer/fertilesoil/app/v1/sql/migrations"
	clientv1 "github.com/infratographer/fertilesoil/client/v1"
	"github.com/infratographer/fertilesoil/internal/httpsrv/common"
	"github.com/infratographer/fertilesoil/internal/httpsrv/treemanager"
	"github.com/infratographer/fertilesoil/notifier"
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

	// Goose is not thread-safe, so we need to lock it.
	gooseDBMutex sync.Mutex
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

func newTestServer(t *testing.T, skt string) *common.Server {
	t.Helper()

	return newTestServerWithNotifier(t, skt, nil)
}

func newTestServerWithNotifier(t *testing.T, skt string, notif notifier.Notifier) *common.Server {
	t.Helper()

	tl, err := zap.NewDevelopment()
	assert.NoError(t, err, "error creating logger")

	gooseDBMutex.Lock()

	dbconn := dbutils.GetNewTestDB(t, baseDBURL)

	gooseDBMutex.Unlock()

	tm := treemanager.NewServer(
		tl,
		dbconn,
		treemanager.WithListen(srvhost),
		treemanager.WithDebug(debug),
		treemanager.WithShutdownTimeout(defaultShutdownTime),
		treemanager.WithUnix(skt),
		treemanager.WithNotifier(notif),
	)

	return tm
}

func setupAppStorage(t *testing.T) appv1.AppStorage {
	t.Helper()

	gooseDBMutex.Lock()
	defer gooseDBMutex.Unlock()
	dbConn := dbutils.GetNewTestDBForApp(t, baseDBURL)

	err := appsqlmig.BootStrap("postgres", dbConn)
	assert.NoError(t, err, "error bootstrapping app storage")

	return appv1sql.New(dbConn)
}

func setupTestApp(
	t *testing.T,
	basedir apiv1.DirectoryID,
	cli clientv1.ReadOnlyClient,
	w clientv1.Watcher,
	store appv1.AppStorage,
) (appv1.Controller, *appReconciler) {
	t.Helper()

	r := newAppReconciler()

	app, err := appv1.NewController(
		basedir,
		appv1.WithClient(cli),
		appv1.WithWatcher(w),
		appv1.WithStorage(store),
		appv1.WithReconciler(r),
		appv1.WithFullReconcileInterval(1, 2, time.Second),
	)
	if err != nil {
		t.Fatalf("error creating app: %v", err)
	}

	return app, r
}

type appReconciler struct {
	trackedEvents  []apiv1.DirectoryEvent
	mutex          sync.Mutex
	reconcileCalls atomic.Uint32
	// Is a channel that can be used to block the reconciler
	reconciled chan struct{}
}

func newAppReconciler() *appReconciler {
	return &appReconciler{
		trackedEvents:  make([]apiv1.DirectoryEvent, 0),
		reconcileCalls: atomic.Uint32{},
		reconciled:     make(chan struct{}, 10),
	}
}

// the app reconciler simply tracks the events it receives.
//
// Note that we don't use a pointer in this function since we need to
// abide by the reconciler interface. This is meant for immutability.
//
//nolint:gocritic // We don't want to use a pointer here
func (ar *appReconciler) Reconcile(ctx context.Context, ev apiv1.DirectoryEvent) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()
	defer ar.reconcileCalls.Add(1)

	ar.trackedEvents = append(ar.trackedEvents, ev)

	ar.reconciled <- struct{}{}

	return nil
}

// popEvents returns the events tracked by the reconciler and clears the
// tracked events slice.
func (ar *appReconciler) popEvents() []apiv1.DirectoryEvent {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	events := ar.trackedEvents
	ar.trackedEvents = make([]apiv1.DirectoryEvent, 0)
	return events
}

func (ar *appReconciler) getReconcileCalls() uint32 {
	return ar.reconcileCalls.Load()
}

func (ar *appReconciler) waitForReconcile() {
	<-ar.reconciled
}
