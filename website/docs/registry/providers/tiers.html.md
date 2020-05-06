---
layout: "registry"
page_title: "Terraform Registry - Provider Tiers
sidebar_current: "docs-registry-provider-tiers
description: |-
  Published Provider tiers in the Terraform Registry
---

# Provider Tiers

There are three tiers of providers in the Terraform Registry:

* **Official Providers** - are built, signed, and supported by HashiCorp. Official Providers can typically be used without providing
  provider source information in your Terraform configuration.
* **Partner Providers** - are built, signed, and supported by a third party. HashiCorp has verified the ownership of the private
  key and we provide a chain of trust to the CLI to verify this programatically. To use Partner Providers in your Terraform
  configuration, you need to specify the provider source, typically this is the namespace and name to download from the registry.
* **Community Providers** - are built, signed, and supported by a third party. HashiCorp does not provide a verification or chain
  of trust for the signing. You will want to obtain and validate fingerprints manually if you want to ensure you are using a
  binary you can trust.
