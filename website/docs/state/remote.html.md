---
layout: "docs"
page_title: "State: Remote Storage"
sidebar_current: "docs-state-remote"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# Remote State

By default, Terraform stores state locally in a file named `terraform.tfstate`.
When working with Terraform in a team, use of a local file makes Terraform
usage complicated because each user must make sure they always have the latest
state data before running Terraform and make sure that nobody else runs
Terraform at the same time.

With _remote_ state, Terraform writes the state data to a remote data store,
which can then be shared between all members of a team. Terraform supports
storing state in [Terraform Cloud](https://www.hashicorp.com/products/terraform/),
[HashiCorp Consul](https://www.consul.io/), Amazon S3, and more.

Remote state is a feature of [backends](/docs/backends). Configuring and
using remote backends is easy and you can get started with remote state
quickly. If you then want to migrate back to using local state, backends make
that easy as well.

## Delegation and Teamwork

Remote state gives you more than just easier version control and
safer storage. It also allows you to delegate the
[outputs](/docs/configuration/outputs.html) to other teams. This allows
your infrastructure to be more easily broken down into components that
multiple teams can access.

Put another way, remote state also allows teams to share infrastructure
resources in a read-only way without relying on any additional configuration
store.

For example, a core infrastructure team can handle building the core
machines, networking, etc. and can expose some information to other
teams to run their own infrastructure. As a more specific example with AWS:
you can expose things such as VPC IDs, subnets, NAT instance IDs, etc. through
remote state and have other Terraform states consume that.

For example usage, see
[the `terraform_remote_state` data source](/docs/providers/terraform/d/remote_state.html).

While remote state is a convenient, built-in mechanism for sharing data
between configurations, it is also possible to use more general stores to
pass settings both to other configurations and to other consumers. For example,
if your environment has [HashiCorp Consul](https://www.consul.io/) then you
can have one Terraform configuration that writes to Consul using
[`consul_key_prefix`](/docs/providers/consul/r/key_prefix.html) and then
another that consumes those values using
[the `consul_keys` data source](/docs/providers/consul/d/keys.html).

## Locking and Teamwork

For fully-featured remote backends, Terraform can also use
[state locking](/docs/state/locking.html) to prevent concurrent runs of
Terraform against the same state.

[Terraform Cloud by HashiCorp](https://www.hashicorp.com/products/terraform/)
is a commercial offering that supports an even stronger locking concept that
can also detect attempts to create a new plan when an existing plan is already
awaiting approval, by queuing Terraform operations in a central location.
This allows teams to more easily coordinate and communicate about changes to
infrastructure.
