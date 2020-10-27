---
layout: "docs"
page_title: "Command: state replace-provider"
sidebar_current: "docs-commands-state-sub-replace-provider"
description: |-
  The `terraform state replace-provider` command replaces the provider for resources in the Terraform state.
---

# Command: state replace-provider

The `terraform state replace-provider` command is used to replace the provider
for resources in a [Terraform state](/docs/state/index.html).

## Usage

Usage: `terraform state replace-provider [options] FROM_PROVIDER_FQN TO_PROVIDER_FQN`

This command will update all resources using the "from" provider, setting the
provider to the specified "to" provider. This allows changing the source of a
provider which currently has resources in state.

This command will output a backup copy of the state prior to saving any
changes. The backup cannot be disabled. Due to the destructive nature
of this command, backups are required.

The command-line flags are all optional. The list of available flags are:

* `-auto-approve` - Skip interactive approval.

* `-backup=path` - Path where Terraform should write the backup for the
  original state. This can't be disabled. If not set, Terraform will write it
  to the same path as the statefile with a ".backup" extension.

* `-lock=true`- Lock the state files when locking is supported.

* `-lock-timeout=0s` - Duration to retry a state lock.

* `-state=path` - Path to the source state file to read from. Defaults to the
  configured backend, or "terraform.tfstate".

## Example

The example below replaces the `hashicorp/aws` provider with a fork by `acme`, hosted at a private registry at `registry.acme.corp`:

```shell
$ terraform state replace-provider hashicorp/aws registry.acme.corp/acme/aws
```
