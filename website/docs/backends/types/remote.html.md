---
layout: "backend-types"
page_title: "Backend Type: remote"
sidebar_current: "docs-backends-types-enhanced-remote"
description: |-
  Terraform can store the state and run operations remotely, making it easier to version and work with in a team.
---

# remote

**Kind: Enhanced**

The remote backend stores state and runs operations remotely. When running
`terraform plan` with this backend, the actual execution occurs in Terraform
Enterprise, with log output streaming to the local terminal.

To use this backend you need a Terraform Enterprise account on
[app.terraform.io](https://app.terraform.io) or a private instance of Terraform
Enterprise.

## Command Support

Currently the remote backend supports the following Terraform commands:

- `fmt`
- `get`
- `init`
- `output`
- `plan`
- `providers`
- `show`
- `taint`
- `untaint`
- `validate`
- `version`
- `workspace`

Importantly, it does not support the `apply` command.

## Workspaces

The remote backend can work with either a single remote workspace, or with multiple similarly-named remote workspaces (like `networking-dev` and `networking-prod`). The `workspaces` block of the backend configuration determines which mode it uses:

- To use a single workspace, set `workspaces.name` to the remote workspace's
  full name (like `networking-prod`).

- To use multiple workspaces, set `workspaces.prefix` to a prefix used in
  all of the desired remote workspace names. For example, set
  `prefix = "networking-"` to use a group of workspaces with names like
  `networking-dev` and `networking-prod`.

    When interacting with workspaces on the command line, Terraform uses
    shortened names without the common prefix. For example, if
    `prefix = "networking-"`, use `terraform workspace select prod` to switch to
    the `networking-prod` workspace.

    In prefix mode, the special `default` workspace is disabled. Before running
    `terraform init`, ensure that there is no state stored for the local
    `default` workspace and that a non-default workspace is currently selected;
    otherwise, the initialization will fail.

The backend configuration requires either `name` or `prefix`. Omitting both or
setting both results in a configuration error.

If previous state is present when you run `terraform init` and the corresponding
remote workspaces are empty or absent, Terraform will create workspaces and/or
update the remote state accordingly.

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

  config {
    organization = "company"

    workspaces {
      name = "my-app-prod"
    }
  }
}
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
    workspace names are used in Terraform Enterprise, and the short names
    (minus the prefix) are used on the command line. If omitted, only the
    default workspace can be used. This option conflicts with `name`.
