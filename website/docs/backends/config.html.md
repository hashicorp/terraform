---
layout: "docs"
page_title: "Backends: Configuration"
sidebar_current: "docs-backends-config"
description: |-
  Backends are configured directly in Terraform files in the `terraform` section.
---

# Backend Configuration

Backends are configured directly in Terraform files in the `terraform`
section. After configuring a backend, it has to be
[initialized](/docs/backends/init.html).

Below, we show a complete example configuring the "consul" backend:

```hcl
terraform {
  backend "consul" {
    address = "demo.consul.io"
    path    = "example_app/terraform_state"
  }
}
```

You specify the backend type as a key to the `backend` stanza. Within the
stanza are backend-specific configuration keys. The list of supported backends
and their configuration is in the sidebar to the left.

Only one backend may be specified and the configuration **may not contain
interpolations**. Terraform will validate this.

## First Time Configuration

When configuring a backend for the first time (moving from no defined backend
to explicitly configuring one), Terraform will give you the option to migrate
your state to the new backend. This lets you adopt backends without losing
any existing state.

To be extra careful, we always recommend manually backing up your state
as well. You can do this by simply copying your `terraform.tfstate` file
to another location. The initialization process should create a backup
as well, but it never hurts to be safe!

Configuring a backend for the first time is no different than changing
a configuration in the future: create the new configuration and run
`terraform init`. Terraform will guide you the rest of the way.

## Partial Configuration

You do not need to specify every required argument in the backend configuration.
Omitting certain arguments may be desirable to avoid storing secrets, such as
access keys, within the main configuration. When some or all of the arguments
are omitted, we call this a _partial configuration_.

With a partial configuration, the remaining configuration arguments must be
provided as part of
[the initialization process](/docs/backends/init.html#backend-initialization).
There are several ways to supply the remaining arguments:

  * **Interactively**: Terraform will interactively ask you for the required
    values, unless interactive input is disabled. Terraform will not prompt for
    optional values.

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
```

The same settings can alternatively be specified on the command line as
follows:

```
$ terraform init \
    -backend-config="address=demo.consul.io" \
    -backend-config="path=example_app/terraform_state"
```

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
