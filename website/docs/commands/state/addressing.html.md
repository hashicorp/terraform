---
layout: "commands-state"
page_title: "Command: state resource addressing"
sidebar_current: "docs-commands-state-address"
description: |-
  The `terraform state` command is used for advanced state management.
---

# Resource Addressing

The `terraform state` subcommands make heavy use of resource addressing
for targeting and filtering specific resources and modules within the state.

Resource addressing is a common feature of Terraform that is used in
multiple locations. For example, resource addressing syntax is also used for
the `-target` flag for apply and plan commands.

Because resource addressing is unified across Terraform, it is documented
in a single place rather than duplicating it in multiple locations. You
can find the [resource addressing documentation here](/docs/internals/resource-addressing.html).
