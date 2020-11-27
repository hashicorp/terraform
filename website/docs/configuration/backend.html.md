---
layout: "language"
page_title: "Backend Configuration - Configuration Language"
---

# Backend Configuration

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Terraform Settings](../configuration-0-11/terraform.html).


Each Terraform configuration can specify a backend, which defines exactly where
and how operations are performed, where [state](/docs/state/index.html)
snapshots are stored, etc. Most non-trivial Terraform configurations configure
a remote backend so that multiple people can work with the same infrastructure.

## Using a Backend Block

Backends are configured with a nested `backend` block within the top-level
`terraform` block:

```hcl
terraform {
  backend "remote" {
    organization = "example_corp"

    workspaces {
      name = "my-app-prod"
    }
  }
}
```

There are some important limitations on backend configuration:

- A configuration can only provide one backend block.
- A backend block cannot refer to named values (like input variables, locals, or data source attributes).

### Backend Types

The block label of the backend block (`"remote"`, in the example above) indicates which backend type to use. Terraform has a built-in selection of backends, and the configured backend must be available in the version of Terraform you are using.

The arguments used in the block's body are specific to the chosen backend type; they configure where and how the backend will store the configuration's state, and in some cases configure other behavior.

Some backends allow providing access credentials directly as part of the configuration for use in unusual situations, for pragmatic reasons. However, in normal use we _do not_ recommend including access credentials as part of the backend configuration. Instead, leave those arguments completely unset and provide credentials via the credentials files or environment variables that are conventional for the target system, as described in the documentation for each backend.

See _[Backend Types](/docs/backends/types/index.html)_ for details about each supported backend type and its configuration arguments.

### Default Backend

If a configuration includes no backend block, Terraform defaults to using the `local` backend, which performs operations on the local system and stores state as a plain file in the current working directory.

## Initialization

Whenever a configuration's backend changes, you must run `terraform init` again
to validate and configure the backend before you can perform any plans, applies,
or state operations.

When changing backends, Terraform will give you the option to migrate
your state to the new backend. This lets you adopt backends without losing
any existing state.

To be extra careful, we always recommend manually backing up your state
as well. You can do this by simply copying your `terraform.tfstate` file
to another location. The initialization process should create a backup
as well, but it never hurts to be safe!

## Partial Configuration

You do not need to specify every required argument in the backend configuration.
Omitting certain arguments may be desirable if some arguments are provided
automatically by an automation script running Terraform. When some or all of
the arguments are omitted, we call this a _partial configuration_.

With a partial configuration, the remaining configuration arguments must be
provided as part of
[the initialization process](/docs/backends/init.html#backend-initialization).
There are several ways to supply the remaining arguments:

  * **File**: A configuration file may be specified via the `init` command line.
    To specify a file, use the `-backend-config=PATH` option when running
    `terraform init`. If the file contains secrets it may be kept in
    a secure data store, such as
    [Vault](https://www.vaultproject.io/), in which case it must be downloaded
    to the local disk before running Terraform.

  * **Command-line key/value pairs**: Key/value pairs can be specified via the
    `init` command line. Note that many shells retain command-line flags in a
    history file, so this isn't recommended for secrets. To specify a single
    key/value pair, use the `-backend-config="KEY=VALUE"` option when running
    `terraform init`.

  * **Interactively**: Terraform will interactively ask you for the required
    values, unless interactive input is disabled. Terraform will not prompt for
    optional values.

If backend settings are provided in multiple locations, the top-level
settings are merged such that any command-line options override the settings
in the main configuration and then the command-line options are processed
in order, with later options overriding values set by earlier options.

The final, merged configuration is stored on disk in the `.terraform`
directory, which should be ignored from version control. This means that
sensitive information can be omitted from version control, but it will be
present in plain text on local disk when running Terraform.

When using partial configuration, Terraform requires at a minimum that
an empty backend configuration is specified in one of the root Terraform
configuration files, to specify the backend type. For example:

```hcl
terraform {
  backend "consul" {}
}
```

A backend configuration file has the contents of the `backend` block as
top-level attributes, without the need to wrap it in another `terraform`
or `backend` block:

```hcl
address = "demo.consul.io"
path    = "example_app/terraform_state"
scheme  = "https"
```

The same settings can alternatively be specified on the command line as
follows:

```
$ terraform init \
    -backend-config="address=demo.consul.io" \
    -backend-config="path=example_app/terraform_state" \
    -backend-config="scheme=https"
```

The Consul backend also requires a Consul access token. Per the recommendation
above of omitting credentials from the configuration and using other mechanisms,
the Consul token would be provided by setting either the `CONSUL_HTTP_TOKEN`
or `CONSUL_HTTP_AUTH` environment variables. See the documentation of your
chosen backend to learn how to provide credentials to it outside of its main
configuration.

## Changing Configuration

You can change your backend configuration at any time. You can change
both the configuration itself as well as the type of backend (for example
from "consul" to "s3").

Terraform will automatically detect any changes in your configuration
and request a [reinitialization](/docs/backends/init.html). As part of
the reinitialization process, Terraform will ask if you'd like to migrate
your existing state to the new configuration. This allows you to easily
switch from one backend to another.

If you're using multiple [workspaces](/docs/state/workspaces.html),
Terraform can copy all workspaces to the destination. If Terraform detects
you have multiple workspaces, it will ask if this is what you want to do.

If you're just reconfiguring the same backend, Terraform will still ask if you
want to migrate your state. You can respond "no" in this scenario.

## Unconfiguring a Backend

If you no longer want to use any backend, you can simply remove the
configuration from the file. Terraform will detect this like any other
change and prompt you to [reinitialize](/docs/backends/init.html).

As part of the reinitialization, Terraform will ask if you'd like to migrate
your state back down to normal local state. Once this is complete then
Terraform is back to behaving as it does by default.
