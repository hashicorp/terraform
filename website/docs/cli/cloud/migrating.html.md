---
layout: "docs"
page_title: "Initializing and Migrating to Terraform Cloud - Terraform CLI"
---

# Initializing and Migrating

After [configuring Terraform Cloud settings](/docs/cli/cloud/settings.html) for a working directory, you must run `terraform init` to finish setting up.

-> **Note:** When initializing Terraform Cloud, `terraform init`'s `-migrate-state` and `-reconfigure` options aren't valid.

There are three main paths for what happens when initializing Terraform Cloud support:

1. **Fresh working directory:** If the working directory has no existing Terraform state, no extra work happens during initialization. You can start using Terraform with Terraform Cloud right away.
2. **`remote` backend:** If the working directory was already connected to Terraform Cloud with the `remote` backend, Terraform can continue using the same Terraform Cloud workspaces. The local names shown for those workspaces will change to match their remote names.
3. **State backend or local state:** If the working directory already has state data in one or more workspaces (using either local state or a [state backend](/docs/language/settings/backends/index.html)), Terraform will try to migrate that state to new Terraform Cloud workspaces. You will need permission to manage workspaces in the destination Terraform Cloud organization, and you might need to rename your workspaces by adding a prefix and/or suffix. Terraform will interactively ask you what to do.

The rest of this page briefly describes cases 2 and 3.

## Migrating from the `remote` Backend

The [`remote` backend]((/docs/language/settings/backends/remote.html)) was the primary
implementation of Terraform Cloud's [CLI-driven run workflow](/docs/cloud/run/cli.html) for
Terraform versions 0.11.13 through 1.0.x. We recommend using the native `cloud`
integration for Terraform versions 1.1 or later, as it provides an improved user experience and
various enhancements.

### Block Replacement

When switching from the `remote` backend to a `cloud` block, Terraform will try
to continue using the same set of Terraform Cloud workspaces. To take advantage
of this, you must replace your `backend "remote"` block with an equivalent
`cloud` block:

- If you were using a single workspace via the `name` argument, change the block
  label to `cloud`. The inner arguments are unchanged.

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
  with a `cloud` block that uses the `tags` argument.

    The existing workspaces don't need to already have these tags — when you
    initialize, Terraform will attempt to add the specified tags to them.

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

### Limitations

To take advantage of automatic tagging of existing workspaces, you must have
permission to change tags for every affected workspace. Thus, switching to the
`cloud` integration works best when someone with permission to manage workspaces
for your organization performs the migration first. When other users
re-initialize later, the workspaces will already be tagged appropriately and
tagging permissions won't be required.

### Workspace Names

After switching to the `cloud` integration, your local workspace names will
change to match their remote names.

The `remote` backend was designed to use two different names for each workspace:
a short name for local use, and a longer name (with a prefix added) for remote
use. For example, a remote workspace named `app-prod` would be called `prod`
locally if the backend was configured with a prefix of `app-`.

This proved confusing in practice, so the `cloud` integration doesn't do that.
Instead, workspaces use the same names remotely and locally.

## Migrating from Local State or Other Backends

If you _weren't_ using the `remote` backend and the working directory already
has state data available (using either local state or a
[state backend](/docs/language/settings/backends/index.html)), Terraform will
ask your approval to migrate that state to Terraform Cloud. This process is
interactive and self-documenting, and resembles moving between state backends.

There's one unique step in migrating to Terraform Cloud: if you configured your
`cloud` block using the `tags` strategy, you might need to rename your
workspaces.

This happens because Terraform Cloud has a global view of your infrastructure
(and expects workspaces to have unique names), but Terraform CLI only knows
about one working directory at a time (and doesn't mind if you re-use the same
workspace names in different directories). If you try to migrate a workspace
whose name is already claimed for a different purpose, bad things can happen.

To help you avoid this, Terraform will offer you a chance to add a prefix and/or
suffix to your workspace names when migrating them. You'll need to provide a
pattern that contains an asterisk (`*`) in the place where the current workspace
name should be inserted. For example, to rename the workspaces `prod` and
`staging` to `my-app-prod-us-west` and `my-app-staging-us-west`, you would
provide a rename pattern of `my-app-*-us-west`.

If your workspace names are already unique and descriptive, you can use a
pattern of `*` (a lone asterisk) to migrate them with unchanged names.
