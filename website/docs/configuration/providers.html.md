---
layout: "docs"
page_title: "Configuring Providers"
sidebar_current: "docs-config-providers"
description: |-
  Providers are responsible in Terraform for managing the lifecycle of a resource: create, read, update, delete.
---

# Provider Configuration

Providers are responsible in Terraform for managing the lifecycle
of a [resource](/docs/configuration/resources.html): create,
read, update, delete.

Most providers require some sort of configuration to provide
authentication information, endpoint URLs, etc. Where explicit configuration
is required, a `provider` block is used within the configuration as
illustrated in the following sections.

By default, resources are matched with provider configurations by matching
the start of the resource name. For example, a resource of type
`vsphere_virtual_machine` is associated with a provider called `vsphere`.

This page assumes you're familiar with the
[configuration syntax](/docs/configuration/syntax.html)
already.

## Example

A provider configuration looks like the following:

```hcl
provider "aws" {
  access_key = "foo"
  secret_key = "bar"
  region     = "us-east-1"
}
```

## Description

A `provider` block represents a configuration for the provider named in its
header. For example, `provider "aws"` above is a configuration for the
`aws` provider.

Within the block body (between `{ }`) is configuration for the provider.
The configuration is dependent on the type, and is documented
[for each provider](/docs/providers/index.html).

The arguments `alias` and `version`, if present, are special arguments
handled by Terraform Core for their respective features described above. All
other arguments are defined by the provider itself.

A `provider` block may be omitted if its body would be empty. Using a resource
in configuration implicitly creates an empty provider configuration for it
unless a `provider` block is explicitly provided.

## Initialization

Each time a new provider is added to configuration -- either explicitly via
a `provider` block or by adding a resource from that provider -- it's necessary
to initialize that provider before use. Initialization downloads and installs
the provider's plugin and prepares it to be used.

Provider initialization is one of the actions of `terraform init`. Running
this command will download and initialize any providers that are not already
initialized.

For more information, see
[the `terraform init` command](/docs/commands/init.html).

## Provider Versions

Providers are released on a separate rhythm from Terraform itself, and thus
have their own version numbers. For production use, it is recommended to
constrain the acceptable provider versions via configuration, to ensure that
new versions with breaking changes will not be automatically installed by
`terraform init` in future.

When `terraform init` is run _without_ provider version constraints, it
prints a suggested version constraint string for each provider:

```
The following providers do not have any version constraints in configuration,
so the latest version was installed.

To prevent automatic upgrades to new major versions that may contain breaking
changes, it is recommended to add version = "..." constraints to the
corresponding provider blocks in configuration, with the constraint strings
suggested below.

* provider.aws: version = "~> 1.0"
```

To constrain the provider version as suggested, add a `version` argument to
the provider configuration block:

```hcl
provider "aws" {
  version = "~> 1.0"

  access_key = "foo"
  secret_key = "bar"
  region     = "us-east-1"
}
```

This special argument applies to _all_ providers.
[`terraform providers`](/docs/commands/providers.html) can be used to
view the specified version constraints for all providers used in the
current configuration.

When `terraform init` is re-run with providers already installed, it will
use an already-installed provider that meets the constraints in preference
to downloading a new version. To upgrade to the latest acceptable version
of each provider, run `terraform init -upgrade`. This command also upgrades
to the latest versions of all Terraform modules.

## Multiple Provider Instances

You can define multiple configurations for the same provider in order to support
multiple regions, multiple hosts, etc. The primary use case for this is
using multiple cloud regions. Other use-cases include targeting multiple
Docker hosts, multiple Consul hosts, etc.

To include multiple configurations fo a given provider, include multiple
`provider` blocks with the same provider name, but set the `alias` field to an
instance name to use for each additional instance. For example:

```hcl
# The default provider configuration
provider "aws" {
  # ...
}

# Additional provider configuration for west coast region
provider "aws" {
  alias  = "west"
  region = "us-west-2"
}
```

A `provider` block with out `alias` set is known as the _default_ provider
configuration. When `alias` is set, it creates an _additional_ provider
configuration. For providers that have no required configuration arguments, the
implied _empty_ configuration is also considered to be a _default_ provider
configuration.

Resources are normally associated with the default provider configuration
inferred from the resource type name. For example, a resource of type
`aws_instance` uses the _default_ (un-aliased) `aws` provider configuration
unless otherwise stated.

