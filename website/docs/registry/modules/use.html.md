---
layout: "registry"
page_title: "Finding and Using Modules from the Terraform Registry"
sidebar_current: "docs-registry-use"
description: |-
  The Terraform Registry makes it simple to find and use modules.
---

# Finding and Using Modules

The [Terraform Registry](https://registry.terraform.io) makes it simple to
find and use modules.

## Finding Modules

Every page on the registry has a search field for finding
modules. Enter any type of module you're looking for (examples: "vault",
"vpc", "database") and resulting modules will be listed. The search query
will look at module name, provider, and description to match your search
terms. On the results page, filters can be used further refine search results.

By default, only [verified modules](/docs/registry/modules/verified.html)
are shown in search results. Verified modules are reviewed by HashiCorp to
ensure stability and compatibility. By using the filters, you may view unverified
modules as well.

## Using Modules

The Terraform Registry is integrated directly into Terraform. This makes
it easy to reference any module in the registry. The syntax for referencing
a registry module is `namespace/name/provider`. For example:
`hashicorp/consul/aws`.

When viewing a module on the registry on a tablet or desktop, usage instructions
are shown on the right side. The screenshot below shows where to find these.
You can copy and paste this to get started with any module. Some modules may
have required inputs you must set before being able to use the module.

```hcl
module "consul" {
  source = "hashicorp/consul/aws"
  version = "0.1.0"
}
```

## Module Versions

Each module in the registry is versioned. These versions syntactically must
follow [semantic versioning](http://semver.org/). In addition to pure syntax,
we encourge all modules to follow the full guidelines of semantic versioning.

Terraform since version 0.11 will resolve any provided 
[module version constraints](/docs/modules/usage.html#module-versions) and 
using them is highly recommended to avoid pulling in breaking changes.

Terraform from version 10.6 through to 0.11 had partial support for the registry 
protocol, however will not honour version constraints and always download the
latest version.
