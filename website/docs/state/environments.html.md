---
layout: "docs"
page_title: "State: Environments"
sidebar_current: "docs-state-env"
description: |-
  Legacy terminology for "Workspaces".
---

# State Environments

The term _state environment_, or just _environment_, was used within the
Terraform 0.9 releases to refer to the idea of having multiple distinct,
named states associated with a single configuration directory.

After this concept was implemented, we received feedback that this terminology
caused confusion due to other uses of the word "environment", both within
Terraform itself and within organizations using Terraform.

As of 0.10, the preferred term is "workspace". For more information on
workspaces, see [the main Workspaces page](/docs/state/workspaces.html).
