---
layout: "language"
page_title: "Provider Requirements - Configuration Language"
---

# Provider Requirements

-> **Note:** This page is about a feature of Terraform 0.13 and later; it also
describes how to use the more limited version of that feature that was available
in Terraform 0.12. If you are using Terraform 0.11 or earlier, see
[0.11 Configuration Language: Provider Versions](../configuration-0-11/providers.html#provider-versions) instead.

Terraform relies on plugins called "providers" to interact with remote systems.

Terraform configurations must declare which providers they require, so that
Terraform can install and use them. Additionally, some providers require
configuration (like endpoint URLs or cloud regions) before they can be used.

- This page documents how to declare providers so Terraform can install them.

- The [Provider Configuration](./providers.html) page documents how to configure
  settings for providers.

## About Providers

Providers are plugins. They are released on a separate rhythm from Terraform
itself, and each provider has its own series of version numbers.

Each provider plugin offers a set of
[resource types](resources.html#resource-types-and-arguments), and defines for
each resource type which arguments it accepts, which attributes it exports, and
how changes to resources of that type are actually applied to remote APIs.

Most providers configure a specific infrastructure platform (either cloud or
self-hosted). Providers can also offer local utilities for tasks like
generating random numbers for unique resource names.

The [Terraform Registry](https://registry.terraform.io/browse/providers)
is the main directory of publicly available Terraform providers, and hosts
providers for most major infrastructure platforms. You can also write and
distribute your own Terraform providers, for public or private use.

> **Hands-on:** If you're interested in developing your own Terraform providers, try the [Call APIs with Terraform Providers](https://learn.hashicorp.com/collections/terraform/providers?utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS) collection on HashiCorp Learn.

### Provider Installation

Terraform finds and installs providers when
[initializing a working directory](/docs/commands/init.html). It can
automatically download providers from a Terraform registry, or load them from a
local mirror or cache.

When you add a new provider to a configuration, Terraform must install the
provider in order to use it. If you are using a persistent working directory,
you can run `terraform init` again to install new providers.

Providers downloaded by `terraform init` are only installed for the current
working directory; other working directories can have their own installed
provider plugins. To help ensure that each working directory will use the same
selected versions, `terraform init` records its version selections in
your configuration's [dependency lock file](dependency-lock.html), named
`.terraform.lock.hcl` and will always make those same selections unless
you run `terraform init -upgrade` to update them.

To save time and bandwidth, Terraform supports an optional plugin cache. You can
enable the cache using the `plugin_cache_dir` setting in
[the CLI configuration file](/docs/commands/cli-config.html).

For more information about provider installation, see
[the `terraform init` command](/docs/commands/init.html).

## Requiring Providers

Each Terraform module must declare which providers it requires, so that
Terraform can install and use them. Provider requirements are declared in a
`required_providers` block.

A provider requirement consists of a local name, a source location, and a
version constraint:

```hcl
terraform {
  required_providers {
    mycloud = {
      source  = "mycorp/mycloud"
      version = "~> 1.0"
    }
  }
}
```

The `required_providers` block must be nested inside the top-level
[`terraform` block](terraform.html) (which can also contain other settings).

Each argument in the `required_providers` block enables one provider. The key
determines the provider's [local name](#local-names) (its unique identifier
within this module), and the value is an object with the following elements:

* `source` - the global [source address](#source-addresses) for the
  provider you intend to use, such as `hashicorp/aws`.

* `version` - a [version constraint](#version-constraints) specifying
  which subset of available provider versions the module is compatible with.

-> **Note:** The `name = { source, version }` syntax for `required_providers`
was added in Terraform v0.13. Previous versions of Terraform used a version
constraint string instead of an object (like `mycloud = "~> 1.0"`), and had no
way to specify provider source addresses. If you want to write a module that
works with both Terraform v0.12 and v0.13, see [v0.12-Compatible Provider
Requirements](#v0-12-compatible-provider-requirements) below.

## Names and Addresses

Each provider has two identifiers:

- A unique _source address,_ which is only used when requiring a provider.
- A _local name,_ which is used everywhere else in a Terraform module.

-> **Note:** Prior to Terraform 0.13, providers only had local names, since
Terraform could only automatically download providers distributed by HashiCorp.

### Local Names

Local names are module-specific, and are assigned when requiring a provider.
Local names must be unique per-module.

Outside of the `required_providers` block, Terraform configurations always refer
to providers by their local names. For example, the following configuration
declares `mycloud` as the local name for `mycorp/mycloud`, then uses that local
name when [configuring the provider](./providers.html):

```hcl
terraform {
  required_providers {
    mycloud = {
      source  = "mycorp/mycloud"
      version = "~> 1.0"
    }
  }
}

provider "mycloud" {
  # ...
}
```

Users of a provider can choose any local name for it. However, nearly every
provider has a _preferred local name,_ which it uses as a prefix for all of its
resource types. (For example, resources from `hashicorp/aws` all begin with
`aws`, like `aws_instance` or `aws_security_group`.)

Whenever possible, you should use a provider's preferred local name. This makes
your configurations easier to understand, and lets you omit the `provider`
meta-argument from most of your resources. (If a resource doesn't specify which
provider configuration to use, Terraform interprets the first word of the
resource type as a local provider name.)

### Source Addresses

A provider's source address is its global identifier. It also specifies the
primary location where Terraform can download it.

Source addresses consist of three parts delimited by slashes (`/`), as
follows:

`[<HOSTNAME>/]<NAMESPACE>/<TYPE>`

* **Hostname** (optional): The hostname of the Terraform registry that
  distributes the provider. If omitted, this defaults to
  `registry.terraform.io`, the hostname of
  [the public Terraform Registry](https://registry.terraform.io/).

* **Namespace:** An organizational namespace within the specified registry.
  For the public Terraform Registry and for Terraform Cloud's private registry,
  this represents the organization that publishes the provider. This field
  may have other meanings for other registry hosts.

* **Type:** A short name for the platform or system the provider manages. Must
  be unique within a particular namespace on a particular registry host.

    The type is usually the provider's preferred local name. (There are
    exceptions; for example,
    [`hashicorp/google-beta`](https://registry.terraform.io/providers/hashicorp/google-beta/latest)
    is an alternate release channel for `hashicorp/google`, so its preferred
    local name is `google`. If in doubt, check the provider's documentation.)

For example,
[the official HTTP provider](https://registry.terraform.io/providers/hashicorp/http)
belongs to the `hashicorp` namespace on `registry.terraform.io`, so its
source address is `registry.terraform.io/hashicorp/http` or, more commonly, just
`hashicorp/http`.

The source address with all three components given explicitly is called the
provider's _fully-qualified address_. You will see fully-qualified address in
various outputs, like error messages, but in most cases a simplified display
version is used. This display version omits the source host when it is the
public registry, so you may see the shortened version `"hashicorp/random"` instead
of `"registry.terraform.io/hashicorp/random"`.


-> **Note:** If you omit the `source` argument when requiring a provider,
Terraform uses an implied source address of
`registry.terraform.io/hashicorp/<LOCAL NAME>`. This is a backward compatibility
feature to support the transition to Terraform 0.13; in modules that require
0.13 or later, we recommend using explicit source addresses for all providers.

### Handling Local Name Conflicts

Whenever possible, we recommend using a provider's preferred local name, which
is usually the same as the "type" portion of its source address.

However, it's sometimes necessary to use two providers with the same preferred
local name in the same module, usually when the providers are named after a
generic infrastructure type. Terraform requires unique local names for each
provider in a module, so you'll need to use a non-preferred name for at least
one of them.

When this happens, we recommend combining each provider's namespace with
its type name to produce compound local names with a dash:

```hcl
terraform {
  required_providers {
    # In the rare situation of using two providers that
    # have the same type name -- "http" in this example --
    # use a compound local name to distinguish them.
    hashicorp-http = {
      source  = "hashicorp/http"
      version = "~> 2.0"
    }
    mycorp-http = {
      source  = "mycorp/http"
      version = "~> 1.0"
    }
  }
}

# References to these providers elsewhere in the
# module will use these compound local names.
provider "mycorp_http" {
  # ...
}

data "http" "example" {
  provider = hashicorp_http
  #...
}
```

Terraform won't be able to guess either provider's name from its resource types,
so you'll need to specify a `provider` meta-argument for every affected
resource. However, readers and maintainers of your module will be able to easily
understand what's happening, and avoiding confusion is much more important than
avoiding typing.

## Version Constraints

Each provider plugin has its own set of available versions, allowing the
functionality of the provider to evolve over time. Each provider dependency you
declare should have a [version constraint](./version-constraints.html) given in
the `version` argument so Terraform can select a single version per provider
that all modules are compatible with.

The `version` argument is optional; if omitted, Terraform will accept any
version of the provider as compatible. However, we strongly recommend specifying
a version constraint for every provider your module depends on.

### Best Practices for Provider Versions

Each module should at least declare the minimum provider version it is known
to work with, using the `>=` version constraint syntax:

```hcl
terraform {
  required_providers {
    mycloud = {
      source  = "hashicorp/aws"
      version = ">= 1.0"
    }
  }
}
```

A module intended to be used as the root of a configuration — that is, as the
directory where you'd run `terraform apply` — should also specify the
_maximum_ provider version it is intended to work with, to avoid accidental
upgrades to incompatible new versions. The `~>` operator is a convenient
shorthand for allowing only patch releases within a specific minor release:

```hcl
terraform {
  required_providers {
    mycloud = {
      source  = "hashicorp/aws"
      version = "~> 1.0.4"
    }
  }
}
```

Do not use `~>` (or other maximum-version constraints) for modules you intend to
reuse across many configurations, even if you know the module isn't compatible
with certain newer versions. Doing so can sometimes prevent errors, but more
often it forces users of the module to update many modules simultaneously when
performing routine upgrades. Specify a minimum version, document any known
incompatibilities, and let the root module manage the maximum version.

## Built-in Providers

While most Terraform providers are distributed separately as plugins, there
is currently one provider that is built in to Terraform itself, which
provides
[the `terraform_remote_state` data source](/docs/providers/terraform/d/remote_state.html).

Because this provider is built in to Terraform, you don't need to declare it
in the `required_providers` block in order to use its features. However, for
consistency it _does_ have a special provider source address, which is
`terraform.io/builtin/terraform`. This address may sometimes appear in
Terraform's error messages and other output in order to unambiguously refer
to the built-in provider, as opposed to a hypothetical third-party provider
with the type name "terraform".

There is also an existing provider with the source address
`hashicorp/terraform`, which is an older version of the now-built-in provider
that was used by older versions of Terraform. `hashicorp/terraform` is not
compatible with Terraform v0.11 or later and should never be declared in a
`required_providers` block.

## In-house Providers

Anyone can develop and distribute their own Terraform providers. See
the [Call APIs with Terraform Providers](https://learn.hashicorp.com/collections/terraform/providers?utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS)
collection on HashiCorp Learn for more
about provider development.

Some organizations develop their own providers to configure
proprietary systems, and wish to use these providers from Terraform without
publishing them on the public Terraform Registry.

One option for distributing such a provider is to run an in-house _private_
registry, by implementing
[the provider registry protocol](/docs/internals/provider-registry-protocol.html).

Running an additional service just to distribute a single provider internally
may be undesirable, so Terraform also supports
[other provider installation methods](/docs/commands/cli-config.html#provider-installation),
including placing provider plugins directly in specific directories in the
local filesystem, via _filesystem mirrors_.

All providers must have a [source address](#source-addresses) that includes
(or implies) the hostname of a registry, but that hostname does not need to
provide an actual registry service. For in-house providers that you intend to
distribute from a local filesystem directory, you can use an arbitrary hostname
in a domain your organization controls.

For example, if your corporate domain were `example.com` then you might choose
to use `terraform.example.com` as your placeholder hostname, even if that
hostname doesn't actually resolve in DNS. You can then choose any namespace and
type you wish to represent your in-house provider under that hostname, giving
a source address like `terraform.example.com/examplecorp/ourcloud`:

```hcl
terraform {
  required_providers {
    mycloud = {
      source  = "terraform.example.com/examplecorp/ourcloud"
      version = ">= 1.0"
    }
  }
}
```

To make version 1.0.0 of this provider available for installation from the
local filesystem, choose one of the
[implied local mirror directories](/docs/commands/cli-config.html#implied-local-mirror-directories)
and create a directory structure under it like this:

```
terraform.example.com/examplecorp/ourcloud/1.0.0
```

Under that `1.0.0` directory, create one additional directory representing the
platform where you are running Terraform, such as `linux_amd64` for Linux on
an AMD64/x64 processor, and then place the provider plugin executable and any
other needed files in that directory.

Thus, on a Windows system, the provider plugin executable file might be at the
following path:

```
terraform.example.com/examplecorp/ourcloud/1.0.0/windows_amd64/terraform-provider-ourcloud.exe
```

If you later decide to switch to using a real private provider registry rather
than distribute binaries out of band, you can deploy the registry server at
`terraform.example.com` and retain the same namespace and type names, in which
case your existing modules will require no changes to locate the same provider
using your registry server.

## v0.12-Compatible Provider Requirements

Explicit provider source addresses were introduced with Terraform v0.13, so the
full provider requirements syntax is not supported by Terraform v0.12.

However, in order to allow writing modules that are compatible with both
Terraform v0.12 and v0.13, versions of Terraform between v0.12.26 and v0.13
will accept but ignore the `source` argument in a `required_providers` block.

Consider the following example written for Terraform v0.13:

```hcl
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 1.0"
    }
  }
}
```

Terraform v0.12.26 will accept syntax like the above but will understand it
in the same way as the following v0.12-style syntax:

```hcl
terraform {
  required_providers {
    aws = "~> 1.0"
  }
}
```

In other words, Terraform v0.12.26 ignores the `source` argument and considers
only the `version` argument, using the given [local name](#local-names) as the
un-namespaced provider type to install.

When writing a module that is compatible with both Terraform v0.12.26 and
Terraform v0.13.0 or later, you must follow the following additional rules so
that both versions will select the same provider to install:

* Use only providers that can be automatically installed by Terraform v0.12.
  Third-party providers, such as community providers in the Terraform Registry,
  cannot be selected by Terraform v0.12 because it does not support the
  hierarchical source address namespace.

* Ensure that your chosen local name exactly matches the "type" portion of the
  source address given in the `source` argument, such as both being "aws" in
  the examples above, because Terraform v0.12 will use the local name to
  determine which provider plugin to download and install.

* If the provider belongs to the `hashicorp` namespace, as with the
  `hashicorp/aws` provider shown above, omit the `source` argument and allow
  Terraform v0.13 to select the `hashicorp` namespace by default.

* Provider type names must always be written in lowercase. Terraform v0.13
  treats provider source addresses as case-insensitive, but Terraform v0.12
  considers its legacy-style provider names to be case-sensitive. Using
  lowercase will ensure that the name is selectable by both Terraform major
  versions.

This compatibility mechanism is provided as a temporary transitional aid only.
When Terraform v0.12 detects a use of the new `source` argument it doesn't
understand, it will emit a warning to alert the user that it is disregarding
the source address given in that argument.
