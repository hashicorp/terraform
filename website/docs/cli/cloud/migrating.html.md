---
layout: "docs"
page_title: "Initializing and Migrating to Terraform Cloud - Terraform CLI"
---

# Initializing and Migrating

After [configuring Terraform Cloud settings](/docs/cli/cloud/settings.html) for a working directory, you must run `terraform init` to finish setting up. When running this command, Terraform may guide you through an interactive process where you may choose whether or not to migrate state from any existing workspaces.

There are three potential scenarios:

1. **Fresh working directory:** If the working directory has no existing Terraform state, no migrations will occur. You can start using Terraform with Terraform Cloud right away, creating workspaces and starting runs.
2. **State backend or local state:** If the working directory already has state data in one or more workspaces (using either local state or a [state backend](/docs/language/settings/backends/index.html)), Terraform will ask if you're like to migrate that state to new Terraform Cloud workspaces. You will need permission to manage workspaces in the destination Terraform Cloud organization. You may also be prompted to rename the workspaces being migrated, to better distinguish them within a Terraform Cloud organization.
3. **`remote` backend:** If the working directory was already connected to Terraform Cloud with the `remote` backend, Terraform can continue using the same Terraform Cloud workspaces. The local names shown for those workspaces will change to match their remote names.

The rest of this page briefly describes cases 2 and 3.

## Migrating from Local State or Other Backends

If the working directory already has state data available (using either local state or a [state
backend](/docs/language/settings/backends/index.html)), Terraform will ask your approval to migrate
that state to Terraform Cloud. This process is interactive and self-documenting, and resembles
moving between state backends.

Terraform may also prompt you to rename your workspaces during the migration, to either give a name to
the unnamed `default` workspace (Terraform Cloud requires all workspaces to have a name) or give
your workspace names more contextual information. Unlike Terraform CLI-only workspaces, which represent
multiple environments associated with the same configuration (e.g. production, staging, development),
Terraform Cloud workspaces can represent totally independent configurations, and must have unique names within the Terraform Cloud organization.

Because of this, Terraform will prompt you to rename the working directory's workspaces
according to a pattern relative to their existing names, which can indicate the fact that these specific workspaces share configuration. A typical strategy to start with is
`<COMPONENT>-<ENVIRONMENT>-<REGION>` (e.g.  `networking-prod-us-east`,
`networking-staging-us-east`). For more information on workspace naming, see [Workspace
Naming](/docs/cloud/workspaces/naming.html) in the Terraform Cloud documentation.

## Migrating from the `remote` Backend

If the working directory was already connected to Terraform Cloud with the `remote` backend, Terraform can continue using the same Terraform Cloud workspaces. The local names shown for those workspaces will change to match their remote names.

The [`remote` backend](/docs/language/settings/backends/remote.html) was the primary implementation of Terraform Cloud's [CLI-driven run workflow](/docs/cloud/run/cli.html) for Terraform versions 0.11.13 through 1.0.x. We recommend using the native `cloud` integration for Terraform versions 1.1 or later, as it provides an improved user experience and various enhancements.

### Block Replacement

When switching from the `remote` backend to a `cloud` block, Terraform will continue using the same
set of Terraform Cloud workspaces. Replace your `backend "remote"` block with an equivalent `cloud`
block:

- If you were using a single workspace with the `name` argument, change the block
  label to `cloud`.

    ```diff
     terraform {
    -  backend "remote" {
    +  cloud {
         organization = "my-org"

         workspaces {
           name = "my-app-prod"
         }
       }
     }
    ```

- If you were using multiple workspaces via the `prefix` argument, replace it
  with a `cloud` block that uses the `tags` argument. You may specify any number of tags to
  distinguish the workspaces for your working directory, but a good starting point may be to use
  whatever the prefix was before.

    The tags you configure do not need to be present on the existing workspaces. When you initialize, Terraform will add the specified tags to the workspaces if necessary.

    ```diff
     terraform {
    -  backend "remote" {
    +  cloud {
         organization = "my-org"

         workspaces {
    -      prefix = "my-app-"
    +      tags = ["app:mine"]
         }
       }
     }
    ```
