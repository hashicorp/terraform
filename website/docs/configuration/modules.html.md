---
layout: "docs"
page_title: "Configuring Modules"
sidebar_current: "docs-config-modules"
description: |-
  Modules allow multiple resources to be grouped together and encapsulated.
---

# Modules

A _module_ is a container for multiple resources that are used together.

Every Terraform configuration has at least one module, known as its
_root module_, which consists of the resources defined in the `.tf` files in
the main working directory.

A module can call other modules, which lets you include the child module's
resources into the configuration in a concise way. Modules
can also be called multiple times, either within the same configuration or
in separate configurations, allowing resource configurations to be packaged
and re-used.

This page describes how to call one module from another. Other pages in this
section of the documentation describe the different elements that make up
modules, and there is further information about how modules can be used,
created, and published in [the dedicated _Modules_ section](/docs/modules/index.html).

## Calling a Child Module

To _call_ a module means to include the contents of that module into the
configuration with specific values for its
[input variables](./variables.html). Modules are called
from within other modules using `module` blocks:

```hcl
module "servers" {
  source = "./app-cluster"

  servers = 5
}
```

A module that includes a `module` block like this is the _calling module_ of the
child module.

The label immediately after the `module` keyword is a local name, which the
calling module can use to refer to this instance of the module.

Within the block body (between `{` and `}`) are the arguments for the module.
Most of the arguments correspond to [input variables](./variables.html)
defined by the module, including the `servers` argument in the above example.

All modules require a `source` argument, which is a meta-argument defined by
Terraform CLI. Its value is either the path to a local directory of the
module's configuration files, or a remote module source that Terraform should
download and use. This value must be a literal string with no template
sequences; arbitrary expressions are not allowed. For more information on
possible values for this argument, see [Module Sources](/docs/modules/sources.html).

The same source address can be specified in multiple `module` blocks to create
multiple copies of the resources defined within, possibly with different
variable values.

## Accessing Module Output Values

The resources defined in a module are encapsulated, so the calling module
cannot access their attributes directly. However, the child module can
declare [output values](./outputs.html) to selectively
export certain values to be accessed by the calling module.

For example, if the `./app-cluster` module referenced in the example above
exported an output value named `instance_ids` then the calling module
can reference that result using the expression `module.servers.instance_ids`:

```hcl
resource "aws_elb" "example" {
  # ...

  instances = module.servers.instance_ids
}
```

For more information about referring to named values, see
[Expressions](./expressions.html).

## Other Meta-arguments

Along with the `source` meta-argument described above, module blocks have
some more meta-arguments that have special meaning across all modules,
described in more detail in other sections:

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

In addition to the above, the argument names `count`, `for_each` and
`lifecycle` are not currently used by Terraform but are reserved for planned
future features.

Since modules are a complex feature in their own right, further detail
about how modules can be used, created, and published is included in
[the dedicated section on modules](/docs/modules/index.html).
