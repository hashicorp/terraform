---
layout: "backend-types"
page_title: "Backend Type: remote"
sidebar_current: "docs-backends-types-enhanced-remote"
description: |-
  Terraform can store the state and run operations remotely, making it easier to version and work with in a team.
---

# remote

**Kind: Enhanced**

-> **Note:** We recommend using Terraform v0.11.13 or newer with this
backend. This backend requires either a Terraform Cloud account on
[app.terraform.io](https://app.terraform.io) or a Terraform Enterprise instance
(version v201809-1 or newer).

The remote backend stores Terraform state and may be used to run operations in Terraform Cloud.

When using full remote operations, operations like `terraform plan` or `terraform apply` can be executed in Terraform
Cloud's run environment, with log output streaming to the local terminal. Remote plans and applies use variable values from the associated Terraform Cloud workspace. 

Terraform Cloud can also be used with local operations, in which case only state is stored in the Terraform Cloud backend.



## Command Support

Currently the remote backend supports the following Terraform commands:

- `apply`
- `console` (supported in Terraform >= v0.11.12)
- `destroy` (requires manually setting `CONFIRM_DESTROY=1` on the workspace)
- `fmt`
- `get`
- `graph` (supported in Terraform >= v0.11.12)
- `import` (supported in Terraform >= v0.11.12)
- `init`
- `output`
- `plan`
- `providers`
- `show`
- `state` (supports all sub-commands: list, mv, pull, push, rm, show)
- `taint`
- `untaint`
- `validate`
- `version`
- `workspace`

## Workspaces

The remote backend can work with either a single remote Terraform Cloud workspace,
or with multiple similarly-named remote workspaces (like `networking-dev`
and `networking-prod`). The `workspaces` block of the backend configuration
determines which mode it uses:

- To use a single remote Terraform Cloud workspace, set `workspaces.name` to the
  remote workspace's full name (like `networking`).

- To use multiple remote workspaces, set `workspaces.prefix` to a prefix used in
  all of the desired remote workspace names. For example, set
  `prefix = "networking-"` to use Terraform cloud workspaces with
  names like `networking-dev` and `networking-prod`. This is helpful when
  mapping multiple Terraform CLI [workspaces](../../state/workspaces.html)
  used in a single Terraform configuration to multiple Terraform Cloud
  workspaces.

When interacting with workspaces on the command line, Terraform uses
shortened names without the common prefix. For example, if
`prefix = "networking-"`, use `terraform workspace select prod` to switch to
the Terraform CLI workspace `prod` within the current configuration. Remote
Terraform operations such as `plan` and `apply` executed against that Terraform
CLI workspace will be executed in the Terraform Cloud workspace `networking-prod`.

Additionally, the [`${terraform.workspace}`](../../state/workspaces.html#current-workspace-interpolation)
interpolation sequence should be removed from Terraform configurations that run
remote operations against Terraform Cloud workspaces. The reason for this is that
each Terraform Cloud workspace currently only uses the single `default` Terraform
CLI workspace internally. In other words, if your Terraform configuration
used `${terraform.workspace}` to return `dev` or `prod`, remote runs in Terraform Cloud
would always evaluate it as `default` regardless of
which workspace you had set with the `terraform workspace select` command. That
would most likely not be what you wanted. (It is ok to use `${terraform.workspace}`
in local operations.)

The backend configuration requires either `name` or `prefix`. Omitting both or
setting both results in a configuration error.

If previous state is present when you run `terraform init` and the corresponding
remote workspaces are empty or absent, Terraform will create workspaces and/or
update the remote state accordingly. However, if your workspace needs variables
set or requires a specific version of Terraform for remote operations, we
recommend that you create your remote workspaces on Terraform Cloud before
running any remote operations against them.

## Example Configuration

```hcl
# Using a single workspace:
terraform {
  backend "remote" {
    hostname = "app.terraform.io"
    organization = "company"

    workspaces {
      name = "my-app-prod"
    }
  }
}

# Using multiple workspaces:
terraform {
  backend "remote" {
    hostname = "app.terraform.io"
    organization = "company"

    workspaces {
      prefix = "my-app-"
    }
  }
}
```

## Example Reference

```hcl
data "terraform_remote_state" "foo" {
  backend = "remote"

  config = {
    organization = "company"

    workspaces {
      name = "workspace"
    }
  }
}
```

## Example configuration using CLI input

```hcl
# main.tf
terraform {
  required_version = "~> 0.12.0"

  backend "remote" {}
}
```

Backend configuration file:

```hcl
# backend.hcl
workspaces { name = "workspace" }
hostname     = "app.terraform.io"
organization = "company"
```

Running `terraform init` with the backend file:

```sh
terraform init -backend-config=backend.hcl
```

## Configuration variables

The following configuration options are supported:

* `hostname` - (Optional) The remote backend hostname to connect to. Defaults
  to app.terraform.io.
* `organization` - (Required) The name of the organization containing the
  targeted workspace(s).
* `token` - (Optional) The token used to authenticate with the remote backend.
  We recommend omitting the token from the configuration, and instead setting it
  as `credentials` in the
  [CLI config file](/docs/commands/cli-config.html#credentials).
* `workspaces` - (Required) A block specifying which remote workspace(s) to use.
  The `workspaces` block supports the following keys:

  * `name` - (Optional) The full name of one remote workspace. When configured,
    only the default workspace can be used. This option conflicts with `prefix`.
  * `prefix` - (Optional) A prefix used in the names of one or more remote
    workspaces, all of which can be used with this configuration. The full
    workspace names are used in Terraform Cloud, and the short names
    (minus the prefix) are used on the command line for Terraform CLI workspaces.
    If omitted, only the default workspace can be used. This option conflicts with `name`.
    
->  **Note** You must use the `name` key when configuring a `terraform_remote_state`
data source that retrieves state from another Terraform Cloud workspace. The `prefix` key is only
intended for use when configuring an instance of the remote backend.

## Excluding Files from Upload with .terraformignore

-> **Version note:** `.terraformignore` support was added in Terraform 0.12.11.

When executing a remote `plan` or `apply` in a [CLI-driven run](/docs/cloud/run/cli.html),
an archive of your configuration directory is uploaded to Terraform Cloud. You can define
paths to ignore from upload via a `.terraformignore` file at the root of your configuration directory. If this file is not present, the archive will exclude the following by default:

* .git/ directories
* .terraform/ directories (exclusive of .terraform/modules)

The `.terraformignore` file can include rules as one would include in a
[.gitignore file](https://git-scm.com/book/en/v2/Git-Basics-Recording-Changes-to-the-Repository#Ignoring-Files)


* Comments (starting with `#`) or blank lines are ignored
* End a pattern with aforward slash / to specify a directory
* Negate a pattern by starting it with an exclamation point `!`

Note that unlike `.gitignore`, only the `.terraformignore` at the root of the configuration
directory is considered.
