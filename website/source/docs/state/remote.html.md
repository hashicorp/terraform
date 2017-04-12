---
layout: "docs"
page_title: "State: Remote Storage"
sidebar_current: "docs-state-remote"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# Remote State

By default, Terraform stores state locally in a file named "terraform.tfstate".
Because this file must exist, it makes working with Terraform in a team
complicated since it is a frequent source of merge conflicts. Remote state
helps alleviate these issues.

With remote state, Terraform stores the state in a remote store. Terraform
supports storing state in [Terraform Enterprise](https://www.hashicorp.com/products/terraform/),
[Consul](https://www.consul.io), S3, and more.

Remote state is a feature of [backends](/docs/backends). Configuring and
using backends is easy and you can get started with remote state quickly.
If you want to migrate back to using local state, backends make that
easy as well.

## Delegation and Teamwork

Remote state gives you more than just easier version control and
safer storage. It also allows you to delegate the
[outputs](/docs/configuration/outputs.html) to other teams. This allows
your infrastructure to be more easily broken down into components that
multiple teams can access.

Put another way, remote state also allows teams to share infrastructure
resources in a read-only way.

For example, a core infrastructure team can handle building the core
machines, networking, etc. and can expose some information to other
teams to run their own infrastructure. As a more specific example with AWS:
you can expose things such as VPC IDs, subnets, NAT instance IDs, etc. through
remote state and have other Terraform states consume that.

For example usage see the
[terraform_remote_state](/docs/providers/terraform/d/remote_state.html) data source.

## Locking and Teamwork

Terraform will automatically lock state depending on the
[backend](/docs/backends) used. Please see the full page dedicated
to [state locking](/docs/state/locking.html).

[Terraform Enterprise by HashiCorp](https://www.hashicorp.com/products/terraform/) is a commercial offering
that in addition to locking supports remote operations that allow you to
safely queue Terraform operations in a central location. This enables
teams to safely modify infrastructure concurrently.
