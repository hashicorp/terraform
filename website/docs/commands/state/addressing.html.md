---
layout: "commands-state"
page_title: "Command: state resource addressing"
sidebar_current: "docs-commands-state-address"
description: |-
  The `terraform state` command is used for advanced state management.
---

# Resource Addressing

The `terraform state` subcommands use
[standard address syntax](/docs/internals/resource-addressing.html) to refer
to individual resources, resource instances, and modules. This is the same
syntax used for the `-target` option to the `apply` and `plan` commands.

Most state commands allow referring to individual resource instances, whole
resources (which may have multiple instances if `count` or `for_each` is used),
or even whole modules.

For more information on the syntax, see [Resource Addressing](/docs/internals/resource-addressing.html).
