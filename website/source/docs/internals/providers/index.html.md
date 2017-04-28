---
layout: "docs"
page_title: "Writing a Provider"
sidebar_current: "docs-internals-provider-guide"
description: |-
  Information on writing a provider for Terraform.
---

# Writing a Terraform Provider

There are two high-level sections of the Terraform codebase: the core section
handles all the graphs, diffing, and essentially turning state and config files
into a list of resources that need to be created, updated, or destroyed; the
providers handle the actual creation, updating, and destruction of those
resources. By dividing things like this, it is much easier to contribute new
functionality or new APIs to Terraform: you can essentially just trust the core
section to do its job, and only worry about your resource.

One way to think of the separation is that the core tells the providers what to
do, and the providers make that change against the APIs.

To hook into this behaviour, a provider framework has been built into
Terraform. This guide aims to document that framework, and help guide thinking
and understanding around it.
