---
layout: "language"
page_title: "Modules - Configuration Language"
sidebar_current: "docs-config-modules"
description: |-
  Modules allow multiple resources to be grouped together and encapsulated.
---

# Module Blocks

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Modules](../configuration-0-11/modules.html).

> **Hands-on:** Try the [Reuse Configuration with Modules](https://learn.hashicorp.com/collections/terraform/modules?utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS) collection on HashiCorp Learn.

A _module_ is a container for multiple resources that are used together.

Every Terraform configuration has at least one module, known as its
_root module_, which consists of the resources defined in the `.tf` files in
the main working directory.

A module can call other modules, which lets you include the child module's
resources into the configuration in a concise way. Modules
can also be called multiple times, either within the same configuration or
in separate configurations, allowing resource configurations to be packaged
and re-used.

This page describes how to call one module from another. For more information
about creating re-usable child modules, see [Module Development](/docs/modules/index.html).

## Calling a Child Module

To _call_ a module means to include the contents of that module into the
configuration with specific values for its
[input variables](/docs/configuration/variables.html). Modules are called
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
Module calls use the following kinds of arguments:

- The `source` argument is mandatory for all modules.

- The `version` argument is recommended for modules from a registry.

- Most other arguments correspond to [input variables](/docs/configuration/variables.html)
  defined by the module. (The `servers` argument in the example above is one of
  these.)

- Terraform defines a few other meta-arguments that can be used with all
  modules, including `for_each` and `depends_on`.

### Source

All modules **require** a `source` argument, which is a meta-argument defined by
Terraform. Its value is either the path to a local directory containing the
module's configuration files, or a remote module source that Terraform should
download and use. This value must be a literal string with no template
sequences; arbitrary expressions are not allowed. For more information on
possible values for this argument, see [Module Sources](/docs/modules/sources.html).

The same source address can be specified in multiple `module` blocks to create
multiple copies of the resources defined within, possibly with different
variable values.

After adding, removing, or modifying `module` blocks, you must re-run
`terraform init` to allow Terraform the opportunity to adjust the installed
modules. By default this command will not upgrade an already-installed module;
use the `-upgrade` option to instead upgrade to the newest available version.

### Version

When using modules installed from a module registry, we recommend explicitly
constraining the acceptable version numbers to avoid unexpected or unwanted
changes.

Use the `version` argument in the `module` block to specify versions:

```shell
module "consul" {
  source  = "hashicorp/consul/aws"
  version = "0.0.5"

  servers = 3
}
```

The `version` argument accepts a [version constraint string](/docs/configuration/version-constraints.html).
Terraform will use the newest installed version of the module that meets the
constraint; if no acceptable versions are installed, it will download the newest
version that meets the constraint.

Version constraints are supported only for modules installed from a module
registry, such as the public [Terraform Registry](https://registry.terraform.io/)
or [Terraform Cloud's private module registry](/docs/cloud/registry/index.html).
Other module sources can provide their own versioning mechanisms within the
source string itself, or might not support versions at all. In particular,
modules sourced from local file paths do not support `version`; since
they're loaded from the same source repository, they always share the same
version as their caller.

### Meta-arguments

Along with `source` and `version`, Terraform defines a few more
optional meta-arguments that have special meaning across all modules,
described in more detail in the following pages:

- `count` - Creates multiple instances of a module from a single `module` block.
  See [the `count` page](/docs/configuration/meta-arguments/count.html)
  for details.

- `for_each` - Creates multiple instances of a module from a single `module`
  block. See
  [the `for_each` page](/docs/configuration/meta-arguments/for_each.html)
  for details.

- `providers` - Passes provider configurations to a child module. See
  [the `providers` page](/docs/configuration/meta-arguments/module-providers.html)
  for details. If not specified, the child module inherits all of the default
  (un-aliased) provider configurations from the calling module.

- `depends_on` - Creates explicit dependencies between the entire
  module and the listed targets. See
  [the `depends_on` page](/docs/configuration/meta-arguments/depends_on.html)
  for details.

In addition to the above, the `lifecycle` argument is not currently used by
Terraform but is reserved for planned future features.

## Accessing Module Output Values

The resources defined in a module are encapsulated, so the calling module
cannot access their attributes directly. However, the child module can
declare [output values](/docs/configuration/outputs.html) to selectively
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
[Expressions](/docs/configuration/expressions/index.html).

## Transferring Resource State Into Modules

When refactoring an existing configuration to split code into child modules,
moving resource blocks between modules causes Terraform to see the new location
as an entirely different resource from the old. Always check the execution plan
after moving code across modules to ensure that no resources are deleted by
surprise.

If you want to make sure an existing resource is preserved, use
[the `terraform state mv` command](/docs/commands/state/mv.html) to inform
Terraform that it has moved to a different module.

When passing resource addresses to `terraform state mv`, resources within child
modules must be prefixed with `module.<MODULE NAME>.`. If a module was called with
[`count`](/docs/configuration/meta-arguments/count.html) or
[`for_each`](/docs/configuration/meta-arguments/for_each.html),
its resource addresses must be prefixed with `module.<MODULE NAME>[<INDEX>].`
instead, where `<INDEX>` matches the `count.index` or `each.key` value of a
particular module instance.

Full resource addresses for module contents are used within the UI and on the
command line, but cannot be used within a Terraform configuration. Only
[outputs](/docs/configuration/outputs.html) from a module can be referenced from
elsewhere in your configuration.

## Tainting resources within a module

The [taint command](/docs/commands/taint.html) can be used to _taint_ specific
resources within a module:

```shell
$ terraform taint module.salt_master.aws_instance.salt_master
```

It is not possible to taint an entire module. Instead, each resource within
the module must be tainted separately.
