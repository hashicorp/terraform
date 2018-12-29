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

- `credentials` — provides credentials for use with Terraform Enterprise.
    Terraform uses this when performing remote operations or state access with
    the [remote backend](../backends/types/remote.html) and when accessing
    Terraform Enterprise's [private module registry.](/docs/enterprise/registry/index.html)

    This setting is a repeatable block, where the block label is a hostname
    (either `app.terraform.io` or the hostname of your private install) and
    the block body contains a `token` attribute. Whenever Terraform accesses
    state, modules, or remote operations from that hostname, it will
    authenticate with that API token.

    ``` hcl
    credentials "app.terraform.io" {
      token = "xxxxxx.atlasv1.zzzzzzzzzzzzz"
    }
    ```

    ~> **Important:** The token provided here must be a
    [user API token](/docs/enterprise/users-teams-organizations/users.html#api-tokens),
    and not a team or organization token.

    -> **Note:** The credentials hostname must match the hostname in your module
    sources and/or backend configuration. If your Terraform Enterprise instance
    is available at multiple hostnames, use one of them consistently. (The SaaS
    version of Terraform Enterprise responds to API calls at both its newer
    hostname, app.terraform.io, and its historical hostname,
    atlas.hashicorp.com.)

## Deprecated Settings

The following settings are supported for backward compatibility but are no
longer recommended for use:

* `providers` - a configuration block that allows specifying the locations of
  specific plugins for each named provider. This mechanism is deprecated
  because it is unable to specify a version number for each plugin, and thus
  it does not co-operate with the plugin versioning mechansim. Instead,
  place the plugin executable files in
  [the third-party plugins directory](/docs/configuration/providers.html#third-party-plugins).
