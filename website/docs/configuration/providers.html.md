---
layout: "docs"
page_title: "Providers - Configuration Language"
sidebar_current: "docs-config-providers"
description: |-
  Providers are responsible in Terraform for managing the lifecycle of a resource: create, read, update, delete.
---

# Providers

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Providers](../configuration-0-11/providers.html).

While [resources](./resources.html) are the primary construct
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
the provider itself; in this example both `project` and `region`
are specific to the `google` provider.

The configuration arguments defined by the provider may be assigned using
[expressions](./expressions.html), which can for example
allow them to be parameterized by input variables. However, since provider
configurations must be evaluated in order to perform any resource type action,
provider configurations may refer only to values that are known before
the configuration is applied. In particular, avoid referring to attributes
exported by other resources unless their values are specified directly in the
configuration.

There are also two "meta-arguments" that are defined by Terraform itself
and available for all `provider` blocks:

- [`version`, for constraining the allowed provider versions][inpage-versions]
- [`alias`, for using the same provider with different configurations for different resources][inpage-alias]

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

[inpage-versions]: #provider-versions

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

To constrain the provider version as suggested, add a `required_providers`
block inside a `terraform` block:

```hcl
terraform {
  required_providers {
    aws = "~> 1.0"
  }
}
```

Use [the `terraform providers` command](/docs/commands/providers.html)
to view the specified version constraints for all providers used in the
current configuration.

For more information on the `required_providers` block, see
[Specifying Required Provider Versions](https://www.terraform.io/docs/configuration/terraform.html#specifying-required-provider-versions).

When `terraform init` is re-run with providers already installed, it will
use an already-installed provider that meets the constraints in preference
to downloading a new version. To upgrade to the latest acceptable version
of each provider, run `terraform init -upgrade`. This command also upgrades
to the latest versions of all Terraform modules.

Provider version constraints can also be specified using a `version` argument
within a `provider` block, but that simultaneously declares a new provider
configuration that may cause problems particularly when writing shared modules.
For that reason, we recommend using the `required_providers` block as described
above, and _not_ using the `version` argument within `provider` blocks.
`version` is still supported for compatibility with older Terraform versions.

## `alias`: Multiple Provider Instances

[inpage-alias]: #alias-multiple-provider-instances

You can optionally define multiple configurations for the same provider, and
select which one to use on a per-resource or per-module basis. The primary
reason for this is to support multiple regions for a cloud platform; other
examples include targeting multiple Docker hosts, multiple Consul hosts, etc.

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

### Referring to Alternate Providers

When Terraform needs the name of a provider configuration, it always expects a
reference of the form `<PROVIDER NAME>.<ALIAS>`. In the example above,
`aws.west` would refer to the provider with the `us-west-2` region.

These references are special expressions. Like references to other named
entities (for example, `var.image_id`), they aren't strings and don't need to be
quoted. But they are only valid in specific meta-arguments of `resource`,
`data`, and `module` blocks, and can't be used in arbitrary expressions.

### Selecting Alternate Providers

By default, resources use a default provider configuration inferred from the
first word of the resource type name. For example, a resource of type
`aws_instance` uses the default (un-aliased) `aws` provider configuration unless
otherwise stated.

To select an aliased provider for a resource or data source, set its `provider`
meta-argument to a `<PROVIDER NAME>.<ALIAS>` reference:

```hcl
resource "aws_instance" "foo" {
  provider = aws.west

  # ...
}
```

To select aliased providers for a child module, use its `providers`
meta-argument to specify which aliased providers should be mapped to which local
provider names inside the module:

```hcl
module "aws_vpc" {
  source = "./aws_vpc"
  providers = {
    aws = aws.west
  }
}
```

Modules have some special requirements when passing in providers; see
[Providers within Modules](./modules.html#providers-within-modules)
for more details. In most cases, only _root modules_ should define provider
configurations, with all child modules obtaining their provider configurations
from their parents.

## Third-party Plugins

Anyone can develop and distribute their own Terraform providers. (See
[Writing Custom Providers](/docs/extend/writing-custom-providers.html) for more
about provider development.)

The main way to distribute a provider is via a provider registry, and the main
provider registry is
[part of the public Terraform Registry](https://registry.terraform.io/browse/providers),
along with public shared modules.

Providers distributed via a public registry to not require any special
additional configuration to use, once you know their source addresses. You can
specify both official and third-party source addresses in the
`required_providers` block in your module:

```hcl
terraform {
  required_providers {
    # An example third-party provider. Not actually available.
    example = {
      source = "example.com/examplecorp/example"
    }
  }
}
```

Installing directly from a registry is not appropriate for all situations,
though. If you are running Terraform from a system that cannot access some or
all of the necessary origin registries, you can configure Terraform to obtain
providers from a local mirror instead. For more information, see
[Provider Installation](../commands/cli-config.html#provider-installation)
in the CLI configuration documentation.

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
[the CLI configuration file](/docs/commands/cli-config.html).
