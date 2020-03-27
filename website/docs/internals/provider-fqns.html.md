---
layout: "docs"
page_title: "Provider Fully-Qualified Names"
sidebar_current: "docs-internals-provider-fqns"
description: |-
  Provider Fully-Qualified Names are unambiguous names for providers which allow users to define multiple providers with the same type in a configuration.
---

# Provider Fully-Qualified Name (FQN)

Terraform providers are referenced by their fully-qualified name (FQN), a new
concept introduced in Terraform v0.13 to support the [provider source
attribute](link). Provider FQNs allow you to use multiple providers with the
same type in a configuration unambiguously. This document explains the concept
behind provider FQNs and how they are used in Terraform.

~> **Advanced Topic!** This page covers technical details
of Terraform. You don't need to understand these details to
effectively use Terraform. The details are documented here for
those who wish to learn about them without having to go
spelunking through the source code.

## Prerequisites 

You should be familiar with the following concepts before continuing:

### Terraform Registry

modules, providers, provider auto-install(links)

### Terraform Registry namespace 

Modules and providers are stored in *namespaces* in the Terraform Registry,
which are similar in concept to GitHub Organizations. Prior to the Terraform v0.13
release, all provider binaries available for automatic installation were in the
terraform-providers namespace. During the Terraform v0.13 development cycle, we
moved all of the HashiCorp owned providers and partner providers into the
`hashicorp` namespace, and the remaining providers will move into their own
namespaces over time. 

### Provider Type

The provider type is the designation used in the binary name and resource names.
The terms _provider type_ and _provider name_ have been used interchangably in
the past, but _provider name_ is deprecated in favor of _Provider
Fully-Qualified Name_ or _Provider FQN_.

The `type` of a provider is defined by the provider binary name (after
`terraform-provider-`) and is the first part of the name of the resources
provided by the provider.

Consider HashiCorp's `random` provider as an example:

Binary name:
`terraform-provider-random_v1.0.0_x4`

Resources:
`random_pet`
`random_id`

## Provider Fully-Qualified Name (FQN)
A provider FQN is made up of the following parts:

```
sourcehost/namespace/type
```

Both of the following examples refer to providers with the type `random` but different FQNs:

```
registry.terraform.io/hashicorp/random
tfe.example.com/mycorp/random
```

### Provider localname
It is possible to have multiple providers with the same type in a single
terraform configuration. To avoid ambiguity, you must declare unique localnames
for providers. These names do not need to be either unique or the same across
modules; terraform references providers by their FQNs regardless of the localname.

In the following example, two providers with the type `random` are declared in `required_providers`:

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

The localnames for the two providers are "random" and "more-random". The next example shows how to use these providers in resources:

```hcl
resource "random_pet" "example1" {
  // The local name for "myorg/random" is used here
  provider = "more-random"
}

resource "random_pet" "example2" {
  // You may omit the provider argument when the resource type and provider
  // local name are the same.
  provider = "random"
}
```

Inside terraform, providers are always referenced by their FQN while provider
configuration remains aware of the localname.
