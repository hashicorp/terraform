---
layout: "docs"
page_title: "CLI Configuration"
sidebar_current: "docs-commands-cli-config"
description: |-
  The general behavior of the Terraform CLI can be customized using the CLI
  configuration file.
---

# CLI Configuration File (`.terraformrc` or `terraform.rc`)

The CLI configuration file configures per-user settings for CLI behaviors,
which apply across all Terraform working directories. This is separate from
[your infrastructure configuration](/docs/language/index.html).

## Locations

The configuration can be placed in a single file whose location depends
on the host operating system:

* On Windows, the file must be named `terraform.rc` and placed
  in the relevant user's `%APPDATA%` directory. The physical location
  of this directory depends on your Windows version and system configuration;
  use `$env:APPDATA` in PowerShell to find its location on your system.
* On all other systems, the file must be named `.terraformrc` (note
  the leading period) and placed directly in the home directory
  of the relevant user.

On Windows, beware of Windows Explorer's default behavior of hiding filename
extensions. Terraform will not recognize a file named `terraform.rc.txt` as a
CLI configuration file, even though Windows Explorer may _display_ its name
as just `terraform.rc`. Use `dir` from PowerShell or Command Prompt to
confirm the filename.

The location of the Terraform CLI configuration file can also be specified
using the `TF_CLI_CONFIG_FILE` [environment variable](/docs/cli/config/environment-variables.html).
Any such file should follow the naming pattern `*.tfrc`.

## Configuration File Syntax

The configuration file uses the same _HCL_ syntax as `.tf` files, but with
different attributes and blocks. The following example illustrates the
general syntax; see the following section for information on the meaning
of each of these settings:

```hcl
plugin_cache_dir   = "$HOME/.terraform.d/plugin-cache"
disable_checkpoint = true
```

## Available Settings

The following settings can be set in the CLI configuration file:

