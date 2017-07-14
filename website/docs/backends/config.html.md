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
    path    = "tfdocs"
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

You do not need to specify every required attribute in the configuration.
This may be desirable to avoid storing secrets (such as access keys) within
the configuration itself. We call this specifying only a _partial_ configuration.

With a partial configuration, the remaining configuration is expected as
part of the [initialization](/docs/backends/init.html) process. There are
a few ways to supply the remaining configuration:

  * **Interactively**: Terraform will interactively ask you for the required
    values. Terraform will not ask you for optional values.

  * **File**: A configuration file may be specified via the command line.
    This file can then be sourced via some secure means (such as
    [Vault](https://www.vaultproject.io)).

  * **Command-line key/value pairs**: Key/value pairs in the format of
    `key=value` can be specified as part of the init command. Note that
    many shells retain command-line flags in a history file, so this isn't
    recommended for secrets.

In all cases, the final configuration is stored on disk in the
".terraform" directory, which should be ignored from version control.

This means that sensitive information can be omitted from version control
but it ultimately still lives on disk. In the future, Terraform may provide
basic encryption on disk so that values are at least not plaintext.

When using partial configuration, Terraform requires at a minimum that
an empty backend configuration is in the Terraform files. For example:

```hcl
terraform {
  backend "consul" {}
}
```

This minimal requirement allows Terraform to detect _unsetting_ backends.
We cannot accept the backend type on the command-line because while it is
technically possible, Terraform would then be unable to detect if you
want to unset your backend (and move back to local state).

## Changing Configuration

You can change your backend configuration at any time. You can change
both the configuration itself as well as the type of backend (for example
from "consul" to "s3").

Terraform will automatically detect any changes in your configuration
and request a [reinitialization](/docs/backends/init.html). As part of
the reinitialization process, Terraform will ask if you'd like to migrate
your existing state to the new configuration. This allows you to easily
switch from one backend to another.

If you're using [state environments](/docs/state/environments.html),
Terraform is able to copy all environments to the destination. If Terraform
detects you have multiple states, it will ask if this is what you want to do.

If you're just reconfiguring the same backend, Terraform will still ask if you
want to migrate your state. You can respond "no" in this scenario.

## Unconfiguring a Backend

If you no longer want to use any backend, you can simply remove the
configuration from the file. Terraform will detect this like any other
change and prompt you to [reinitialize](/docs/backends/init.html).

As part of the reinitialization, Terraform will ask if you'd like to migrate
your state back down to normal local state. Once this is complete then
Terraform is back to behaving as it does by default.
