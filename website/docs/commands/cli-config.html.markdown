---
layout: "docs"
page_title: "CLI Configuration"
sidebar_current: "docs-commands-cli-config"
description: |-
  The general behavior of the Terraform CLI can be customized using the CLI
  configuration file.
---

# CLI Configuration File

The CLI configuration file allows customization of some behaviors of the
Terraform CLI in general. This is separate from
[your infrastructure configuration](/docs/configuration/index.html), and
provides per-user customization that applies regardless of which working
directory Terraform is being applied to.

For example, the CLI configuration file can be used to activate a shared
plugin cache directory that allows provider plugins to be shared between
different working directories, as described in more detail below.

The configuration is placed in a single file whose location depends on the
host operating system:

* On Windows, the file must be named named `terraform.rc` and placed
  in the relevant user's "Application Data" directory. The physical location
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

* `disable_checkpoint` - when set to `true`, disables
  [upgrade and security bulletin checks](/docs/commands/index.html#upgrade-and-security-bulletin-checks)
  that require reaching out to HashiCorp-provided network services.

* `disable_checkpoint_signature` - when set to `true`, allows the upgrade and
  security bulletin checks described above but disables the use of an anonymous
  id used to de-duplicate warning messages.

* `plugin_cache_dir` - enables
  [plugin caching](/docs/configuration/providers.html#provider-plugin-cache)
  and specifies, as a string, the location of the plugin cache directory.

## Deprecated Settings

The following settings are supported for backward compatibility but are no
longer recommended for use:

* `providers` - a configuration block that allows specifying the locations of
  specific plugins for each named provider. This mechanism is deprecated
  because it is unable to specify a version number for each plugin, and thus
  it does not co-operate with the plugin versioning mechansim. Instead,
  place the plugin executable files in
  [the third-party plugins directory](/docs/configuration/providers.html#third-party-plugins).