- `credentials` - configures credentials for use with Terraform Cloud or
  Terraform Enterprise. See [Credentials](#credentials) below for more
  information.

- `credentials_helper` - configures an external helper program for the storage
  and retrieval of credentials for Terraform Cloud or Terraform Enterprise.
  See [Credentials Helpers](#credentials-helpers) below for more information.

- `disable_checkpoint` — when set to `true`, disables
  [upgrade and security bulletin checks](/docs/cli/commands/index.html#upgrade-and-security-bulletin-checks)
  that require reaching out to HashiCorp-provided network services.

- `disable_checkpoint_signature` — when set to `true`, allows the upgrade and
  security bulletin checks described above but disables the use of an anonymous
  id used to de-duplicate warning messages.

- `plugin_cache_dir` — enables
  [plugin caching](#provider-plugin-cache)
  and specifies, as a string, the location of the plugin cache directory.

- `provider_installation` - customizes the installation methods used by
  `terraform init` when installing provider plugins. See
  [Provider Installation](#provider-installation) below for more information.

## Credentials

[Terraform Cloud](/docs/cloud/index.html) provides a number of remote network
services for use with Terraform, and
[Terraform Enterprise](/docs/enterprise/index.html) allows hosting those
services inside your own infrastructure. For example, these systems offer both
[remote operations](/docs/cloud/run/cli.html) and a
[private module registry](/docs/cloud/registry/index.html).

When interacting with Terraform-specific network services, Terraform expects
to find API tokens in CLI configuration files in `credentials` blocks:

```hcl
credentials "app.terraform.io" {
  token = "xxxxxx.atlasv1.zzzzzzzzzzzzz"
}
```

If you are running the Terraform CLI interactively on a computer with a web browser, you can use [the `terraform login` command](/docs/cli/commands/login.html)
to get credentials and automatically save them in the CLI configuration. If
not, you can manually write `credentials` blocks.

You can have multiple `credentials` blocks if you regularly use services from
multiple hosts. Many users will configure only one, for either
Terraform Cloud (at `app.terraform.io`) or for their organization's own
Terraform Enterprise host. Each `credentials` block contains a `token` argument
giving the API token to use for that host.

~> **Important:** If you are using Terraform Cloud or Terraform Enterprise,
the token provided must be either a
[user token](/docs/cloud/users-teams-organizations/users.html#api-tokens)
or a
[team token](/docs/cloud/users-teams-organizations/api-tokens.html#team-api-tokens);
organization tokens cannot be used for command-line Terraform actions.

-> **Note:** The credentials hostname must match the hostname in your module
sources and/or backend configuration. If your Terraform Enterprise instance
is available at multiple hostnames, use only one of them consistently.
Terraform Cloud responds to API calls at both its current hostname
`app.terraform.io`, and its historical hostname `atlas.hashicorp.com`.

### Credentials Helpers

If you would prefer not to store your API tokens directly in the CLI
configuration as described in the previous section, you can optionally instruct
Terraform to use a different credentials storage mechanism by configuring a
special kind of plugin program called a _credentials helper_.

```hcl
credentials_helper "example" {
  args = []
}
```

`credentials_helper` is a configuration block that can appear at most once
in the CLI configuration. Its label (`"example"` above) is the name of the
credentials helper to use. The `args` argument is optional and allows passing
additional arguments to the helper program, for example if it needs to be
configured with the address of a remote host to access for credentials.

A configured credentials helper will be consulted only to retrieve credentials
for hosts that are _not_ explicitly configured in a `credentials` block as
described in the previous section.
Conversely, this means you can override the credentials returned by the helper
for a specific hostname by writing a `credentials` block alongside the
`credentials_helper` block.

Terraform does not include any credentials helpers in the main distribution.
To learn how to write and install your own credentials helpers to integrate
with existing in-house credentials management systems, see
[the guide to Credentials Helper internals](/docs/internals/credentials-helpers.html).

## Provider Installation

The default way to install provider plugins is from a provider registry. The
origin registry for a provider is encoded in the provider's source address,
like `registry.terraform.io/hashicorp/aws`. For convenience in the common case,
Terraform allows omitting the hostname portion for providers on
`registry.terraform.io`, so you can write shorter public provider addresses like
`hashicorp/aws`.

Downloading a plugin directly from its origin registry is not always
appropriate, though. For example, the system where you are running Terraform
may not be able to access an origin registry due to firewall restrictions
within your organization or your locality.

To allow using Terraform providers in these situations, there are some
alternative options for making provider plugins available to Terraform which
we'll describe in the following sections.

### Explicit Installation Method Configuration

A `provider_installation` block in the CLI configuration allows overriding
Terraform's default installation behaviors, so you can force Terraform to use
a local mirror for some or all of the providers you intend to use.

The general structure of a `provider_installation` block is as follows:

```hcl
provider_installation {
  filesystem_mirror {
    path    = "/usr/share/terraform/providers"
    include = ["example.com/*/*"]
  }
  direct {
    exclude = ["example.com/*/*"]
  }
}
```

Each of the nested blocks inside the `provider_installation` block specifies
one installation method. Each installation method can take both `include`
and `exclude` patterns that specify which providers a particular installation
method can be used for. In the example above, we specify that any provider
whose origin registry is at `example.com` can be installed only from the
filesystem mirror at `/usr/share/terraform/providers`, while all other
providers can be installed only directly from their origin registries.

If you set both `include` and `exclude` for a particular installation
method, the exclusion patterns take priority. For example, including
`registry.terraform.io/hashicorp/*` but also excluding
`registry.terraform.io/hashicorp/dns` will make that installation method apply
to everything in the `hashicorp` namespace with the exception of
`hashicorp/dns`.

As with provider source addresses in the main configuration, you can omit
the `registry.terraform.io/` prefix for providers distributed through the
public Terraform registry, even when using wildcards. For example,
`registry.terraform.io/hashicorp/*` and `hashicorp/*` are equivalent.
`*/*` is a shorthand for `registry.terraform.io/*/*`, not for
`*/*/*`.

The following are the two supported installation method types:

* `direct`: request information about the provider directly from its origin
  registry and download over the network from the location that registry
  indicates. This method expects no additional arguments.

* `filesystem_mirror`: consult a directory on the local disk for copies of
  providers. This method requires the additional argument `path` to indicate
  which directory to look in.

    Terraform expects the given directory to contain a nested directory structure
    where the path segments together provide metadata about the available
    providers. The following two directory structures are supported:

    * Packed layout: `HOSTNAME/NAMESPACE/TYPE/terraform-provider-TYPE_VERSION_TARGET.zip`
      is the distribution zip file obtained from the provider's origin registry.
    * Unpacked layout: `HOSTNAME/NAMESPACE/TYPE/VERSION/TARGET` is a directory
      containing the result of extracting the provider's distribution zip file.

    In both layouts, the `VERSION` is a string like `2.0.0` and the `TARGET`
    specifies a particular target platform using a format like `darwin_amd64`,
    `linux_arm`, `windows_amd64`, etc.

    If you use the unpacked layout, Terraform will attempt to create a symbolic
    link to the mirror directory when installing the provider, rather than
    creating a deep copy of the directory. The packed layout prevents this
    because Terraform must extract the zip file during installation.

    You can include multiple `filesystem_mirror` blocks in order to specify
    several different directories to search.

* `network_mirror`: consult a particular HTTPS server for copies of providers,
  regardless of which registry host they belong to. This method requires the
  additional argument `url` to indicate the mirror base URL, which should
  use the `https:` scheme and end with a trailing slash.

    Terraform expects the given URL to be a base URL for an implementation of
    [the provider network mirror protocol](/docs/internals/provider-network-mirror-protocol.html),
    which is designed to be relatively easy to implement using typical static
    website hosting mechanisms.

~> **Warning:** Don't configure `network_mirror` URLs that you do not trust.
Provider mirror servers are subject to TLS certificate checks to verify
identity, but a network mirror with a TLS certificate can potentially serve
modified copies of upstream providers with malicious content.

Terraform will try all of the specified methods whose include and exclude
patterns match a given provider, and select the newest version available across
all of those methods that matches the version constraint given in each
Terraform configuration. If you have a local mirror of a particular provider
and intend Terraform to use that local mirror exclusively, you must either
remove the `direct` installation method altogether or use its `exclude`
argument to disable its use for specific providers.

### Implied Local Mirror Directories

If your CLI configuration does not include a `provider_installation` block at
all, Terraform produces an _implied_ configuration. The implied configuration
includes a selection of `filesystem_mirror` methods and then the `direct`
method.

The set of directories Terraform can select as filesystem mirrors depends on
the operating system where you are running Terraform:

* **Windows:** `%APPDATA%/terraform.d/plugins` and `%APPDATA%/HashiCorp/Terraform/plugins`
* **Mac OS X:** `$HOME/.terraform.d/plugins`,
  `~/Library/Application Support/io.terraform/plugins`, and
  `/Library/Application Support/io.terraform/plugins`
* **Linux and other Unix-like systems**:`$HOME/.terraform.d/plugins` and
  `terraform/plugins` located within a valid
  [XDG Base Directory](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html)
  data directory such as `$XDG_DATA_HOME/terraform/plugins`.
  Without any XDG environment variables set, Terraform will use
  `~/.local/share/terraform/plugins`,
  `/usr/local/share/terraform/plugins`, and `/usr/share/terraform/plugins`.

If a `terraform.d/plugins` directory exists in the current working directory
then Terraform will also include that directory, regardless of your operating
system.

Terraform will check each of the paths above to see if it exists, and if so
treat it as a filesystem mirror. The directory structure inside each one must
therefore match one of the two structures described for `filesystem_mirror`
blocks in [Explicit Installation Method Configuration](#explicit-installation-method-configuration).

In addition to the zero or more implied `filesystem_mirror` blocks, Terraform
also creates an implied `direct` block. Terraform will scan all of the
filesystem mirror directories to see which providers are placed there and
automatically exclude all of those providers from the implied `direct` block.
(This automatic `exclude` behavior applies only to _implicit_ `direct` blocks;
if you use explicit `provider_installation` you will need to write the intended
exclusions out yourself.)

### Provider Plugin Cache

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
the CLI configuration file. For example:

```hcl
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
still use the configured or implied installation methods to obtain metadata
about which plugins are available, but once a suitable version has been
selected it will first check to see if the chosen plugin is already available
in the cache directory. If so, Terraform will use the previously-downloaded
copy.

If the selected plugin is not already in the cache, Terraform will download
it into the cache first and then copy it from there into the correct location
under your current working directory. When possible Terraform will use
symbolic links to avoid storing a separate copy of a cached plugin in multiple
directories.

The plugin cache directory _must not_ also be one of the configured or implied
filesystem mirror directories, since the cache management logic conflicts with
the filesystem mirror logic when operating on the same directory.

Terraform will never itself delete a plugin from the plugin cache once it has
been placed there. Over time, as plugins are upgraded, the cache directory may
grow to contain several unused versions which you must delete manually.

-> **Note:** The plugin cache directory is not guaranteed to be concurrency
safe. The provider installer's behavior in environments with multiple `terraform
init` calls is undefined. 

### Development Overrides for Provider Developers

-> **Note:** Development overrides work only in Terraform v0.14 and later.
Using a `dev_overrides` block in your CLI configuration will cause Terraform
v0.13 to reject the configuration as invalid.

Normally Terraform verifies version selections and checksums for providers
in order to help ensure that all operations are made with the intended version
of a provider, and that authors can gradually upgrade to newer provider versions
in a controlled manner.

These version and checksum rules are inconvenient when developing a provider
though, because we often want to try a test configuration against a development
build of a provider that doesn't even have an associated version number yet,
and doesn't have an official set of checksums listed in a provider registry.

As a convenience for provider development, Terraform supports a special
additional block `dev_overrides` in `provider_installation` blocks. The contents
of this block effectively override all of the other configured installation
methods, so a block of this type must always appear first in the sequence:

```hcl
provider_installation {

  # Use /home/developer/tmp/terraform-null as an overridden package directory
  # for the hashicorp/null provider. This disables the version and checksum
  # verifications for this provider and forces Terraform to look for the
  # null provider plugin in the given directory.
  dev_overrides {
    "hashicorp/null" = "/home/developer/tmp/terraform-null"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

With development overrides in effect, the `terraform init` command will still
attempt to select a suitable published version of your provider to install and
record in
[the dependency lock file](/docs/language/dependency-lock.html)
for future use, but other commands like
`terraform apply` will disregard the lock file's entry for `hashicorp/null` and
will use the given directory instead. Once your new changes are included in a
published release of the provider, you can use `terraform init -upgrade` to
select the new version in the dependency lock file and remove your development
override.

The override path for a particular provider should be a directory similar to
what would be included in a `.zip` file when distributing the provider. At
minimum that includes an executable file named with a prefix like
`terraform-provider-null`, where `null` is the provider type. If your provider
makes use of other files in its distribution package then you can copy those
files into the override directory too.

You may wish to enable a development override only for shell sessions where
you are actively working on provider development. If so, you can write a
local CLI configuration file with content like the above in your development
directory, perhaps called `dev.tfrc` for the sake of example, and then use the
`TF_CLI_CONFIG_FILE` environment variable to instruct Terraform to use that
localized CLI configuration instead of the default one:

```
export TF_CLI_CONFIG_FILE=/home/developer/tmp/dev.tfrc
```

Development overrides are not intended for general use as a way to have
Terraform look for providers on the local filesystem. If you wish to put
copies of _released_ providers in your local filesystem, see
[Implied Local Mirror Directories](#implied-local-mirror-directories)
or
[Explicit Installation Method Configuration](#explicit-installation-method-configuration)
instead.

This development overrides mechanism is intended as a pragmatic way to enable
smoother provider development. The details of how it behaves, how to
configure it, and how it interacts with the dependency lock file may all evolve
in future Terraform releases, including possible breaking changes. We therefore
recommend using development overrides only temporarily during provider
development work.

## Removed Settings

The following settings are supported in Terraform 0.12 and earlier but are
no longer recommended for use:

* `providers` - a configuration block that allows specifying the locations of
  specific plugins for each named provider. This mechanism is deprecated
  because it is unable to specify a version number and source for each provider.
  See [Provider Installation](#provider-installation) above for the replacement
  of this setting in Terraform 0.13 and later.
