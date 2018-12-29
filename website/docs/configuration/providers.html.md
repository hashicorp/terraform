---
layout: "docs"
page_title: "Configuring Providers"
sidebar_current: "docs-config-providers"
description: |-
  Providers are responsible in Terraform for managing the lifecycle of a resource: create, read, update, delete.
---

# Providers

While [resources](/docs/configuration/resources.html) are the primary construct
in the Terraform language, the _behaviors_ of resources rely on their
associated resource types, and these types are defined by _providers_.

Each provider offers a set of named resource types, and defines for each
resource type which arguments it accepts, which attributes it exports,
and how changes to resources of that type are actually applied to remote
APIs.

Most of the available providers correspond to one cloud or on-premises
infrastructure platform, and offer resource types that correspond to each
of the features of that platform.

Providers usually require some configuration of their own to specify endpoint
URLs, regions, authentication settings, and so on. All resource types belonging
to the same provider will share the same configuration, avoiding the need to
repeat this common information across every resource declaration.

## Provider Configuration

A provider configuration is created using a `provider` block:

```hcl
provider "google" {
  project = "acme-app"
  region  = "us-central1"
}
```

The name given in the block header (`"google"` in this example) is the name
of the provider to configure. Terraform associates each resource type with
a provider by taking the first word of the resource type name (separated by
underscores), and so the "google" provider is assumed to be the provider for
the resource type name `google_compute_instance`.

The body of the block (between `{` and `}`) contains configuration arguments
for the provider itself. Most arguments in this section are specified by
the provider itself, and indeed in this example both `project` and `region`
are specific to the `google` provider.

The configuration arguments defined by the provider may be assigned using
[expressions](/docs/configuration/expressions.html), which can for example
allow them to be parameterized by input variables. However, since provider
configurations must be evaluated in order to perform any resource type action,
provider configurations may refer only to values that are known before
the configuration is applied. In particular, avoid referring to attributes
exported by other resources unless their values are specified directly in the
configuration.

A small number of "meta-arguments" are defined by Terraform Core itself and
available for all `provider` blocks. These will be described in the following
sections.

Unlike many other objects in the Terraform language, a `provider` block may
be omitted if its contents would otherwise be empty. Terraform assumes an
empty default configuration for any provider that is not explicitly configured.

## Initialization

Each time a new provider is added to configuration -- either explicitly via
a `provider` block or by adding a resource from that provider -- Terraform
must initialize the provider before it can be used. Initialization downloads
and installs the provider's plugin so that it can later be executed.

Provider initialization is one of the actions of `terraform init`. Running
this command will download and initialize any providers that are not already
initialized.

Providers downloaded by `terraform init` are only installed for the current
working directory; other working directories can have their own installed
provider versions.

