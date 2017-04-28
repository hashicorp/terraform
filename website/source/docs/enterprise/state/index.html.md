---
layout: "enterprise"
page_title: "State - Terraform Enterprise"
sidebar_current: "docs-enterprise-state"
description: |-
  Terraform stores the state of your managed infrastructure from the last time Terraform was run. This section is about states.
---

# State

Terraform Enterprise stores the state of your managed infrastructure from the
last time Terraform was run. The state is stored remotely, which works better in a
team environment, allowing you to store, version and collaborate on state.

Remote state gives you more than just easier version control and safer storage.
It also allows you to delegate the outputs to other teams. This allows your
infrastructure to be more easily broken down into components that multiple teams
can access.

Remote state is automatically updated when you run [`apply`](/docs/commands/apply.html)
locally. It is also updated when an `apply` is executed in a [Terraform Enterprise
Run](/docs/enterprise/runs/index.html).

Read [more about remote state](/docs/state/remote.html).
