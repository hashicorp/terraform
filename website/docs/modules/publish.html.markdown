---
layout: "docs"
page_title: "Publishing Modules"
sidebar_current: "docs-modules-publish"
description: |-
  A module is a container for multiple resources that are used together.
---

## Publishing Modules

If you've built a module that you intend to be reused, we recommend
[publishing the module](/docs/registry/modules/publish.html) on the
[Terraform Registry](https://registry.terraform.io). This will version
your module, generate documentation, and more.

Published modules can be easily consumed by Terraform, and users can
[constrain module versions](/docs/configuration/modules.html#module-versions)
for safe and predictable updates. The following example shows how a caller
might use a module from the Terraform Registry:

```hcl
module "consul" {
  source = "hashicorp/consul/aws"
}
```

If you do not wish to publish your modules in the public registry, you can
instead use a [private registry](/docs/registry/private.html) to get
the same benefits.

### Distribution via other sources

Although the registry is the native mechanism for distributing re-usable
modules, Terraform can also install modules from
[various other sources](/docs/modules/sources.html). The alternative sources
do not support the first-class versioning mechanism, but some sources have
their own mechanisms for selecting particular VCS commits, etc.

We recommend that modules distributed via other protocols still use the
[standard module structure](./#standard-module-structure) so that it can
be used in a similar way to a registry module, or even _become_ a registry
module at a later time.
