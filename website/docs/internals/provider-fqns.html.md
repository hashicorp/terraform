---
layout: "docs"
page_title: "Provider Fully-Qualified Names"
sidebar_current: "docs-internals-provider-fqns"
description: |-
  Provider Fully-Qualified Names are unambiguous identifiers for providers which allow users to define multiple providers with the same type in a configuration.
---

# Provider Fully-Qualified Name (FQN)

Terraform providers are referenced by their fully-qualified name (FQN), a new
concept introduced in Terraform v0.13 to support the [provider source
attribute](../configuration/terraform.html#inpage-source). Provider FQNs allow
you to use multiple providers with the same type in a configuration
unambiguously. This document explains the concept behind provider FQNs and how
they are used in Terraform.

~> **Advanced Topic!** This page covers technical details of Terraform. You
don't need to understand these details to effectively use Terraform. The details
are documented here for those who wish to learn about them without having to go
spelunking through the source code.

## Prerequisites 

You should be familiar with the following concepts before continuing:

### Terraform Registry namespace 

Modules and providers are stored in *namespaces* in the [Terraform
Registry](https://rigstry.terraform.io), which are similar in concept to GitHub
Organizations. Prior to the Terraform v0.13 release, all provider binaries
available for automatic installation were in the `terraform-providers` registry
namespace and the `terraform-providers` GitHub organization. During the
Terraform v0.13 development cycle, we moved all of the Official HashiCorp
provideresinto the `hashicorp` namespace, and the remaining Trusted Partner
providers will move into their own namespaces over time.  See the [Registry documentation](../registry/index.html) to learn more. 

### Provider Type

The provider type is the designation used in the binary name and resource names.
The terms _provider type_ and _provider name_ have been used interchangably in
the past, but _provider name_ is deprecated in favor of _provider
fully-qualified name_ or _provider FQN_.

Terraform associates each resource type with a provider by taking the first word
of the resource type name (separated by underscores), and so the `"random"`
provider is assumed to be the provider for the resource type name
`random_pet`. The provider type is also found in the name of the provider
binary (in this example, `terraform-provider-random`)

## Provider Fully-Qualified Name (FQN)
A Provider FQN encodes three parts. In a string representation, an FQN is
separated by a forward-slash (`/`). The parts are:

* `hostname`: The `hostname` is the registry host which indexes the provider.
  The default value is HashiCorp's Terraform Registry, `registry.terraform.io`.

* `namespace`: The registry namespace that the provider is in. The default value
  is HashiCorp's namespace, `hashicorp`.

* `type`: The provider type.

The following example shows two providers with the type `random` but different FQNs:

```
registry.terraform.io/hashicorp/random
tfe.example.com/mycorp/random
```

The provider FQN is decoded from the `"source"` atttribute. You are not required
to declare the source for any of HashiCorp's Official Providers: if no source is
given, Terraform will use default values for "sourcehost"
("registry.terraform.io") and "namespace" ("hashicorp").

### Provider local name
It is possible to have multiple providers with the same type in a single
terraform configuration. To avoid ambiguity, you must declare unique local names
for providers. Local names are module-specific, and do not need to be unique or
the same in all of your modules; Terraform references providers by their FQNs.

In the following example, two providers with the type `random` are declared in
`required_providers`:

```hcl
terraform {
    required_providers {
      "random" {
          source = "hashicorp/random"
      }
      "more-random" { 
        source = "myorg/random"
      }
    }
}
```

The local names for the two providers are `random` and `more-random`. The next
example shows how to use these providers in resources:

```hcl
resource "random_pet" "example1" {
  // The local name for "myorg/random" is used here
  provider = "more-random"
}

resource "random_pet" "example2" {
  // You may omit the provider argument when the resource type and provider
  // local name are the same.
  provider = "random"   // optional
}
```

Internally, Terraform references all providers by their FQN. You will see full
FQNs in various outputs, but in some cases a simplified display version is used.
This display version omits the source host when it is the public registry, so
you may see the shortened version `"hashicorp/random"` instead of
`"registry.terraform.io/hashicorp/random"`.
