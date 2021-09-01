---
layout: "docs"
page_title: "Manipulating State - Terraform CLI"
description: "State data tracks which real-world object corresponds to each resource. Inspect state, move or import resources, and more."
---

# Manipulating Terraform State

> **Hands-on:** Try the [Manage Resources in Terraform State](https://learn.hashicorp.com/tutorials/terraform/state-cli?utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS) tutorial on HashiCorp Learn.

Terraform uses [state data](/docs/language/state/index.html) to remember which
real-world object corresponds to each resource in the configuration;
this allows it to modify an existing object when its resource declaration
changes.

Terraform updates state automatically during plans and applies. However, it's
sometimes necessary to make deliberate adjustments to Terraform's state data,
usually to compensate for changes to the configuration or the real managed
infrastructure.

Terraform CLI supports several workflows for interacting with state:

- [Inspecting State](/docs/cli/state/inspect.html)
- [Forcing Re-creation (Tainting)](/docs/cli/state/taint.html)
- [Moving Resources](/docs/cli/state/move.html)
- Importing Pre-existing Resources (documented in the
  [Importing Infrastructure](/docs/cli/import/index.html) section)
- [Disaster Recovery](/docs/cli/state/recover.html)

~> **Important:** Modifying state data outside a normal plan or apply can cause
Terraform to lose track of managed resources, which might waste money, annoy
your colleagues, or even compromise the security of your operations. Make sure
to keep backups of your state data when modifying state out-of-band.
