---
layout: "docs"
page_title: "Backends: Init"
sidebar_current: "docs-backends-init"
description: |-
  Terraform must initialize any configured backend before use. This can be done by simply running `terraform init`.
---

# Backend Initialization

Terraform must initialize any configured backend before use. This can be
done by simply running `terraform init`.

The `terraform init` command should be run by any member of your team on
any Terraform configuration as a first step. It is safe to execute multiple
times and performs all the setup actions required for a Terraform environment,
including initializing the backend.

The `init` command must be called:

  * On any new environment that configures a backend
  * On any change of the backend configuration (including type of backend)
  * On removing backend configuration completely

You don't need to remember these exact cases. Terraform will detect when
initialization is required and error in that situation. Terraform doesn't
auto-initialize because it may require additional information from the user,
perform state migrations, etc.

The `init` command will do more than just initialize the backend. Please see
the [init documentation](/docs/commands/init.html) for more information.
