---
layout: "language"
page_title: "Migrating from the remote backend"
sidebar_current: "terraform-cloud-configuration"
description: "Migrating from the remote backend"
---

# Migrating from the remote backend

The [remote backend]((/docs/language/settings/backends/remote.html)) was used as the primary
implementation of Terraform Cloud's [CLI-driven run workflow](/docs/cloud/run/cli.html) for
Terraform versions 0.11.13 through 1.0.x. We recommend migrating to using the native `cloud`
integration for Terraform versions 1.1 or later, as it provides an improved user experience and
various enhancements.

## Migrating an existing working directory automatically

If you've already been doing Terraform runs in a local directory with the remote backend, Terraform
can migrate to using the `cloud` option automatically.

Given the following pre-existing configuration, using Terraform 1.1+:

```
terraform {
  backend "remote" {
    organization = "my-org"

    workspaces {
      prefix = "my-app-"
    }
  }
}
```

### Step 1: Replace the Backend configuration with the `cloud` option

Remove the `backend` block entirely, replacing it with an equivalent `cloud` block:

```
terraform {
  cloud {
    organization = "my-org"

    workspaces {
      tags = ["app"]
    }
  }
}
```

The `tags` value can be any number of tags appropriate for categorizing workspaces in your Terraform
Cloud organization.

### Step 2: Run `terraform init`

Run `terraform init -migrate-state`. A prompt will appear, explaining the migration path:

```
» terraform init -migrate-state

Initializing Terraform Cloud...
Backend configuration changed!

Terraform has detected that the configuration specified for the backend
has changed. Terraform will now check for existing state in the backends.

Terraform detected that the backend type changed from "remote" to Terraform Cloud.

Do you wish to proceed?
  When migrating from the 'remote' backend to Terraform's native integration
  with Terraform Cloud, Terraform will automatically create or use existing
  workspaces based on the previous backend configuration's 'prefix' value.

  When the migration is complete, workspace names in Terraform will match the
  fully qualified Terraform Cloud workspace name. If necessary, the workspace
  tags configured in the 'cloud' option block will be added to the associated
  Terraform Cloud workspaces.

  Enter "yes" to proceed or "no" to cancel.

  Enter a value:
```

Entering "yes" should result in output similar to the following, presenting your workspaces as they
are in Terraform Cloud:

```
Migration complete! Your workspaces are as follows:
  my-app-prod
* my-app-staging


Initializing provider plugins...

Terraform Cloud has been successfully initialized!

You may now begin working with Terraform Cloud. Try running "terraform plan" to
see any changes that are required for your infrastructure.

If you ever set or change modules or Terraform Settings, run "terraform init"
again to reinitialize your working directory.
```

## Manually reconfiguring

If you were already using Terraform Cloud with the remote backend, you may also 'migrate' manually
by adding any number of workspace tags to the Terraform Cloud workspaces previously used by the
remote backend, replacing the `backend` block with a `cloud` block containing those workspace tag
values, and running `terraform init -reconfigure`. This will reconfigure your local working
directory to use the native integration with the same workspaces as was configured previously.
