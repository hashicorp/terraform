---
layout: "language"
page_title: "State: CLI Workspaces"
sidebar_current: "docs-state-workspaces"
description: |-
  "Workspace" is a deprecated name for having multiple states
---

# CLI Workspaces

From Terraform v0.10 through to Terraform v0.14 inclusive, Terraform CLI used
the term "workspaces" to refer to the idea of having
[multiple states associated with the same configuration](multiple.html).

We no longer use that terminology because it's confusing with
[Terraform Cloud's idea of workspaces](/docs/cloud/workspaces/), which is a
separate concept with a different meaning.

The various CLI commands for working with multiple states are now, from
v0.15 onwards, `terraform state` subcommands rather than `terraform workspace`
subcommands. Other mentions of "Workspace" in the product have similarly been
replaced by mentions of "state" instead.
See [the Terraform v0.15 upgrade guide](https://www.terraform.io/upgrade-guides/0-15.html)
for more information.
