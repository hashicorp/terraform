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

Official private registries are available via [Terraform Enterprise](https://www.hashicorp.com/products/terraform).
There are two tiers: Pro and Enterprise. The Pro version is only available
as a SaaS service whereas the Enterprise version is available for private
install. Both versions fully support private registries.

Terraform interacts with module registries using [the registry API](/docs/registry/api.html).
The Terraform open source project does not provide a server implementation, but
we welcome community members to create their own private registries by following
the published protocol.

Modules can alternatively be referenced
[directly from version control and other sources](/docs/modules/sources.html),
but only registry modules support certain features such as
[version constraints](/docs/modules/usage.html#module-versions).

## Private Registry Module Sources

Public Terraform Registry modules have source strings of the form
`namespace/name/provider`. Private registries -- whether integrated into
Terraform Enterprise or via a third-party implementation -- require an
additional hostname prefix:

```hcl
module "example" {
  source = "example.com/namespace/name/provider"
}
```

Private registry module sources are supported in Terraform v0.11.0 and
newer.

## Coming Soon

Terraform Enterprise will be publicly available for self service signup
soon. In the mean time, if you're interested in private
registries and being part of the beta, please contact us at
[hello@hashicorp.com](mailto:hello@hashicorp.com).

When Terraform Enterprise is publicly available, Private Registry documentation
will be available here.