The `provider` argument within any `resource` or `data` block overrides this
default behavior and allows an additional provider configuration to be
selected using its alias:

```hcl
resource "aws_instance" "foo" {
  provider = "aws.west"

  # ...
}
```

The value of the `provider` argument is always the provider name and an
alias separated by a period, such as `"aws.west"` above.

Provider configurations may also be passed from a parent module into a
child module, as described in
[_Providers within Modules_](/docs/modules/usage.html#providers-within-modules).

## Interpolation

Provider configurations may use [interpolation syntax](/docs/configuration/interpolation.html)
to allow dynamic configuration:

```hcl
provider "aws" {
  region = "${var.aws_region}"
}
```

Interpolation is supported only for the per-provider configuration arguments.
It is not supported for the special `alias` and `version` arguments.

Although in principle it is possible to use any interpolation expression within
a provider configuration argument, providers must be configurable to perform
almost all operations within Terraform, and so it is not possible to use
expressions whose value cannot be known until after configuration is applied,
such as the id of a resource.

It is always valid to use [input variables](/docs/configuration/variables.html)
and [data sources](/docs/configuration/data-sources.html) whose configurations
do not in turn depend on as-yet-unknown values. [Local values](/docs/configuration/locals.html)
may also be used, but currently may cause errors when running `terraform destroy`.

## Third-party Plugins

At present Terraform can automatically install only the providers distributed
by HashiCorp. Third-party providers can be manually installed by placing
their plugin executables in one of the following locations depending on the
host operating system:

* On Windows, in the sub-path `terraform.d/plugins` beneath your user's
  "Application Data" directory.
* On all other systems, in the sub-path `.terraform.d/plugins` in your
  user's home directory.

`terraform init` will search this directory for additional plugins during
plugin initialization.

The naming scheme for provider plugins is `terraform-provider-NAME-vX.Y.Z`,
and Terraform uses the name to understand the name and version of a particular
provider binary. Third-party plugins will often be distributed with an
appropriate filename already set in the distribution archive so that it can
be extracted directly into the plugin directory described above.

## Provider Plugin Cache

By default, `terraform init` downloads plugins into a subdirectory of the
working directory so that each working directory is self-contained. As a
consequence, if you have multiple configurations that use the same provider
then a separate copy of its plugin will be downloaded for each configuration.

Given that provider plugins can be quite large (on the order of hundreds of
megabytes), this default behavior can be inconvenient for those with slow
or metered Internet connections. Therefore Terraform optionally allows the
use of a local directory as a shared plugin cache, which then allows each
distinct plugin binary to be downloaded only once.

To enable the plugin cache, use the `plugin_cache_dir` setting in
[the CLI configuration file](https://www.terraform.io/docs/commands/cli-config.html).
For example:

```hcl
# (Note that the CLI configuration file is _not_ the same as the .tf files
#  used to configure infrastructure.)

plugin_cache_dir = "$HOME/.terraform.d/plugin-cache"
```

Please note that on Windows it is necessary to use forward slash separators
(`/`) rather than the conventional backslash (`\`) since the configuration
file parser considers a backslash to begin an escape sequence.

Setting this in the configuration file is the recommended approach for a
persistent setting. Alternatively, the `TF_PLUGIN_CACHE_DIR` environment
variable can be used to enable caching or to override an existing cache
directory within a particular shell session:

```bash
export TF_PLUGIN_CACHE_DIR="$HOME/.terraform.d/plugin-cache"
```

When a plugin cache directory is enabled, the `terraform init` command will
still access the plugin distribution server to obtain metadata about which
plugins are available, but once a suitable version has been selected it will
first check to see if the selected plugin is already available in the cache
directory. If so, the already-downloaded plugin binary will be used.

If the selected plugin is not already in the cache, it will be downloaded
into the cache first and then copied from there into the correct location
under your current working directory.

When possible, Terraform will use hardlinks or symlinks to avoid storing
a separate copy of a cached plugin in multiple directories. At present, this
is not supported on Windows and instead a copy is always created.

The plugin cache directory must *not* be the third-party plugin directory
or any other directory Terraform searches for pre-installed plugins, since
the cache management logic conflicts with the normal plugin discovery logic
when operating on the same directory.

Please note that Terraform will never itself delete a plugin from the
plugin cache once it's been placed there. Over time, as plugins are upgraded,
the cache directory may grow to contain several unused versions which must be
manually deleted.
