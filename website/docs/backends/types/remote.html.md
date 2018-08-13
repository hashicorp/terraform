---
layout: "backend-types"
page_title: "Backend Type: remote"
sidebar_current: "docs-backends-types-enhanced-remote"
description: |-
  Terraform can store the state and run operations remotely, making it easier to version and work with in a team.
---

# remote

**Kind: Enhanced**

The remote backend stores state and runs operations remotely. In order
use this backend you need a Terraform Enterprise account or have Private
Terraform Enterprise running on-premises.

### Commands

Currently the remote backend supports the following Terraform commands:

 1.  fmt
 2.  get
 3.  init
 4.  output
 5.  plan
 6.  providers
 7.  show
 8.  taint
 9.  untaint
 10. validate
 11. version
 11. workspace

### Workspaces
To work with remote workspaces we need either a name or a prefix. You will
get a configuration error when neither or both options are configured.

#### Name
When a name is provided, that name is used to make a one-to-one mapping
between your local “default” workspace and a named remote workspace. This
option assumes you are not using workspaces when working with TF, so it
will act as a backend that does not support names states.

#### Prefix
When a prefix is provided it will be used to filter and map workspaces that
can be used with a single configuration. This allows you to dynamically
filter and map all remote workspaces with a matching prefix.

The prefix is added when making calls to the remote backend and stripped
again when receiving the responses. This way any locally used workspace
names will remain the same short names (e.g. “tst”, “acc”) while the remote
names will be mapped by adding the prefix.

It is assumed that you are only using named workspaces when working with
Terraform and so the “default” workspace is ignored in this case. If there
is a state file for the “default” config, this will give an error during
`terraform init`. If the default workspace is selected when running the
`init` command, the `init` process will succeed but will end with a message
that tells you how to select an existing workspace or create a new one.

## Example Configuration

```hcl
terraform {
  backend "remote" {
    hostname = "app.terraform.io"
    organization = "company"

    workspaces {
      name = "workspace"
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
      name = "workspace"
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
	We recommend omitting the token which can be set as `credentials` in the
  [CLI config file](/docs/commands/cli-config.html#credentials).
* `workspaces` - (Required) Workspaces contains arguments used to filter down
  to a set of workspaces to work on. Parameters defined below.

The `workspaces` block supports the following keys:
* `name` - (Optional) A workspace name used to map the default workspace to a
  named remote workspace. When configured only the default workspace can be
  used. This option conflicts with `prefix`.
* `prefix` - (Optional) A prefix used to filter workspaces using a single
  configuration. New workspaces will automatically be prefixed with this
  prefix. If omitted only the default workspace can be used. This option
  conflicts with `name`.
