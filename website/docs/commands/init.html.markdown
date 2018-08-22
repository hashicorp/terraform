---
layout: "docs"
page_title: "Command: init"
sidebar_current: "docs-commands-init"
description: |-
  The `terraform init` command is used to initialize a Terraform configuration. This is the first command that should be run for any new or existing Terraform configuration. It is safe to run this command multiple times.
---

# Command: init

The `terraform init` command is used to initialize a working directory
containing Terraform configuration files. This is the first command that should
be run after writing a new Terraform configuration or cloning an existing one
from version control. It is safe to run this command multiple times.

## Usage

Usage: `terraform init [options] [DIR]`

This command performs several different initialization steps in order to
prepare a working directory for use. More details on these are in the
sections below, but in most cases it is not necessary to worry about these
individual steps.

This command is always safe to run multiple times, to bring the working
directory up to date with changes in the configuration. Though subsequent runs
may give errors, this command will never delete your existing configuration or
state.

If no arguments are given, the configuration in the current working directory
is initialized. It is recommended to run Terraform with the current working
directory set to the root directory of the configuration, and omit the `DIR`
argument.

## General Options

The following options apply to all of (or several of) the initialization steps:

* `-input=true` Ask for input if necessary. If false, will error if
  input was required.

* `-lock=false` Disable locking of state files during state-related operations.

* `-lock-timeout=<duration>` Override the time Terraform will wait to acquire
  a state lock. The default is `0s` (zero seconds), which causes immediate
  failure if the lock is already held by another process.

* `-no-color` Disable color codes in the command output.

* `-upgrade` Opt to upgrade modules and plugins as part of their respective
  installation steps. See the sections below for more details.

## Copy a Source Module

By default, `terraform init` assumes that the working directory already
contains a configuration and will attempt to initialize that configuration.

Optionally, init can be run against an empty directory with the
`-from-module=MODULE-SOURCE` option, in which case the given module will be
copied into the target directory before any other initialization steps are
run.

This special mode of operation supports two use-cases:

* Given a version control source, it can serve as a shorthand for checking out
  a configuration from version control and then initializing the work directory
  for it.

* If the source refers to an _example_ configuration, it can be copied into
  a local directory to be used as a basis for a new configuration.

For routine use it is recommended to check out configuration from version
control separately, using the version control system's own commands. This way
it is possible to pass extra flags to the version control system when necessary,
and to perform other preparation steps (such as configuration generation, or
activating credentials) before running `terraform init`.

## Backend Initialization

During init, the root configuration directory is consulted for
[backend configuration](/docs/backends/config.html) and the chosen backend
is initialized using the given configuration settings.

Re-running init with an already-initalized backend will update the working
directory to use the new backend settings. Depending on what changed, this
may result in interactive prompts to confirm migration of workspace states.
The `-force-copy` option suppresses these prompts and answers "yes" to the
migration questions. The `-reconfigure` option disregards any existing
configuration, preventing migration of any existing state.

To skip backend configuration, use `-backend=false`. Note that some other init
steps require an initialized backend, so it is recommended to use this flag only
when the working directory was already previously initialized for a particular
backend.

The `-backend-config=...` option can be used for
[partial backend configuration](/docs/backends/config.html#partial-configuration),
in situations where the backend settings are dynamic or sensitive and so cannot
be statically specified in the configuration file.

## Child Module Installation

During init, the configuration is searched for `module` blocks, and the source
code for referenced [modules](/docs/modules/) is retrieved from the locations
given in their `source` arguments.

Re-running init with modules already installed will install the sources for
any modules that were added to configuration since the last init, but will not
change any already-installed modules. Use `-upgrade` to override this behavior,
updating all modules to the latest available source code.

To skip child module installation, use `-get=false`. Note that some other init
steps can complete only when the module tree is complete, so it's recommended
to use this flag only when the working directory was already previously
initialized with its child modules.

## Plugin Installation

During init, the configuration is searched for both direct and indirect
references to [providers](/docs/configuration/providers.html), and the plugins
for the providers are retrieved from the plugin repository. The downloaded
plugins are installed to a subdirectory of the working directory, and are thus
local to that working directory.

Re-running init with plugins already installed will install plugins only for
any providers that were added to the configuration since the last init. Use
`-upgrade` to additionally update already-installed plugins to the latest
versions that comply with the version constraints given in configuration.

To skip plugin installation, use `-get-plugins=false`.

The automatic plugin installation behavior can be overridden by extracting
the desired providers into a local directory and using the additional option
`-plugin-dir=PATH`. When this option is specified, _only_ the given directory
is consulted, which prevents Terraform from making requests to the plugin
repository or looking for plugins in other local directories. Passing an empty
string to `-plugin-dir` removes any previously recorded paths.

Custom plugins can be used along with automatically installed plugins by
placing them in `terraform.d/plugins/OS_ARCH/` inside the directory being
initialized. Plugins found here will take precedence if they meet the required
constraints in the configuration. The `init` command will continue to
automatically download other plugins as needed.

When plugins are automatically downloaded and installed, by default the
contents are verified against an official HashiCorp release signature to
ensure that they were not corrupted or tampered with during download. It is
recommended to allow Terraform to make these checks, but if desired they may
be disabled using the option `-verify-plugins=false`.

## Running `terraform init` in automation

For teams that use Terraform as a key part of a change management and
deployment pipeline, it can be desirable to orchestrate Terraform runs in some
sort of automation in order to ensure consistency between runs, and provide
other interesting features such as integration with version control hooks.

There are some special concerns when running `init` in such an environment,
including optionally making plugins available locally to avoid repeated
re-installation. For more information, see
[`Running Terraform in Automation`](/guides/running-terraform-in-automation.html).
