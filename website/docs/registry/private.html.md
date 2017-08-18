---
layout: "registry"
page_title: "Terraform Registry - Private Registry"
sidebar_current: "docs-registry-private"
description: |-
  Terraform is capable of loading modules from private registries for private modules via Terraform Enterprise.
---

# Private Registry

The registry at [registry.terraform.io](https://registry.terraform.io)
may only host public modules. Terraform is capable of loading modules from
private registries for private modules.

Official private registries are available via [Terraform Enterprise](#).
There are two tiers: Pro and Enterprise. The Pro version is only available
as a SaaS service whereas the Enterprise version is available for private
install. Both versions fully support private registries.

The Terraform project does not provide any free or open source solution
to have a private registry. Terraform only requires that the
[read API](/docs/registry/api.html) to be
available to load modules from a registry. We welcome the community to create
their own private registries by recreating this API.

## Coming Soon

Terraform Enterprise is currently in beta and does not allow open signups.

Terraform Enterprise will be publicly available for self service signup
by the end of 2017. In the mean time, if you're interested in private
registries and being part of the beta, please contact us at
[hello@hashicorp.com](mailto:hello@hashicorp.com).

When Terraform Enterprise is publicly available, the documentation will
be available here.
