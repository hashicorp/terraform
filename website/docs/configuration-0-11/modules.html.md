---
layout: "docs"
page_title: "Configuring Modules"
sidebar_current: "docs-config-modules"
description: |-
  Modules are used in Terraform to modularize and encapsulate groups of resources in your infrastructure. For more information on modules, see the dedicated modules section.
---

# Module Configuration

Modules are used in Terraform to modularize and encapsulate groups of
resources in your infrastructure. For more information on modules, see
the dedicated
[modules section](/docs/modules/index.html).

This page assumes you're familiar with the
[configuration syntax](/docs/configuration/syntax.html)
already.

## Example

```hcl
module "consul" {
  source  = "hashicorp/consul/aws"
  servers = 5
}
```

## Description

A `module` block instructs Terraform to create an instance of a module,
and in turn to instantiate any resources defined within it.

The name given in the block header is used to reference the particular module
instance from expressions within the calling module, and to refer to the
module on the command line. It has no meaning outside of a particular
Terraform configuration.

Within the block body is the configuration for the module. All attributes
within the block must correspond to [variables](/docs/configuration/variables.html)
within the module, with the exception of the following which Terraform
treats as special:

* `source` - (Required) A [module source](/docs/modules/sources.html) string
  specifying the location of the child module source code.

* `version` - (Optional) A [version constraint](/docs/modules/usage.html#module-versions)
  string that specifies which versions of the referenced module are acceptable.
  The newest version matching the constraint will be used. `version` is supported
  only for modules retrieved from module registries.

* `providers` - (Optional) A map whose keys are provider configuration names
  that are expected by child module and whose values are corresponding
  provider names in the calling module. This allows
  [provider configurations to be passed explicitly to child modules](/docs/modules/usage.html#providers-within-modules).
  If not specified, the child module inherits all of the default (un-aliased)
  provider configurations from the calling module.
