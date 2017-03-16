---
layout: "docs"
page_title: "Backends: Operations (refresh, plan, apply, etc.)"
sidebar_current: "docs-backends-ops"
description: |-
  Some backends support the ability to run operations (`refresh`, `plan`, `apply`, etc.) remotely. Terraform will continue to look and behave as if they're running locally while they in fact run on a remote machine.
---

# Operations (plan, apply, etc.)

Some backends support the ability to run operations (`refresh`, `plan`, `apply`,
etc.) remotely. Terraform will continue to look and behave as if they're
running locally while they in fact run on a remote machine.

Backends should not modify the actual infrastructure change behavior of
these commands. They will only modify how they're invoked.

At the time of writing, no backends support this. This shouldn't be linked
in the sidebar yet!