Note that `terraform init` cannot automatically download providers that are not
distributed by HashiCorp. See [Third-party Plugins](#third-party-plugins) below
for installation instructions.

For more information, see
[the `terraform init` command](/docs/commands/init.html).

## Provider Versions

Providers are plugins released on a separate rhythm from Terraform itself, and
so they have their own version numbers. For production use, you should
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

To constrain the provider version as suggested, add the `version` meta-argument
to the provider configuration block:

```hcl
provider "aws" {
  version = "~> 1.0"

  region     = "us-east-1"
}
```

This meta-argument applies to all providers.
[The `terraform providers` command](/docs/commands/providers.html) can be used
to view the specified version constraints for all providers used in the
current configuration.

The `version` argument value may either be a single explicit version or
a version constraint string. Constraint strings use the following syntax to
specify a _range_ of versions that are acceptable:

* `>= 1.2.0`: version 1.2.0 or newer
* `<= 1.2.0`: version 1.2.0 or older
* `~> 1.2.0`: any non-beta version `>= 1.2.0` and `< 1.3.0`, e.g. `1.2.X`
* `~> 1.2`: any non-beta version `>= 1.2.0` and `< 2.0.0`, e.g. `1.X.Y`
* `>= 1.0.0, <= 2.0.0`: any version between 1.0.0 and 2.0.0 inclusive

When `terraform init` is re-run with providers already installed, it will
use an already-installed provider that meets the constraints in preference
to downloading a new version. To upgrade to the latest acceptable version
of each provider, run `terraform init -upgrade`. This command also upgrades
to the latest versions of all Terraform modules.

## Multiple Provider Instances

You can optionally define multiple configurations for the same provider
to allow managing objects in multiple regions, on multiple hosts, etc. The
primary reason is multiple regions for a cloud platform. Other examples include
targeting multiple Docker hosts, multiple Consul hosts, etc.

To include multiple configurations for a given provider, include multiple
`provider` blocks with the same provider name, but set the `alias` meta-argument
to an alias name to use for each additional configuration. For example:

```hcl
# The default provider configuration
provider "aws" {
  region = "us-east-1"
}

# Additional provider configuration for west coast region
provider "aws" {
  alias  = "west"
  region = "us-west-2"
}
```

The `provider` block without `alias` set is known as the _default_ provider
configuration. When `alias` is set, it creates an _additional_ provider
configuration. For providers that have no required configuration arguments, the
implied _empty_ configuration is considered to be the _default_ provider
configuration.

Resources are normally associated with the default provider configuration
inferred from the resource type name. For example, a resource of type
`aws_instance` uses the _default_ (un-aliased) `aws` provider configuration
unless otherwise stated.

The `provider` meta-argument within any `resource` or `data` block overrides
this default behavior and allows an additional provider configuration to be
selected using its alias:

```hcl
resource "aws_instance" "foo" {
  provider = aws.west

  # ...
}
```

The value of the `provider` meta-argument is always the provider name and an
alias separated by a period, such as `aws.west` above.

Provider configurations may also be passed from a parent module into a
child module, as described in
[_Providers within Modules_](/docs/modules/usage.html#providers-within-modules).
In most cases, only _root modules_ should define provider configurations, with
all child modules obtaining their provider configurations from their parents.

## Third-party Plugins

Anyone can develop and distribute their own Terraform providers. (See
[Writing Custom Providers](/docs/extend/writing-custom-providers.html) for more
about provider development.) These third-party providers must be manually
installed, since `terraform init` cannot automatically download them.

Install third-party providers by placing their plugin executables in the user
plugins directory. The user plugins directory is in one of the following
locations, depending on the host operating system:

Operating system  | User plugins directory
------------------|-----------------------
Windows           | `%APPDATA%\terraform.d\plugins`
All other systems | `~/.terraform.d/plugins`

Once a plugin is installed, `terraform init` can initialize it normally.

Providers distributed by HashiCorp can also go in the user plugins directory. If
a manually installed version meets the configuration's version constraints,
Terraform will use it instead of downloading that provider. This is useful in
airgapped environments and when testing pre-release provider builds.

### Plugin Names and Versions

The naming scheme for provider plugins is `terraform-provider-<NAME>_vX.Y.Z`,
and Terraform uses the name to understand the name and version of a particular
provider binary.

If multiple versions of a plugin are installed, Terraform will use the newest
version that meets the configuration's version constraints.

Third-party plugins are often distributed with an appropriate filename already
set in the distribution archive, so that they can be extracted directly into the
user plugins directory.

### OS and Architecture Directories

Terraform plugins are compiled for a specific operating system and architecture,
and any plugins in the root of the user plugins directory must be compiled for
the current system.

If you use the same plugins directory on multiple systems, you can install
plugins into subdirectories with a naming scheme of `<OS>_<ARCH>` (for example,
`darwin_amd64`). Terraform uses plugins from the root of the plugins directory
and from the subdirectory that corresponds to the current system, ignoring
other subdirectories.

Terraform's OS and architecture strings are the standard ones used by the Go
language. The following are the most common:

* `darwin_amd64`
* `freebsd_386`
* `freebsd_amd64`
* `freebsd_arm`
* `linux_386`
* `linux_amd64`
* `linux_arm`
* `openbsd_386`
* `openbsd_amd64`
* `solaris_amd64`
* `windows_386`
* `windows_amd64`

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

This directory must already exist before Terraform will cache plugins;
Terraform will not create the directory itself.

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

The plugin cache directory must _not_ be the third-party plugin directory
or any other directory Terraform searches for pre-installed plugins, since
the cache management logic conflicts with the normal plugin discovery logic
when operating on the same directory.

Please note that Terraform will never itself delete a plugin from the
plugin cache once it's been placed there. Over time, as plugins are upgraded,
the cache directory may grow to contain several unused versions which must be
manually deleted.
