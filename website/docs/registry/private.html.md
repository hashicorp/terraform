---
layout: "registry"
page_title: "Terraform Registry - Private Registry"
sidebar_current: "docs-registry-private"
description: |-
  Terraform can load private modules from private registries via Terraform Cloud.
---

# Private Registries

The registry at [registry.terraform.io](https://registry.terraform.io)
only hosts public modules, but most organizations have some modules that
can't, shouldn't, or don't need to be public.

You can load private modules [directly from version control and other
sources](/docs/modules/sources.html), but those sources don't support [version
constraints](/docs/configuration/modules.html#module-versions) or a browsable
marketplace of modules, both of which are important for enabling a
producers-and-consumers content model in a large organization.

If your organization is specialized enough that teams frequently use modules
created by other teams, you will benefit from a private module registry.

## Terraform Cloud's Private Registry

[Terraform Cloud](https://www.hashicorp.com/products/terraform)
includes a private module registry, available at both Pro and Premium tiers.

It uses the same VCS-backed tagged release workflow as the Terraform Registry,
but imports modules from your private VCS repos (on any of Terraform Cloud's supported VCS
providers) instead of requiring public GitHub repos. You can seamlessly
reference private modules in your Terraform configurations (just include a
hostname in the module source), and Terraform Cloud's UI provides a searchable marketplace
of private modules to help your users find the code they need.

[Terraform Cloud's private module registry is documented here.](/docs/cloud/registry/index.html)

## Other Private Registries

Terraform can use versioned modules from any service that implements
[the registry API](/docs/registry/api.html).
The Terraform open source project does not provide a server implementation, but
we welcome community members to create their own private registries by following
the published protocol.

