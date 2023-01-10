# Application framework

As mentioned in the [Applications documentation](docs/apps.md), an application
is any program that interfaces with the directory tree. The application framework
is a framework that can be used to build applications that are scoped to a specific
node in the tree. Applications don't necessarily need to know or care about
the tree structure, but they do need to keep track of changes in the tree that may
pertain to the sub-tree they're following.

Thus, the intent of the application framework is to make it easy for service
authors to bootstrap services that may natively use directories as means
for multi-tenancy.

## Setup / Usage

As a service author you need to import the relevant packages and initialize
the application framework. By the default, the application framework provides
a SQL store implementation with a schema that helps you track directories.

If you're using SQL and want to take this into use. you may add the following
to you're migration script.

```go
import (
    ...
    appsqlmig "github.com/infratographer/fertilesoil/app/v1/sql/migrations"
    ...
)

func Migrate(db *sql.DB) error {
	if err := goose.SetDialect(dialect); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	// This ensures that we have the latest version of the app migrations
	// in the database. This is where we get the tracked_directories table
	// and the app migrations are added to it.
	if err := appsqlmig.BootStrap(dialect, db); err != nil {
		return fmt.Errorf("failed to bootstrap app migrations: %w", err)
	}

	goose.SetBaseFS(migrations)

	return goose.Up(db, ".")
}
```

Note that the above snippet assumes you're using the [goose tooling](https://github.com/pressly/goose)
to do SQL migrations. Given that app migrations are tied to the application framework,
by importing the app package, your application will automatically get the latest
schema.

```go
import (
    ...
    apiv1 "github.com/infratographer/fertilesoil/api/v1"
	appv1 "github.com/infratographer/fertilesoil/app/v1"
	appv1sql "github.com/infratographer/fertilesoil/app/v1/sql"
	clientv1 "github.com/infratographer/fertilesoil/client/v1"
	cv1nats "github.com/infratographer/fertilesoil/client/v1/nats"
    ...
)

func setupApp() error {}
	// Initialize database connection
	dbconn := // ...

	// Initialize app storage
	appStore := appv1sql.New(dbconn)

	// Initialize NATS connection
	natsconn := // ...

	// Create NATS directory subscriber
	watcher, err := cv1nats.NewSubscriber(natsconn, viper.GetString("nats.directories_subjects"))
	if err != nil {
		return fmt.Errorf("failed to create nats subscriber: %w", err)
	}

	// Create directory client
	dirclient := clientv1.NewHTTPClient(
        // ...
    )

	// Initialize our reconciler
	r := reconciler.NewReconciler()

	// Get base directory
	rawID := v.GetString("base_directory_id")

	baseDirID, err := apiv1.ParseDirectoryID(rawID)
	if err != nil {
		return fmt.Errorf("failed to parse base directory id: %w", err)
	}

	ctrl, err := appv1.NewController(
		baseDirID,
		appv1.WithStorage(appStore),
		appv1.WithWatcher(watcher),
		appv1.WithClient(dirclient),
		appv1.WithReconciler(r),
	)
	if err != nil {
		return fmt.Errorf("failed to create directory controller: %w", err)
	}

	ctx := cmd.Context()

	go func() {
		if err := ctrl.Run(ctx); err != nil {
			logger.Fatal("failed to run controller", zap.Error(err))
		}
	}()

    return nil
}
```

The snippet above will initialize the application framework and start the
controller. The controller will receive events pertaining to directories
and ensure they're appropriately tracked in the application's database.

The controller needs the following components in order to function:
* A storage implementation that implements the `appv1.AppStorage` interface.
* A watcher implementation that implements the `clientv1.Watcher` interface.
* A directory client implementation that implements the `clientv1.ReadOnlyClient` interface.
* A reconciler implementation that implements the `appv1.Reconciler` interface.

There are a few implementations of the above interfaces that you can use
to get started. The `appv1sql` package provides a SQL implementation of the
`appv1.AppStorage` interface. The `cv1nats` package provides a NATS
implementation of the `clientv1.Watcher` interface. The `clientv1` package
provides a HTTP implementation of the `clientv1.ReadOnlyClient` interface.

The reconciler is the most important component of the application framework.
It is responsible for ensuring that the application is in sync
with the directory tree. The reconciler is also responsible for ensuring that
the application is aware of the directories that are relevant to it.

In order to implement a reconciler, you need to implement the `appv1.Reconciler`
interface. The reconciler interface has a single method `Reconcile` that
receives a golang context and a `apiv1.DirectoryEvent`.

A sample and empty implementation would look as follows:

```go
import (
	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	appv1 "github.com/infratographer/fertilesoil/app/v1"
)

type Reconciler struct{}

var _ appv1.Reconciler = &Reconciler{}

func NewReconciler() *Reconciler {
	return &Reconciler{}
}

//nolint:gocritic // passing the directory event by value ensures we don't modify it
func (r *Reconciler) Reconcile(ctx context.Context, evt apiv1.DirectoryEvent) error {
	// Here we will look at a directory event and perform the appropriate
	// action on the database.
	return nil
}
```

Note that the `Reconcile` method will only be called if there are changes to
a particular directory.
