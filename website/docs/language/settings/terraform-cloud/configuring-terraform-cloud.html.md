---
layout: "language"
page_title: "Configuring Terraaform Cloud"
sidebar_current: "terraform-cloud-configuration"
description: "Configuring Terraform Cloud"
---

# Configuring Terraform Cloud

To enable the [CLI-driven run workflow](https://www.terraform.io/docs/cloud/run/cli.html), a
Terraform configuration can integrate Terraform Cloud via a special `cloud` block within the
top-level `terraform` block, e.g.:

```
terraform {
  cloud {
    organization = "my-org"
    workspaces {
      tags = ["networking"]
    }
  }
}
```

Using the Cloud integration is mutually exclusive of declaring any backend; that is, a configuration
can only declare one or the other. Similar to backends...

- A configuration can only provide one cloud block.
- A cloud block cannot refer to named values (like input variables, locals, or data source attributes).

## Configuration variables

The following configuration options are supported:

* `organization` - (Required) The name of the organization containing the
  workspace(s) the current configuration should be mapped to.

* `workspaces` - (Required) A block declaring a strategy for mapping local CLI workspaces to remote
  Terraform Cloud workspaces.
  The `workspaces` block supports the following keys, each denoting a 'strategy':

  * `tags` - (Optional) A set of tags used to select remote Terraform Cloud workspaces to be used for this single
configuration.  New workspaces will automatically be tagged with these tag values.  Generally, this
is the primary and recommended strategy to use.  This option conflicts with "name".

  * `name` - (Optional) The name of a single Terraform Cloud workspace to be used with this configuration When configured
only the specified workspace can be used. This option conflicts with "tags".

* `hostname` - (Optional) The hostname of a Terraform Enterprise installation, if using Terraform
  Enterprise. Defaults to Terraform Cloud (app.terraform.io).

* `token` - (Optional) The token used to authenticate with Terraform Cloud.
  We recommend omitting the token from the configuration, and instead using
  [`terraform login`](/docs/cli/commands/login.html) or manually configuring
  `credentials` in the
  [CLI config file](/docs/cli/config/config-file.html#credentials).

## Workspaces

Terraform can be configured to work with multiple Terraform Cloud workspaces using [Terraform's named workspaces feature](/docs/cli/workspaces/index.html) or a single explicit Terraform Cloud workspace.

* _Terraform CLI workspaces_ are representations of multiple state files associated with a single
_configuration_. They typically represent multiple _deployment environments_ that the single
configuration can be provisioned to (`prod`, `staging`, `dev`, etc).

* _Terraform Cloud workspaces_ are more flexible in that they are not associated with a particular
configuration but by a Terraform Cloud organization. Two related workspaces (representing the `prod`
and `staging` deployment environments of a single set of resources) may _or may not_ be associated
with the same configuration. In this sense, a Terraform Cloud workspace behaves more like a
completely separate working directory, and is typically named by both the set of resources it
contains as well as the deployment environment it provisions to (`networking-prod-us-east`,
`networking-staging-us-east`, etc).

The `workspaces` block of the `cloud` configuration determines how Terraform maps workspaces for the
current configuration to Terraform Cloud workspaces in the given organization.

## Example Configurations

### Basic Configuration

```hcl
terraform {
  cloud {
    organization = "company"
    workspaces {
      tags = ["networking", "source:cli"]
    }
  }
}
```

In the example above, all Terraform Cloud workspaces with the `networking` and `source:cli` tags
will be mapped to the current configuration. `terraform workspace new example` would similarly
create a new Terraform Cloud workspace named `example` tagged with `networking` and `source:cli`.

### Configurating a single workspace

```hcl
terraform {
  cloud {
    organization = "company"
    workspaces {
      name = "networking-prod-us-east"
    }
  }
}
```

In the example above, the `workspaces` block maps the current configuration to a single specific
Terraform Cloud workspace named `networking-prod-us-east`; Terraform will create this workspace if
it does not yet exist when running `terraform init`. Note that using a particular workspace in this
way means that commands which utilize multiple workspaces have no effect (e.g. `terraform workspace
new`, `terraform workspace select`, etc).

### Using Partial Configuration

Like a state backend, the `cloud` option supports [partial
configuration](/docs/language/settings/backends/configuration.html#partial-configuration) with the
`-backend-config` flag, allowing you to supply configuration values from separate file.

```hcl
# main.tf
terraform {
  required_version = "~> 1.1.0"

  cloud {}
}
```

Configuration file:

```hcl
# config.cloud
workspaces { tags = ["networking"] }
hostname     = "app.terraform.io"
organization = "company"
```

Running `terraform init` with the configuration file:

```sh
$ terraform init -backend-config=config.cloud
```

### Excluding Files from Upload with .terraformignore

When executing a remote `plan` or `apply` in a [CLI-driven run](/docs/cloud/run/cli.html),
an archive of your configuration directory is uploaded to Terraform Cloud. You can define
paths to ignore from upload via a `.terraformignore` file at the root of your configuration directory. If this file is not present, the archive will exclude the following by default:

* .git/ directories
* .terraform/ directories (exclusive of .terraform/modules)

The `.terraformignore` file can include rules as one would include in a
[.gitignore file](https://git-scm.com/book/en/v2/Git-Basics-Recording-Changes-to-the-Repository#_ignoring)


* Comments (starting with `#`) or blank lines are ignored
* End a pattern with a forward slash / to specify a directory
* Negate a pattern by starting it with an exclamation point `!`

Note that unlike `.gitignore`, only the `.terraformignore` at the root of the configuration
directory is considered.
