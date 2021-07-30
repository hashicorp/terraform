---
layout: "docs"
page_title: "Managing Plugins - Terraform CLI"
description: "Commands to install, configure, and show information about providers. Also commands to reduce install effort in air-gapped environments."
---

# Managing Plugins

Terraform relies on plugins called "providers" in order to manage various types
of resources. (For more information about providers, see
[Providers](/docs/language/providers/index.html) in the Terraform
language docs.)

-> **Note:** Providers are currently the only plugin type most Terraform users
will interact with. Terraform also supports third-party provisioner plugins, but
we discourage their use.

Terraform downloads and/or installs any providers
[required](/docs/language/providers/requirements.html) by a configuration
when [initializing](/docs/cli/init/index.html) a working directory. By default,
this works without any additional interaction but requires network access to
download providers from their source registry.

You can configure Terraform's provider installation behavior to limit or skip
network access, and to enable use of providers that aren't available via a
networked source. Terraform also includes some commands to show information
about providers and to reduce the effort of installing providers in airgapped
environments.

## Configuring Plugin Installation

Terraform's configuration file includes options for caching downloaded plugins,
or explicitly specifying a local or HTTPS mirror to install plugins from. For
more information, see [CLI Config File](/docs/cli/config/config-file.html).

## Getting Plugin Information

Use the [`terraform providers`](/docs/cli/commands/providers.html) command to get information
about the providers required by the current working directory's configuration.

Use the [`terraform version`](/docs/cli/commands/version.html) command (or
`terraform -version`) to show the specific provider versions installed for the
current working directory.

Use the [`terraform providers schema`](/docs/cli/commands/providers/schema.html) command to
get machine-readable information about the resources and configuration options
offered by each provider.

## Managing Plugin Installation

Use the [`terraform providers mirror`](/docs/cli/commands/providers/mirror.html) command to
download local copies of every provider required by the current working
directory's configuration. This directory will use the nested directory layout
that Terraform expects when installing plugins from a local source, so you can
transfer it directly to an airgapped system that runs Terraform.

Use the [`terraform providers lock`](/docs/cli/commands/providers/lock.html) command
to update the lock file that Terraform uses to ensure predictable runs when
using ambiguous provider version constraints.
