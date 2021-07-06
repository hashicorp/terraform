---
layout: "docs"
page_title: "Command: refresh"
sidebar_current: "docs-commands-refresh"
description: |-
  The `terraform refresh` command reads the current settings from all managed
  remote objects and updates the Terraform state to match.
---

# Command: refresh

> **Hands-on:** Try the [Use Refresh-Only Mode to Sync Terraform State](https://learn.hashicorp.com/tutorials/terraform/refresh) tutorial on HashiCorp Learn.

The `terraform refresh` command reads the current settings from all managed
remote objects and updates the Terraform state to match.

~> *Warning:* This command is deprecated, because its default behavior is
unsafe if you have misconfigured credentials for any of your providers.
See below for more information and recommended alternatives.

This won't modify your real remote objects, but it will modify the
[the Terraform state](/docs/language/state/).

You shouldn't typically need to use this command, because Terraform
automatically performs the same refreshing actions as a part of creating
a plan in both the
[`terraform plan`](./plan.html)
and
[`terraform apply`](./apply.html)
commands. This command is here primarily for backward compatibility, but
we don't recommend using it because it provides no opportunity to review
the effects of the operation before updating the state.

## Usage

Usage: `terraform refresh [options]`

This command is effectively an alias for the following command:

```
terraform apply -refresh-only -auto-approve
```

Consequently, it supports all of the same options as
[`terraform apply`](./apply.html) except that it does not accept a saved
plan file, it doesn't allow selecting a planning mode other than "refresh only",
and `-auto-approve` is always enabled.

Automatically applying the effect of a refresh is risky, because if you have
misconfigured credentials for one or more providers then the provider may
be misled into thinking that all of the managed objects have been deleted,
and thus remove all of the tracked objects without any confirmation prompt.

Instead, we recommend using the following command in order to get the same
effect but with the opportunity to review the the changes that Terraform has
detected before committing them to the state:

```
terraform apply -refresh-only
```

This alternative command will present an interactive prompt for you to confirm
the detected changes.

The `-refresh-only` option for `terraform plan` and `terraform apply` was
introduced in Terraform v0.15.4. For prior versions you must use
`terraform refresh` directly if you need this behavior, while taking into
account the warnings above. Wherever possible, avoid using `terraform refresh`
explicitly and instead rely on Terraform's behavior of automatically refreshing
existing objects as part of creating a normal plan.
