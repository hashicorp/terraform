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

Module configuration looks like the following:

```hcl
module "consul" {
  source  = "github.com/hashicorp/consul/terraform/aws"
  servers = 5
}
```

## Description

The `module` block configures a module and tells Terraform to build
its resources. Multiple module blocks may be used to configure and use
multiple modules.

The NAME of the module is logical: it is used only to reference the
module in other places in the configuration. It has no effect on the
source of the module. Therefore, you may name modules whatever you'd like.

Within the block (the `{ }`) is configuration for the module.
The only required key is `source`, which tells Terraform where this module
can be downloaded from. Valid source values are covered in more detail
in the
[module section](/docs/modules/index.html).

Other configuration within the module are dependent on the module itself.
Module configuration maps directly to
[variables](/docs/configuration/variables.html) within the module, so
parameters can have any of the data types that variables support, including
lists and maps.

## Syntax

The full syntax is:

```text
module NAME {
  source = SOURCE_URL

  CONFIG ...
}
```

where `CONFIG` is:

```text
KEY = VALUE
```
