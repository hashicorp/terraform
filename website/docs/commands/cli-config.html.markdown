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
[your infrastructure configuration](/docs/configuration/index.html).

## Location

The configuration is placed in a single file whose location depends on the
host operating system:

* On Windows, the file must be named named `terraform.rc` and placed
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
using the `TF_CLI_CONFIG_FILE` [environment variable](/docs/commands/environment-variables.html).

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

- `disable_checkpoint` — when set to `true`, disables
  [upgrade and security bulletin checks](/docs/commands/index.html#upgrade-and-security-bulletin-checks)
  that require reaching out to HashiCorp-provided network services.

- `disable_checkpoint_signature` — when set to `true`, allows the upgrade and
  security bulletin checks described above but disables the use of an anonymous
  id used to de-duplicate warning messages.

- `plugin_cache_dir` — enables
  [plugin caching](/docs/configuration/providers.html#provider-plugin-cache)
  and specifies, as a string, the location of the plugin cache directory.

- `credentials` - configures credentials for use with Terraform Cloud or
  Terraform Enterprise. See [Credentials](#credentials) below for more
  information.

- `credentials_helper` - configures an external helper program for the storage
  and retrieval of credentials for Terraform Cloud or Terraform Enterprise.
  See [Credentials Helpers](#credentials-helpers) below for more information.

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

If you are running the Terraform CLI interactively on a computer that is capable
of also running a web browser, you can optionally obtain credentials and save
them in the CLI configuration automatically using
[the `terraform login` command](./login.html).

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

## Deprecated Settings

The following settings are supported for backward compatibility but are no
longer recommended for use:

* `providers` - a configuration block that allows specifying the locations of
  specific plugins for each named provider. This mechanism is deprecated
  because it is unable to specify a version number for each plugin, and thus
  it does not co-operate with the plugin versioning mechanism. Instead,
  place the plugin executable files in
  [the third-party plugins directory](/docs/configuration/providers.html#third-party-plugins).
