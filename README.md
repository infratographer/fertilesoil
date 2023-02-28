Fertile Soil
===========
Status
------
![Tests](https://github.com/infratographer/fertilesoil/actions/workflows/test.yml/badge.svg)
![Security](https://github.com/infratographer/fertilesoil/actions/workflows/security.yml/badge.svg)
![CodeQL](https://github.com/infratographer/fertilesoil/actions/workflows/codeql-analysis.yml/badge.svg)
![Dependency Review](https://github.com/infratographer/fertilesoil/actions/workflows/dependency-review.yml/badge.svg)
![release main](https://github.com/infratographer/fertilesoil/actions/workflows/release-latest.yml/badge.svg)
![release](https://github.com/infratographer/fertilesoil/actions/workflows/release.yml/badge.svg)

Summary
-------

It's what needed for healthy trees to grow in.

Fertile Soil is a framework to build platforms with. It builds upon the concept
of a tree (a directory structure) which is the basis of a multi-tenant platform.

It provides a tree representation, as well as the backend to store it in a
database. It also provides an HTTP API to access the tree.

For more information on multi-tenancy, view the [Multi-Tenancy](docs/multitenancy.md) doc.

The Tree(s)
-----------

The overall model is as follows:

![Tree structure overview](/docs/images/trees.jpg)

Since the intent is to build multi-tenant platforms, what is a platform without
applications? In this model, everything is scoped to the tree, and thus
applications are meant to be scoped to specific nodes.

The intention is to build a bunch of micro-services that would call the tree manager
to get the tree for a given tenant, and then use that tree to determine what
to do.

### What's provided?

The tree manager provides the following:
- A directory tree representation
- A storage system to store the tree in a database
- APIs to access and manage the tree

### What's not provided?

The tree manager does not provide the following:

- A way to authenticate users
- A way to authorize users
- A way to manage users
- A way to manage applications
- Resources for applications

All of these are application-specific and are not provided by the tree manager.

Components
----------

### API

The API defines the structures that both the client, server and storage system use.

In order to accommodate to changing requirements and needs, the API is kept simple
and only defines the structures that are needed to build the platform.

The main structures are located in [`api/v1/directory.go`](api/v1/directory.go)

### Storage System

The storage system contains the implementation details of how the tree is stored
in the database.

Currently, the only implementation is using CockroachDB, but the intention is to
allow for other implementations to be used.

Any storage implementation needs to implement the [`storage.Storage`](storage/interface.go) interface.

Since the access pattern is read-heavy, the storage system needs to be optimized for that.

### Notification System

The notification system is used to notify applications of changes to the tree.

Given that applications wouldn't have direct access to the database, it is up
to the individual applications to subscribe to the notification system to know
if a child node has been added or removed and update their internal state
accordingly.

Any notification system needs to implement the [`notification.Notifier`](notification/interface.go)
interface.

### Tree Manager

A sample server implementation is provided in as the `treeman` command that's built
as part of this project. Further documentation exists in the [Tree Manager Server](docs/server.md) doc.

### Tree Client

A client library that can be used to access the tree manager [is provided](client/v1).

### An Application Framework

The app framework is a framework that can be used to build applications that are
scoped to a specific node in the tree. The intent is for service authors to
use this framework to seamlessly get events on the tree and update their internal
state accordingly. There relevant files are located in the [`app`](app) directory,
and documentation on how to use it is provided in the
[Application Framework](docs/appframework.md) doc.

Applications
------------

In this model, nothing is special about applications. They are all expected to just 
scoped to a specific node in the tree. The tree manager provides the APIs to access
the tree, and the applications are expected to use that to determine what to do.

Thus, any notion of a global resource or application is not provided by the tree manager nor encouraged.

For a more detailed descriptions on the components or applications that
are to be built for this platform, view the [Applications](docs/apps.md) doc.

Development
-----------

This project depends on a couple of services in order to run.
A database, in this case CockroachDB and NATS an event broker / event handler.

To simplify local testing, a [docker compose](compose.yaml) file has been created
which will spin up basic implementations of both of these services.
It's recommended however to use the `make` target `dev-infra-up` which will generate
the necessary dependencies to start these services.

Example output:

```shell
$ make dev-infra-up
Generating nats .dc-data/nkey.key
Generating nats .dc-data/nkey.pub
Generating OAuth2 config .dc-data/oauth2.json
Starting services
[+] Running 4/4
 ⠿ Network fertilesoil_default                 Created                                                                                                                                                0.1s
 ⠿ Container fertilesoil-crdb-1                Started                                                                                                                                                0.8s
 ⠿ Container fertilesoil-nats-1                Started                                                                                                                                                0.8s
 ⠿ Container fertilesoil-mock-oauth2-server-1  Started                                                                                                                                                0.9s
Running migrations
{"level":"info","ts":1674847104.337401,"caller":"cmd/migrate.go:44","msg":"executing migrations","app":"treemanager","version":"unknown"}
2023/01/27 19:18:24 OK   20221222105349_init.sql (5.81ms)
2023/01/27 19:18:24 goose: no migrations to run. current version: 20221222105349
2023/01/27 19:18:24 OK   20230101000000_init.sql (14.94ms)
2023/01/27 19:18:24 goose: no migrations to run. current version: 20230101000000

Use "make dev-oauth2-token" to create a token
```

Next create an OIDC token:

```shell
$ make dev-oauth2-token
Generating OAuth2 token
Audience: fertilesoil
Issuer: http://localhost:8082/fertilesoil
JWKS URL: http://localhost:8082/fertilesoil/jwks
token_type      Bearer
access_token    eyJraWQiO...SNIP...bTSVi5a6w
expires_in      119
scope   test
```

Now you may run treeman:

```shell
$ export FERTILESOIL_CRDB_HOST=localhost:26257
$ export FERTILESOIL_CRDB_USER=root
$ export FERTILESOIL_CRDB_PARAMS=sslmode=disable
$ export FERTILESOIL_OIDC_AUDIENCE=fertilesoil
$ export FERTILESOIL_OIDC_ISSUER=http://localhost:8082/fertilesoil
$ export FERTILESOIL_OIDC_JWKSURI=http://localhost:8082/fertilesoil/jwks
$ go run ./main.go serve --nats-url 127.0.0.1:4222 --nats-nkey .dc-data/nkey.key --audit-log-path ./.dc-data/audit/audit.log
[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /metrics                  --> github.com/zsais/go-gin-prometheus.prometheusHandler.func1 (6 handlers)
[GIN-debug] GET    /livez                    --> github.com/infratographer/fertilesoil/internal/httpsrv/common.(*Server).livenessCheckHandler-fm (6 handlers)
[GIN-debug] GET    /readyz                   --> github.com/infratographer/fertilesoil/internal/httpsrv/common.(*Server).readinessCheckHandler-fm (6 handlers)
[GIN-debug] GET    /api                      --> github.com/infratographer/fertilesoil/internal/httpsrv/treemanager.apiVersionHandler (7 handlers)
[GIN-debug] GET    /api/v1                   --> github.com/infratographer/fertilesoil/internal/httpsrv/treemanager.apiVersionHandler (8 handlers)
[GIN-debug] GET    /api/v1/roots             --> github.com/infratographer/fertilesoil/internal/httpsrv/treemanager.listRoots.func1 (8 handlers)
[GIN-debug] POST   /api/v1/roots             --> github.com/infratographer/fertilesoil/internal/httpsrv/treemanager.createRootDirectory.func1 (8 handlers)
[GIN-debug] GET    /api/v1/directories/:id   --> github.com/infratographer/fertilesoil/internal/httpsrv/treemanager.getDirectory.func1 (8 handlers)
[GIN-debug] POST   /api/v1/directories/:id   --> github.com/infratographer/fertilesoil/internal/httpsrv/treemanager.createDirectory.func1 (8 handlers)
[GIN-debug] DELETE /api/v1/directories/:id   --> github.com/infratographer/fertilesoil/internal/httpsrv/treemanager.deleteDirectory.func1 (8 handlers)
[GIN-debug] GET    /api/v1/directories/:id/children --> github.com/infratographer/fertilesoil/internal/httpsrv/treemanager.listChildren.func1 (8 handlers)
[GIN-debug] GET    /api/v1/directories/:id/parents --> github.com/infratographer/fertilesoil/internal/httpsrv/treemanager.listParents.func1 (8 handlers)
[GIN-debug] GET    /api/v1/directories/:id/parents/:until --> github.com/infratographer/fertilesoil/internal/httpsrv/treemanager.listParentsUntil.func1 (8 handlers)
{"level":"info","ts":1674847681.642747,"caller":"common/common.go:129","msg":"listening on","app":"treemanager","version":"unknown","address":":8080"}
```

Run `make help` for additional useful commands.

Now you can test your connection with:

```shell
$ BEARER="eyJraWQiO...SNIP...bTSVi5a6w"
$ curl -H "Authorization: Bearer $BEARER" localhost:8080/api/v1/roots
{"directories":null,"version":"v1"}
```
