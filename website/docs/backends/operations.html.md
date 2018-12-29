---
layout: "docs"
page_title: "Backends: Remote Operations (plan, apply, etc.)"
sidebar_current: "docs-backends-operations"
description: |-
  Some backends support the ability to run operations (`refresh`, `plan`, `apply`, etc.) remotely. Terraform will continue to look and behave as if they're running locally while they in fact run on a remote machine.
---

# Remote Operations (plan, apply, etc.)

Most backends run all operations on the local system — although Terraform stores
its state remotely with these backends, it still executes its logic locally and
makes API requests directly from the system where it was invoked.

This is simple to understand and work with, but when many people are
collaborating on the same Terraform configurations, it requires everyone's
execution environment to be similar. This includes sharing access to
infrastructure provider credentials, keeping Terraform versions in sync,
keeping Terraform variables in sync, and installing any extra software required
by Terraform providers. This becomes more burdensome as teams get larger.

Some backends can run operations (`plan`, `apply`, etc.) on a remote machine,
while appearing to execute locally. This enables a more consistent execution
environment and more powerful access controls, without disrupting workflows
for users who are already comfortable with running Terraform.

Currently, [the `remote` backend](./types/remote.html) is the only backend to
support remote operations, and [Terraform Enterprise](/docs/enterprise/index.html)
is the only remote execution environment that supports it. For more information, see:

- [The `remote` backend](./types/remote.html)
- [Terraform Enterprise's CLI-driven run workflow](/docs/enterprise/run/cli.html)
