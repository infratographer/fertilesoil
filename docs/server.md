# Treeman server

No, `treeman` is not a superhero. It's a server that manages directory trees.

The server contains all basic utilities and operations that the Fertile Soil framework
uses to build multi-tenant platforms. It provides basic CRUD operations on trees,
and also provides for a way to advertise the trees to other services (e.g. using
the notification subsystem).

# Modes of Operation

The server can be run in three modes:

- `admin/read-write`: This is the default mode. It allows for all operations on
  the trees, including creating, deleting, and modifying them.

- `read-only`: This mode allows only for doing read operations on the trees. It
  does not allow for creating, deleting, or modifying them.

- `fast-reads`: This mode is highly dependent on the backend storage system. It
  allows for doing read operations on the trees, but it does not allow for
  creating, deleting, or modifying them. It also allows for doing fast reads on
  the trees, which means that it does not guarantee that the data is consistent
  across all nodes.

These modes are controlled by command line flags.

e.g. to run the server in read-only mode, run:

```bash
$ treeman serve --read-only
```

To run the server in fast-reads mode, run:

```bash
$ treeman serve --fast-reads
```

## Fast Reads

While the first two modes (admin/read-write and read-only) are pretty self-explanatory,
the third mode (fast-reads) is a bit more complicated.

The fast-reads mode is highly dependent on the backend storage system. In the current
implementation's case, CockroachDB, it leverages the Follower Reads feature to allow
for querying the database without having to go through the Raft consensus algorithm.
Instead, it may query any follower holding the data, which means that it may not
be consistent across all nodes.

Given that the access pattern of FertileSoil is read-heavy, this mode is useful
for scaling out the fertile soil deployment accross multiple regions and/or
multiple data centers with high latency.

For more information on CockroachDB's Follower Reads, see the [CockroachDB
documentation](https://www.cockroachlabs.com/docs/stable/follower-reads.html) or
the blog post [Follower Reads in CockroachDB](https://www.cockroachlabs.com/blog/follower-reads-stale-data/).

### Important Note

It is recommended that the `fast-reads` flag be used alongside the `read-only` flag.

It is also recommended that this be done while leveraging [CockroachDB's Non-Voting
Replicas construct](https://www.cockroachlabs.com/docs/stable/architecture/replication-layer.html#non-voting-replicas)

# Database schema setup/migration

The `treeman` command provides a way to setup the database schema and perform
database migrations.

To setup the database schema, run:

```bash
$ treeman migrate
```

Note that this assumes you're providing the database connection details via
environment variables.