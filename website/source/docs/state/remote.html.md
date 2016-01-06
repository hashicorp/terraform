---
layout: "docs"
page_title: "Remote State"
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
supports storing state in [Atlas](https://atlas.hashicorp.com),
[Consul](https://www.consul.io), S3, and more.

You can begin using remote state from the beginning with flags to the
[init](/docs/commands/init.html) command, or you can migrate an existing
local state to remote state using the
[remote config](/docs/commands/remote-config.html) command. You can also
use the remote config to disable remote state and move back to local
state.

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

For example usage see the [terraform_remote_state](/docs/providers/terraform/r/remote_state.html) resource.

## Locking and Teamwork

Remote state currently **does not** lock regions of your infrastructure
to allow parallel modification using Terraform. Therefore, you must still
collaborate with teammates to safely run Terraform.

[Atlas by HashiCorp](https://atlas.hashicorp.com) is a commercial offering
that does safely allow parallel Terraform runs and handles infrastructure
locking for you.

In the future, we'd like to extend the remote state system to allow some
minimal locking functionality, but it is a difficult problem without a
central system that we currently aren't focused on solving.
