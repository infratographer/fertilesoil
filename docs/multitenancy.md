Introduction
============

As mentioned in the [README](../README/md), the intention of this project is to
provide a the means of building multi-tenant platforms and applications that
understand the divisions between tenants.

## Multi-Tenancy

Tenancy is the concept of providing isolated access to resources. In the context of
this project, we are talking about the isolation of data and the ability to provide
a single application to multiple tenants. If a system is multi-tenant, it means that
it can provide a single application to multiple tenants. The tenants are isolated
from each other and the application is aware of the tenants.

Many other open-source projects have implemented tenancy concepts in different ways.

Kubernetes, for example, has a concept of namespaces. Namespaces are a way of
providing isolation between tenants. Based on this construct, Kubernetes separates
resources and allows for segmentation via constructs like RBAC, Network Policies and
Resource Quotas [[1](https://kubernetes.io/docs/concepts/security/multi-tenancy/)].
Kubernetes resources may be scoped to namespaces or cluster-wide.
It is with these cluster-wide resources that Kubernetes has limitations in providing
full multi-tenancy, since segmenting said resources requires authorization or 
admission constructs not provided by the platform.

OpenStack is another example of a system providing multi-tenancy. OpenStack presents
the concept of domains and projects
[[2](https://access.redhat.com/documentation/en-us/red_hat_openstack_platform/13/html/users_and_identity_management_guide/projects)].
A domain is a collection of projects. A project is a collection of resources and a way
of isolating them. A project, in this case, represents a tenant.

The multiple services that form OpenStack are aware of the projects and provide the
means of segmenting resources. This allows for a single application to be provided
to multiple tenants. Quotas, policies, and role assignments are all provided by
OpenStack and are aware of the tenants.

### Hierarchical multi-tenancy

Later in OpenStack's development, the concept of hierarchical multi-tenancy was
introduced
[[3](https://youtu.be/KvKiAzKSVYs)]
[[4](https://object-storage-ca-ymq-1.vexxhost.net/swift/v1/6e4619c416ff4bd19e1c087f27a43eea/www-assets-prod/presentation-media/Flat-no-more-Hierarchical-multitenancy-and-projects-acting-as-domains-in-OpenStack.pdf)].
The idea is that a project can have sub-projects. This allows for a
hierarchy of tenants. The hierarchy is not limited to a single level, so a tenant
can have sub-tenants that have sub-tenants. This allows for a tree-like structure
of tenants. Domains were not removed, but instead the presence of a domain
was replicated by a project acting as a domain. This allows for a domain to take
advantage of the security and resource constructs that projects provide, like quotas,
role assignments, and policies.

Hierarchical multi-tenancy is not a common concept in Kubernetes. However, it was
recently introduced in the form of Hierarchical Namespaces
[[5](https://youtu.be/j5x6NumP21c)] which are usable through the Hierarchical
Namespaces Controller. Similarly to OpenStack, the idea is that a namespace may
have sub-namespaces. This which also allows for a hierarchy of tenants. Having
the added advantage of providing role inheritance, and resource access which was
tedious to implement before. Note that this is not a concept that is provided by
Kubernetes itself, but by a third-party controller.

The concept of hierarchical multi-tenancy is not limited to Kubernetes and OpenStack.
It is a concept that can be applied to any system that provides multi-tenancy.
In fact, other systems like GCP provide hierarchical multi-tenancy. In GCP the tree
structure is very clear and is represented by the organization, folders, and projects 
[[5](https://cloud.google.com/resource-manager/docs/cloud-platform-resource-hierarchy)].

### Hierarchical multi-tenancy in this project

This project provides the means of building multi-tenant platforms and applications
that understand the divisions between tenants. The project also provides the means
of providing hierarchical multi-tenancy. In this project, a tenant is represented as
a directory. A directory tree is a way of representing the hierarchy of tenants.

By centralizing tree management, applications can be built around these concepts,
and thus provide a concise way of accessing, segmenting and managing resources.

The goal is to provide a platform where applications don't need to re-implement
the tree logic and thus can focus on their core functionality.

Both OpenStack and Kubernetes suffered from the decision to treat a global resource
as something special. In OpenStack, domains eventually were projected as projects
to address this issue. In Kubernetes, global resources often require special
authorization or admission constructs to be able to segment them. In this project,
the idea is to learn from this and not give any special care to nodes in the tree.
Thus, there will be no concept of something global, instead it should just be scoped
to the root of the tree. If later on, the need arises to de-scope a resource, it
can be done without breaking the API.

# References

[1] https://kubernetes.io/docs/concepts/security/multi-tenancy/

[2] https://access.redhat.com/documentation/en-us/red_hat_openstack_platform/13/html/users_and_identity_management_guide/projects

[3] https://youtu.be/KvKiAzKSVYs

[4] https://object-storage-ca-ymq-1.vexxhost.net/swift/v1/6e4619c416ff4bd19e1c087f27a43eea/www-assets-prod/presentation-media/Flat-no-more-Hierarchical-multitenancy-and-projects-acting-as-domains-in-OpenStack.pdf

