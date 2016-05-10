---
layout: "pluginpatterns"
page_title: "Resource Design Patterns"
sidebar_current: "docs-plugins-patterns-resources"
description: |-
  Design patterns for writing Terraform resources.
---

# Resource Design Patterns

A *Resource* represents some object type for Terraform to instantiate and
manage in an external service. The service itself is represented by a
*Provider*, and these have [their own set of design patterns](providers.html).

## Resource Names

Resource names are nouns, since resource blocks each represent a single
object Terraform is managing. Resource names must always start with their
containing provider's name followed by an underscore, so a resource from
the provider `postgresql` might be named `postgresql_database`.

It is preferable to use resource names that will be familiar to those with
prior experience using the service in question, e.g. via a web UI it provides.

## First-class Resources

First-class resources represent the main entities in a system. For example,
a provider of virtual machines may have entities like "machine",
"network interface", and "machine image" that may be named
`example_machine`, `example_network_interface`, and `example_machine_image`.

A first-class resource instance will usually map one-to-one with an object
in the target system. The `id` of such a resource will thus often be the
unique id assigned to that object by the target system, and its other
attributes will correspond pretty directly with the system's data model,
using terminology familiar within that system.

`aws_instance`, `mysql_database` and `google_compute_disk` are all examples
of first-class resources, and map directly to concepts within their
respective systems.

## Connecting Resources

Connecting resources represent relationships between entities in a system.

This is most common when a "many-to-many" relationship is present in the
system's object model, where we can use connecting resource instances to
each represent one connection between a set of objects. It can also be
used when an entity has a "set" attribute and individual members of that set
need to be managed separately.

Such a relationship tends not to exist as a true entity in a system, and thus
it may not have an obvious unique identifier assigned. In this case, the `id`
used within Terraform may be a combination of some or all of the resource
attributes.

Connecting resources often lack an "Update" implementation, instead preferring
to force a new resource for each change.

`aws_security_group_rule` is an example of a connecting resource. Security
group rules do not exist as a first-class entity in the EC2 data model, and
are instead modelled as a set attribute on each security group. The unique
id of a security group within Terraform is the combination of all of its
arguments.

## Read-only Resources

A *read-only resource* is one that reads an object, rather than creating and
managing a new object.

For read-only resources, the arguments represent parameters for a search
query and the exported attributes represent the attributes of the object
being retrieved.

The resource lifecycle is not a good fit for read-only resources, so it is
planned to introduce a new concept called "data sources" in a forthcoming
Terraform version. New plugin implementers may thus prefer to wait for that
feature to be ready, and then create a data source instead.

`atlas_artifact` and `terraform_remote_state` are examples of read-only
resources.

## Logical Resources

A *logical resource* is similar to a *read-only resource*, but rather than
retrieving data from an external system it instead runs some logic within
Terraform itself and generates a result locally.

A logical resource is conceptually like a *function* in mathematics: it takes
arguments from its configuration block and exports a result in its
attributes.

The resource lifecycle is not a good fit for logical resources, so it is
planned to introduce a new concept called "data sources" in a forthcoming
Terraform version. New plugin implementers may thus prefer
to wait for that feature to be ready, and then create a data source instead.

`template_file` and `tls_cert_request` are examples of logical resources.

## Versioned Resources

One example is entities that are versioned within the remote system.
Terraform's model has no first-class support for versioning, so we need to
decide on an appropriate mapping.

This is not an area that has been explored in great detail yet, but the
following sub-section describes one pattern that is appropriate for some
situations.

### Versioning as an implementation detail

In this pattern we hide the versioning within the provider implementation,
with each update quietly creating a new version and each read always reading
the latest version. Whenever Terraform makes an update, it always sets it live
immediately.

This approach mimics the standard Terraform workflow but eliminates any
benefits that the remote system's version mechanism might provide, like
rolling back to an earlier version; a "rollback" from Terraform would just
involve creating a new version that happens to exactly match an older one.

It may also prove confusing if users are also looking at the configuration
within the system's UI, since the UI model won't match well to the Terraform
model.

## Non-CRUD Operations

Services will sometimes have operations that do not fit well into the resource
model, because they cannot be translated into a subset of the
"create, read, update, delete" (CRUD) operations.

At present, such operations are difficult to model in Terraform. It may be
reasonable to model such an operation as a provisioner, but that path has not
yet been well-trodden and should be approached with care.
