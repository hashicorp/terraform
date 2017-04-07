---
layout: "enterprise"
page_title: "State - Terraform Enterprise"
sidebar_current: "docs-enterprise-state"
description: |-
  Terraform stores the state of your managed infrastructure from the last time Terraform was run. This section is about states.
---

# State

Terraform stores the state of your managed infrastructure from the last time
Terraform was run. By default this state is stored in a local file named
`terraform.tfstate`, but it can also be stored remotely, which works better in a
team environment.

Terraform Enterprise is a remote state provider, allowing you to store, version
and collaborate on states.

Remote state gives you more than just easier version control and safer storage.
It also allows you to delegate the outputs to other teams. This allows your
infrastructure to be more easily broken down into components that multiple teams
can access.

Read [more about remote state](https://www.terraform.io/docs/state/remote.html).
