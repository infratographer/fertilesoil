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

### Tree Manager

A sample server implementation is provided in as the `treeman` command that's built
as part of this project.

### Tree Client

A client library that can be used to access the tree manager [is provided](client/v1).
