---
layout: "docs"
page_title: "Provider Requirements - Configuration Language"
---

## Provider Requirements

-> **Note:** If you are using Terraform 0.11 or
earlier, see
[0.11 Configuration Language: Provider Versions](../configuration-0-11/providers.html#provider-versions) instead.

Terraform relies on plugins called "providers" to interact with remote systems.
Each provider offers a set of named
[resource types](resources.html#resource-types-and-arguments), and defines for
each resource type which arguments it accepts, which attributes it exports, and
how changes to resources of that type are actually applied to remote APIs.

You can discover publicly-available providers
[via the Terraform Registry](https://registry.terraform.io/browse/providers).
Which providers you will use will depend on which remote cloud services you are
intending to configure. Additionally, some Terraform providers provide
local-only functionality which is useful to integrate functionality offered by
different providers, such as generating random numbers to help construct
unique resource names.

Once you've selected one or more providers, use a `required_providers` block to
declare them so that Terraform will make them available for use. A provider
dependency consists of both a source location and a version constraint:

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

The `required_providers` block must be nested inside a
[`terraform` block](terraform.html). The `terraform` block can include other
settings too, but we'll only focus on `required_providers` here.

The keys inside the `required_providers` block represent each provider's
[local name](#local-names), which is the unique identifier for a provider within
a particular module. Each item inside the `required_providers` block is an
object expecting the following arguments:

* `source` - the global [source address](#source-addresses) for the
  provider you intend to use, such as `hashicorp/aws`.

* `version` - a [version constraint](#version-constraints) specifying
  which subset of available provider versions the module is compatible with.

-> **Note:** The `required_providers` object syntax described above was added in Terraform v0.13. Previous versions of Terraform used a single string instead of an object, with the string specifying only a version constraint. For example, `mycloud = "~> 1.0"`. Explicit provider source addresses are supported only in Terraform v0.13 and later. If you want to write a module that works with both Terraform v0.12 and v0.13, see [v0.12-Compatible Provider Requirements](#v012-compatible-provider-requirements) below.

### Source Addresses

A provider _source address_ both globally identifies a particular provider and
specifies the primary location from which Terraform can download it.
Source addresses consist of three parts delimited by slashes (`/`), as
follows:

* **Hostname**: the hostname of the Terraform registry that indexes the provider.
  You can omit the hostname portion and its following slash if the provider
  is hosted on [the public Terraform Registry](https://registry.terraform.io/),
  whose hostname is `registry.terraform.io`.

* **Namespace**: an organizational namespace within the specified registry.
  For the public Terraform Registry and Terraform Cloud's private registry,
  this represents the organization that is publishing the provider. This field
  may have other meanings for other registry hosts.

* **Type**: The provider type name, which must be unique within a particular
  namespace on a particular registry host.

For example,
[the official HTTP provider](https://registry.terraform.io/providers/hashicorp/http)
belongs to the `hashicorp` namespace on `registry.terraform.io`, so its
source address can be written as either `registry.terraform.io/hashicorp/http`
or, more commonly, just `hashicorp/http`.

-> **Note**: As a concession for backward compatibility with earlier versions of
Terraform, the `source` argument is actually optional. If you omit it, Terraform
will construct an implied source address by appending the local name to the prefix
`hashicorp/`. For example, a provider dependency with local name `http` that
does not have an explicit `source` will be treated as equivalent to
`hashicorp/http`. We recommend using explicit source addresses for all providers
in modules that require Terraform 0.13 or later, so a future reader of your
module can clearly see exactly which provider is required, without needing to
first understand this default behavior.

### Local Names

Full [source addresses](#source-addresses) are verbose, so the Terraform
language uses them only when declaring dependencies. We associate each required
provider with a module-specific _local name_, which is a short identifier that
will refer to the associated source address within declarations inside a
particular module.

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

The above example declares `mycloud` as the local name for `mycorp/mycloud`
(which is short for `registry.terraform.io/mycorp/mycloud`) in the current
module only. That means we will refer to this provider as `mycloud` elsewhere
in the module, such as in a `provider "mycloud"` block used to create a
[provider configuration](providers.html):

```hcl
provider "mycloud" {
  # ...
}
```

We strongly recommend setting the local name of a provider to match the "type"
portion of its source address, as in the above example. Consistent use of the
provider's canonical type can help avoid the need for readers of the rest of
the module to refer to the `required_providers` block to understand which
provider the module is using.

The one situation where it is reasonable to use a different local name is the
relatively-rare case of having two providers in the same module that have the
same type name. In that case, Terraform requires choosing a unique local name
for each one. In that situation, we recommend to combine the namespace with
the type name to produce a compound local name to disambiguate:

```hcl
terraform {
  required_providers {
    # In the rare situation of using two providers that
    # have the same type name -- "http" in this example --
    # use a compound local name to distinguish them.
    hashicorp_http = {
      source  = "hashicorp/http"
      version = "~> 2.0"
    }
    mycorp_http = {
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
```

### Version Constraints

A [source address](#source-addresses) uniquely identifies a particular
provider, but each provider can have one or more distinct _versions_, allowing
the functionality of the provider to evolve over time. Each provider dependency
you declare should have a [version constraint](./version-constraints.html)
given in the `version` argument.

Each module should at least declare the minimum provider version it is known
to work with, using the `>=` version constraint syntax:

```
terraform {
  required_providers {
    mycloud = {
      source  = "hashicorp/aws"
      version = ">= 1.0"
    }
  }
}
```

A module intended to be used as the root of a configuration -- that is, as the
directory where you'd run `terraform apply` -- should also specify the
_maximum_ provider version it is intended to work with, to avoid accidental
upgrading when new versions are released. The `~>` operator is a convenient
shorthand for allowing only patch releases within a specific minor release:

```
terraform {
  required_providers {
    mycloud = {
      source  = "hashicorp/aws"
      version = "~> 1.0.4"
    }
  }
}
```

_Do not_ use the `~>` or other maximum-version constraints for modules you
intend to reuse across many configurations. All of the version constraints
across all modules in a configuration must work collectively to select a
single version to use, so many modules all specifying maximum version
constraints would require those upper limits to all be updated simultaneously
if one module begins requiring a newer provider version.

The `version` argument is optional. If you omit it, Terraform will accept
_any_ version of the provider as compatible. That's risky for a provider
distributed by a third-party, because they may release a version containing
breaking changes at any time and prevent you from making progress until you
update your configuration. We strongly recommend always specifying a version
constraint, as described above, for every provider your module depends on.

### Built-in Providers

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

### In-house Providers

Some organizations develop their own providers to allow interacting with
proprietary systems, and wish to use these providers from Terraform without
publishing them on the public Terraform Registry.

One option for distributing such a provider is to run an in-house _private_
registry, by implementing
[the provider registry protocol](/docs/internals/provider-registry-protocol.html).

Running an additional service just to distribute a single provider internally
may be undesirable though, so Terraform also supports
[other provider installation methods](https://github.com/hashicorp/terraform/blob/master/website/docs/commands/cli-config.html.markdown#provider-installation),
including placing provider plugins directly in specific directories in the
local filesystem, via _filesystem mirrors_.

All providers must have a [source address](#source-addresses) that includes
(or implies) the hostname of a host registry, but for an in-house provider that
you intend only to distribute from a local filesystem directory you can choose
an artificial hostname in a domain your organization controls and use that to
mark your in-house providers.

For example, if your corporate domain were `example.com` then you might choose
to use `terraform.example.com` as your artificial hostname, even if that
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

The provider plugin executable file might therefore be at the following path,
on a Windows system for the sake of example:

```
terraform.example.com/examplecorp/ourcloud/1.0.0/windows_amd64/terraform-provider-ourcloud.exe
```

If you later decide to switch to using a real private provider registry, rather
than an artifical local hostname, you can deploy the registry server at
`terraform.example.com` and retain the same namespace and type names, in which
case your existing modules will require no changes to locate the same provider
using your registry server instead.

### v0.12-Compatible Provider Requirements

Explicit provider source addresses were introduced with Terraform v0.13, so the
full provider requirements syntax is not supported by Terraform v0.12.

However, in order to allow writing modules that are compatible with both
Terraform v0.12 and v0.13 at the same time, later versions of Terraform v0.12
will accept but ignore the `source` argument in a required_providers block.

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

* Use only providers that are automatically-installable under Terraform v0.12.
  Third-party providers, such as community providers in the Terraform Registry,
  cannot be selected by Terraform v0.12 because it does not support the
  hierarchical source address namespace.

* Ensure that your chosen local name exactly matches the "type" portion of the
  source address given in the `source` argument, such as both being "aws" in
  the examples above, because Terraform v0.12 will use the local name to
  determine which provider plugin to download and install.

* If the provider belongs to the `hashicorp` namespace, as with the
  `hashicorp/aws` provider shown above, omit the `source` argument and allow
  Terraform v0.13 select the `hashicorp` namespace by default.

* Provider type names must always be written in lowercase. Terraform v0.13
  treats provider source addresses as case-insensitive, but Terraform v0.12
  considers its legacy-style provider names to be case-sensitive. Using
  lowercase will ensure that the name is selectable by both Terraform major
  versions.

This compatibility mechanism is provided as a temporary transitional aid only.
When Terraform v0.12 detects the use of the new `source` argument it doesn't
understand, it will emit a warning to alert the user that it is disregarding
the source address given in that argument.
