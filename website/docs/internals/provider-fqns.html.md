---
layout: "docs"
page_title: "Provider Fully-Qualified Names"
sidebar_current: "docs-internals-provider-fqns"
description: |-
  Provider Fully-Qualified Names are unambiguous names for providers which allow users to define multiple providers with the same type in a configuration.
---

# Provider Fully-Qualified Name (FQN)

Terraform providers are referenced by their fully-qualified name (FQN), a new
concept introduced in Terraform v0.13 to support the [provider source attribute](link). Provider FQNs allow you to use multiple providers with the same type in a configuration unambiguously. This document explains the concept behind provider FQNs and how they are used in Terraform. 

~> **Advanced Topic!** This page covers technical details
of Terraform. You don't need to understand these details to
effectively use Terraform. The details are documented here for
those who wish to learn about them without having to go
spelunking through the source code.

## Prerequisites 
You should be familiar with the following concepts before reading about provider FQNs:

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
The terms _provider type_ and _provider name_ have been used interchangably in the past, but _provider name_ is deprecated in favor of _Provider Fully Qualified Name_. 

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
[sourcehost]/[namespace]/type
```

Both of the following examples refer to providers with the type `random` but different FQNs:

```
registry.terraform.io/hashicorp/random
tfe.example.com/mycorp/random
```

## Security Considerations 

### HashiCorp provider binaries 
HashiCorp provider binaries are signed with a gpg signing key and verified against a hard-coded public key stored in the terraform binary.

### Partner provider binaries  
Partner providers are signed with their own signing keys. The public registry response includes the public keys so terraform can confirm that the binary matches the registry response. To protect against man-in-the-middle attacks, the partner keys are signed by HashiCorp so Terraform can confirm the validity of the partner signing key against a public key stored in the terraform binary. 

### Community provider binaries 
Community provider binaries are signed and the terraform registry response includes the public keys so terraform can confirm that the binary matches the registry response. However these keys are not signed by HashiCorp and the provider code and binaries have not been vetter by HashiCorp. Proceed at your own risk. 

References 
Provider Source Doc link 
Provider auto install doc link
provider discovery on disk link 
Provider Tiers, more info
