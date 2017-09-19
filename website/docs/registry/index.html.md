---
layout: "registry"
page_title: "Terraform Registry"
sidebar_current: "docs-registry-home"
description: |-
  The Terraform Registry is a repository of modules written by the Terraform community.
---

# Terraform Registry

The [Terraform Registry](https://registry.terraform.io) is a repository
of modules written by the Terraform community. The registry can be used to
help you get started with Terraform more quickly, see examples of how
Terraform is written, and find pre-made modules for infrastructure components
you require.

The Terraform Registry is integrated directly into Terraform to make
consuming modules easy. The following example shows how easy it is to
build a fully functional [Consul](https://www.consul.io) cluster using the
[Consul module for AWS](https://registry.terraform.io/modules/hashicorp/consul/aws).

```hcl
module "consul" {
	source = "hashicorp/consul/aws"
}
```

~> **Note:** Module registry integration was added in Terraform 10.6

You can also publish your own modules on the Terraform Registry. You may
use the [public registry](https://registry.terraform.io) for public modules.
For private modules, you must use [Terraform Enterprise](https://www.hashicorp.com/products/terraform).
You can use modules without a registry by
[sourcing modules directly](/docs/modules/sources.html), however non-registry
modules do not support versioning, documentation generation, and more.

Use the navigation to the left to learn more about using the registry.
