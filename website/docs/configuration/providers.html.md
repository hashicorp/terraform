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

Every resource in Terraform is mapped to a provider based
on longest-prefix matching. For example the `aws_instance`
resource type would map to the `aws` provider (if that exists).

Most providers require some sort of configuration to provide
authentication information, endpoint URLs, etc. Provider configuration
blocks are a way to set this information globally for all
matching resources.

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

The `provider` block configures the provider of the given `NAME`.
Multiple provider blocks can be used to configure multiple providers.

Terraform matches providers to resources by matching two criteria.
Both criteria must be matched for a provider to manage a resource:

- They must share a common prefix. Longest matching prefixes are tried first.
  For example, `aws_instance` would choose the `aws` provider.

- The provider must report that it supports the given resource type. Providers
  internally tell Terraform the list of resources they support.

Within the block (the `{ }`) is configuration for the resource.
The configuration is dependent on the type, and is documented
[for each provider](/docs/providers/index.html).

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

You can define multiple instances of the same provider in order to support
multiple regions, multiple hosts, etc. The primary use case for this is
utilizing multiple cloud regions. Other use cases include targeting multiple
Docker hosts, multiple Consul hosts, etc.

To define multiple provider instances, repeat the provider configuration
multiple times, but set the `alias` field and name the provider. For
example:

```hcl
# The default provider
provider "aws" {
  # ...
}

# West coast region
provider "aws" {
  alias  = "west"
  region = "us-west-2"
}
```

After naming a provider, you reference it in resources with the `provider`
field:

```hcl
resource "aws_instance" "foo" {
  provider = "aws.west"

  # ...
}
```

If a provider isn't specified, then the default provider configuration
is used (the provider configuration with no `alias` set). The value of the
`provider` field is `TYPE.ALIAS`, such as "aws.west" above.

## Syntax

The full syntax is:

```text
provider NAME {
  CONFIG ...
  [alias = ALIAS]
}
```

where `CONFIG` is:

```text
KEY = VALUE

KEY {
  CONFIG
}
```

## Interpolation
Providers support [interpolation syntax](/docs/configuration/interpolation.html) allowing dynamic configuration at run time.

```hcl
provider "aws" {
  region = "${var.aws_region}"
}
```

An exception to this is the special `version` attribute that applies to all `provider` blocks for specifying [provider versions](#provider-versions); interpolation is not supported for provider versions since provider compatibility is a property of the configuration rather than something dynamic, and provider plugin installation happens too early for variables to be resolvable in this context.

-> **NOTE:** Because providers are one of the first things loaded when Terraform parses the graph, it is not possible to use the output from modules or resources as inputs to the provider. At this time, only [variables](/docs/configuration/variables.html) and [data sources](/docs/configuration/data-sources.html), including [remote state](/docs/providers/terraform/d/remote_state.html) may be used in an interpolation inside a provider stanza.
