---
layout: "docs"
page_title: "Command: push"
sidebar_current: "docs-commands-push"
description: |-
  The `terraform push` command is used to upload the Terraform configuration to HashiCorp's Terraform Enterprise service for automatically managing your infrastructure in the cloud.
---

# Command: push

~> **Important:** The `terraform push` command is deprecated, and only works with the legacy version of Terraform Enterprise. In the current version of Terraform Enterprise, you can upload configurations using the API. See [the docs about API-driven runs](/docs/enterprise/run/api.html) for more details.

The `terraform push` command uploads your Terraform configuration to
be managed by HashiCorp's [Terraform Enterprise](https://www.hashicorp.com/products/terraform/).
Terraform Enterprise can automatically run
Terraform for you, save all state transitions, save plans,
and keep a history of all Terraform runs.

This makes it significantly easier to use Terraform as a team: team
members modify the Terraform configurations locally and continue to
use normal version control. When the Terraform configurations are ready
to be run, they are pushed to Terraform Enterprise, and any member of your team can
run Terraform with the push of a button.

Terraform Enterprise can also be used to set ACLs on who can run Terraform, and a
future update of Terraform Enterprise will allow parallel Terraform runs and automatically
perform infrastructure locking so only one run is modifying the same
infrastructure at a time.

~> When using this command, it is important to match your local Terraform version with
   the version selected for the target workspace in Terraform Enterprise, since
   otherwise the uploaded configuration archive may not be compatible with the remote
   Terraform process.

## Usage

Usage: `terraform push [options] [path]`

The `path` argument is the same as for the
[apply](/docs/commands/apply.html) command.

The command-line flags are all optional. The list of available flags are:

* `-atlas-address=<url>` - An alternate address to an instance.
  Defaults to `https://atlas.hashicorp.com`.

* `-upload-modules=true` - If true (default), then the
  [modules](/docs/modules/index.html)
  being used are all locked at their current checkout and uploaded
  completely. This prevents Terraform Enterprise from running `terraform get`
  for you.

* `-name=<name>` - Name of the infrastructure configuration in Terraform Enterprise.
  The format of this is: "username/name" so that you can upload
  configurations not just to your account but to other accounts and
  organizations. This setting can also be set in the configuration
  in the
  [Terraform Enterprise section](/docs/configuration/terraform-enterprise.html).

* `-no-color` - Disables output with coloring


* `-overwrite=foo` - Marks a specific variable to be updated.
  Normally, if a variable is already set Terraform will not
  send the local value (even if it is different). This forces it to
  send the local value to Terraform Enterprise. This flag can be repeated multiple times.

* `-token=<token>` - Terraform Enterprise API token to use to authorize the upload.
  If blank or unspecified, the `ATLAS_TOKEN` environment variable
  will be used.

* `-var='foo=bar'` - Set the value of a variable for the Terraform configuration.

* `-var-file=foo` - Set the value of variables using a variable file. This flag
  can be used multiple times.


* `-vcs=true` - If true (default), then Terraform will detect if a VCS
  is in use, such as Git, and will only upload files that are committed to
  version control. If no version control system is detected, Terraform will
  upload all files in `path` (parameter to the command).

## Packaged Files

The files that are uploaded and packaged with a `push` are all the
files in the `path` given as the parameter to the command, recursively.
By default (unless `-vcs=false` is specified), Terraform will automatically
detect when a VCS such as Git is being used, and in that case will only
upload the files that are committed. Because of this built-in intelligence,
you don't have to worry about excluding folders such as ".git" or ".hg" usually.

If Terraform doesn't detect a VCS, it will upload all files.

The reason Terraform uploads all of these files is because Terraform
cannot know what is and isn't being used for provisioning, so it uploads
all the files to be safe. To exclude certain files, specify the `-exclude`
flag when pushing, or specify the `exclude` parameter in the
[Terraform Enterprise configuration section](/docs/configuration/terraform-enterprise.html).

Terraform also includes in the package all of the modules that were installed
during the most recent `terraform init` or `terraform get` command. Since the
details of how modules are cached in the filesystem vary between Terraform versions,
it is important to use the same version of Terraform both locally (when running
`terraform init` and then `terraform push`) and in your remote Terraform Enterprise
workspace.

## Terraform Variables

When you `push`, Terraform will automatically set the local values of
your Terraform variables on Terraform Enterprise. The values are only set if they
don't already exist. If you want to force push a certain
variable value to update it, use the `-overwrite` flag.

All the variable values stored are encrypted and secured
using [Vault](https://www.vaultproject.io). We blogged about the
[architecture of our secure storage system](https://www.hashicorp.com/blog/how-atlas-uses-vault-for-managing-secrets.html) if you want more detail.

The variable values can be updated using the `-overwrite` flag or via
the [Terraform Enterprise website](https://www.hashicorp.com/products/terraform/). An example of updating
just a single variable `foo` is shown below:

```shell
$ terraform push -var 'foo=bar' -overwrite foo
```

Both the `-var` and `-overwrite` flag are required. The `-var` flag
sets the value locally (the exact same process as commands such as apply
or plan), and the `-overwrite` flag tells the push command to update Terraform Enterprise.

## Remote State Requirement

~> **Important:** This section only refers to the legacy version of Terraform Enterprise. The current version of Terraform Enterprise always manages its own state, and does not support arbitrary remote state backends.

`terraform push` requires that
[remote state](/docs/state/remote.html)
is enabled. The reasoning for this is simple: `terraform push` sends your
configuration to be managed remotely. For it to keep the state in sync
and for you to be able to easily access that state, remote state must
be enabled instead of juggling local files.

While `terraform push` sends your configuration to be managed by Terraform Enterprise,
the remote state backend _does not_ have to be Terraform Enterprise. It can be anything
as long as it is accessible by the public internet, since Terraform Enterprise will need
to be able to communicate to it.

**Warning:** The credentials for accessing the remote state will be
sent up to Terraform Enterprise as well. Therefore, we recommend you use access keys
that are restricted if possible.
