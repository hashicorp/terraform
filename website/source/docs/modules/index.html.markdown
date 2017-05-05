---
layout: "docs"
page_title: "Modules"
sidebar_current: "docs-modules"
description: |-
  Modules in Terraform are self-contained packages of Terraform configurations that are managed as a group. Modules are used to create reusable components in Terraform as well as for basic code organization.
---

# Modules

Modules in Terraform are self-contained packages of Terraform configurations
that are managed as a group. Modules are used to create reusable components
in Terraform as well as for basic code organization.

Modules are very easy to both use and create. Depending on what you're
looking to do first, use the navigation on the left to dive into how
modules work.

## Definitions
**Root module**
That is the current working directory when you run [`terraform apply`](/docs/commands/apply.html) or [`get`](/docs/commands/get.html), holding the Terraform [configuration files](/docs/configuration/index.html).
It is itself a valid module.
