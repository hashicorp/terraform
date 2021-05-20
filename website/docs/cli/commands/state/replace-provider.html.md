---
layout: "docs"
page_title: "Command: state replace-provider"
sidebar_current: "docs-commands-state-sub-replace-provider"
description: |-
  The `terraform state replace-provider` command replaces the provider for resources in the Terraform state.
---

# Command: state replace-provider

The `terraform state replace-provider` command is used to replace the provider
for resources in a [Terraform state](/docs/language/state/index.html).

## Usage

Usage: `terraform state replace-provider [options] FROM_PROVIDER_FQN TO_PROVIDER_FQN`

This command will update all resources using the "from" provider, setting the
provider to the specified "to" provider. This allows changing the source of a
provider which currently has resources in state.

This command will output a backup copy of the state prior to saving any
changes. The backup cannot be disabled. Due to the destructive nature
of this command, backups are required.

This command also accepts the following options:

* `-auto-approve` - Skip interactive approval.

* `-lock=false` - Don't hold a state lock during the operation. This is
   dangerous if others might concurrently run commands against the same
   workspace.

* `-lock-timeout=0s` - Duration to retry a state lock.

For configurations using
[the `remote` backend](/docs/language/settings/backends/remote.html)
only, `terraform state replace-provider`
also accepts the option
[`-ignore-remote-version`](/docs/language/settings/backends/remote.html#command-line-arguments).

For configurations using
[the `local` state rm](/docs/language/settings/backends/local.html) only,
`terraform state replace-provider` also accepts the legacy options
[`-state`, `-state-out`, and `-backup`](/docs/language/settings/backends/local.html#command-line-arguments).


## Example

The example below replaces the `hashicorp/aws` provider with a fork by `acme`, hosted at a private registry at `registry.acme.corp`:

```shell
$ terraform state replace-provider hashicorp/aws registry.acme.corp/acme/aws
```
