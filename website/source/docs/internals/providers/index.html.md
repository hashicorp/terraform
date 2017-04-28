---
layout: "docs"
page_title: "Writing a Provider"
sidebar_current: "docs-internals-provider-guide"
description: |-
  Information on writing a provider for Terraform.
---

# Writing a Terraform Provider

There are two high-level sections of the Terraform codebase:

![Relationship between core and
providers](docs/core-provider-diagram-labeled.jpg)

1. The core section turns state and config files into a list of resources that
   require action. 
2. The providers handle the actual creation, updating, and destruction of those
   resources.

By dividing things like this, it is much easier to contribute new functionality
or new APIs to Terraform: contributtors can essentially just trust the core
section to do its job, and only worry about their resources.

One way to think of the separation is that the core tells the providers what to
do, and the providers make that change against the APIs.

To hook into this behaviour, Terraform has a provider framework. This guide
aims to document that framework, and help guide thinking and understanding
around it.
