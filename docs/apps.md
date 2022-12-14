Applications
============

Normally, applications are to abide by the following rules:

- They must scope their resources to a specific node in the tree.
- They must not assume that a resource is global.
- They must not modify the tree in any way.
- They must not modify the tree manager in any way.
- They must not modify the storage system in any way.
- They must not assume that the tree manager is running on the same machine.

The tree manager provides the APIs to access the tree, and the applications are
expected to use that to determine what to do.

All components of the platform are merely considered as applications.

Identity
--------

The tree manager does not provide any notion of identity. It is expected that
the applications that use the tree manager will provide their own identity
management. However, the tree manager does provide a way to associate an
identity with a node in the tree. For this, an application will be built
that will associate an identity mapping with a node in the tree.

![Identity](images/identity.jpg)

By associating an identity with a node in the tree, we can provide a way to
further segment tenants. For example, if we have a tenant that has multiple
departments, we can create a node for each department and associate an identity
with each node. This will help us segment the view of the tree for each
sub-tenant and allow for use-cases such as a re-seller and consultants.

In the current view, we think it's best not to deal with identity management and
instead natively rely on users bringing their own. However, to unify the
platform and allow users into the system, we can follow a federated approach and
map tokens coming from upstream identity providers to an approved downstream
identity that we can trust and is scoped to a node in the tree. The idea is to follow
RFC 8693 (OAuth 2.0 Token Exchange) [[1](https://www.rfc-editor.org/rfc/rfc8693.html)],
in order to allow for a token exchange between the upstream identity provider and
the node-scoped platform identity.

To follow the work, see [the DMV project](https://github.com/infratographer/dmv) [[2](https://github.com/infratographer/dmv)].

References
----------

[1] https://www.rfc-editor.org/rfc/rfc8693.html

[2] https://github.com/infratographer/dmv