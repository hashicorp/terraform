---
layout: "docs"
page_title: "State"
sidebar_current: "docs-state-purpose"
description: |-
  Terraform must store state about your managed infrastructure and configuration. This state is used by Terraform to map real world resources to your configuration, keep track of metadata, and to improve performance for large infrastructures.
---

# Purpose of Terraform State

State is a necessary requirement for Terraform to function. It is often
asked if it is possible for Terraform to work without state, or for Terraform
to not use state and just inspect cloud resources on every run. This page
will help explain why Terraform state is required.

As you'll see from the reasons below, state is required. And in the scenarios
where Terraform may be able to get away without state, doing so would require
shifting massive amounts of complexity from one place (state) to another place
(the replacement concept).

## Mapping to the Real World

Terraform requires some sort of database to map Terraform config to the real
world. When you have a resource `resource "aws_instance" "foo"` in your
configuration, Terraform uses this map to know that instance `i-abcd1234`
is represented by that resource.

For some providers like AWS, Terraform could theoretically use something like
AWS tags. Early prototypes of Terraform actually had no state files and used
this method. However, we quickly ran into problems. The first major issue was
a simple one: not all resources support tags, and not all cloud providers
support tags.

Therefore, for mapping configuration to resources in the real world,
Terraform uses its own state structure.

## Metadata

Alongside the mappings between resources and remote objects, Terraform must
also track metadata such as resource dependencies.

Terraform typically uses the configuration to determine dependency order.
However, when you delete a resource from a Terraform configuration, Terraform
must know how to delete that resource. Terraform can see that a mapping exists
for a resource not in your configuration and plan to destroy. However, since
the configuration no longer exists, the order cannot be determined from the
configuration alone.

To ensure correct operation, Terraform retains a copy of the most recent set
of dependencies within the state. Now Terraform can still determine the correct
order for destruction from the state when you delete one or more items from
the configuration.

One way to avoid this would be for Terraform to know a required ordering
between resource types. For example, Terraform could know that servers must be
deleted before the subnets they are a part of. The complexity for this approach
quickly explodes, however: in addition to Terraform having to understand the
ordering semantics of every resource for every cloud, Terraform must also
understand the ordering _across providers_.

Terraform also stores other metadata for similar reasons, such as a pointer
to the provider configuration that was most recently used with the resource
in situations where multiple aliased providers are present.

## Performance

In addition to basic mapping, Terraform stores a cache of the attribute
values for all resources in the state. This is the most optional feature of
Terraform state and is done only as a performance improvement.

When running a `terraform plan`, Terraform must know the current state of
resources in order to effectively determine the changes that it needs to make
to reach your desired configuration.

For small infrastructures, Terraform can query your providers and sync the
latest attributes from all your resources. This is the default behavior
of Terraform: for every plan and apply, Terraform will sync all resources in
your state.

For larger infrastructures, querying every resource is too slow. Many cloud
providers do not provide APIs to query multiple resources at once, and the
round trip time for each resource is hundreds of milliseconds. On top of this,
cloud providers almost always have API rate limiting so Terraform can only
request a certain number of resources in a period of time. Larger users
of Terraform make heavy use of the `-refresh=false` flag as well as the
`-target` flag in order to work around this. In these scenarios, the cached
state is treated as the record of truth.

## Syncing

In the default configuration, Terraform stores the state in a file in the
current working directory where Terraform was run. This is okay for getting
started, but when using Terraform in a team it is important for everyone
to be working with the same state so that operations will be applied to the
same remote objects.

[Remote state](/docs/state/remote.html) is the recommended solution
to this problem. With a fully-featured state backend, Terraform can use
remote locking as a measure to avoid two or more different users accidentally
running Terraform at the same time, and thus ensure that each Terraform run
begins with the most recent updated state.
