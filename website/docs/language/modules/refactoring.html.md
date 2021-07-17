---
layout: "language"
page_title: "Refactoring"
sidebar_current: "docs-modules-recactoring"
description: Making backward-compatible changes to modules already in use.
---

# Refactoring

In shared modules and long-lived configurations you may eventually outgrow
your initial module heirarchy and resource naming and need to adopt a new
strategy. For example, you might decide that what was previously one
child module makes more sense as two separate modules, with each of the
resources moving to one of them.

Terraform compares previous state with new configuration, correlating by
each module or resource's unique address. Therefore _by default_ Terraform will
understand moving or renaming an object as an intent to destroy the object
at the old address and to create a new object at the new address.

You can use `moved` blocks in your configuration to record where you've
historically moved or renamed an object, in which case Terraform will instead
treat an existing object at the old address as if it now belongs to the new
address.

## `moved` Block Syntax

A `moved` block expects no labels and contains only `from` and `two` arguments:

```hcl
moved {
  from = aws_instance.a
  to   = aws_instance.b
}
```

The example above records that the resource currently known as `aws_instance.b`
was, in an earlier version of this module, known instead as `aws_instance.a`.

Prior to creating a new plan for `aws_instance.b`, Terraform will first check
whether there's an existing object for `aws_instance.a` recorded in the state,
and if so will rename that object to `aws_instance.b` before taking any other
planning actions. The resulting plan is as if the object had originally been
created at `aws_instance.b`, avoiding any need to destroy that object.

The `from` and `to` addresses both use a special addressing syntax, different
than other references in a module, which allows selecting both modules and
resources, and possibly selecting resources inside child modules. The following
sections describe the various different refactoring use-cases, along with the
appropriate addressing syntax for each situation.

* [Renaming a Resource](#renaming-a-resource)
* [Enabling `count` and `for_each` For a Resource](#enabling-count-and-for_each-for-a-resource)
* [Renaming a Module Call](#renaming-a-module-call)
* [Enabling `count` and `for_each` For a Module Call](#enabling-count-and-for_each-for-a-module-call)
* [Splitting One Module into Multiple](#splitting-one-module-into-multiple)
* [Removing `moved` blocks](#removing-moved-blocks)

## Renaming a Resource

The following is an expanded version of the example above showing the full
configurations at each step.

Imagine that the inital version of your module included a resource configuration
like the following:

```hcl
resource "aws_instance" "a" {
  count = 2

  # (resource-type-specific configuration)
}
```

Applying this configuration for the first time would cause Terraform to
create `aws_instance.a[0]` and `aws_instance.a[1]`.

If you later choose a better name for this resource, then you can change the
name label in the `resource` block and record the old name inside a `moved` block:

```hcl
resource "aws_instance" "b" {
  count = 2

  # (resource-type-specific configuration)
}

moved {
  from = aws_instance.a
  to   = aws_instance.b
}
```

When creating the next plan for each configuration using this module, Terraform
will treat any existing objects belonging to `aws_instance.a` as if they had
been created as `aws_instance.b`. `aws_instance.a[0]` will be treated as
`aws_instance.b[0]`, and `aws_instance.a[1]` as `aws_instance.b[1]`.

New instances of the module, which _never_ had an
`aws_instance.a`, will ignore the `moved` block and propose to create
`aws_instance.b[0]` and `aws_instance.b[1]` as normal.

Both of the addresses in this example referred to a resource as a whole, and
so Terraform recognizes the move for all instances of the resource. That is,
it covers both `aws_instance.a[0]` and `aws_instance.a[1]` without the need
to identify each one separately.

Each resource type has a separate schema and so objects of different types
are not compatible. Therefore although you can use `moved` to change the name
of a resource, you _cannot_ use `moved` to change to a different resource type
or to change a managed resource (a `resource` block) into a data resource
(a `data` block).

There is one exception to that constraint: a provider developer may mark a
particular resource type name as deprecated and nominate a new name to replace
it. In that case, the provider developer is responsible for making the old
and new names schema-compatible, and can then tell Terraform that it's safe to
support changing from the old to the new type via a `moved` block. Provider
documentation will include an example for any situation where this exception
applies.

## Enabling `count` or `for_each` For a Resource

Imagine that the initial version of your module contains a single-instance
resource:

```hcl
resource "aws_instance" "a" {
  # (resource-type-specific configuration)
}
```

Applying this configuration would lead to Terraform creating an object
bound to the address `aws_instance.a`.

Later requirements might lead to you needing to use
[`for_each`](../meta-arguments/for_each.html) with this resource, in order
to systematically declare multiple instances. To preserve an object that
was previously associated with `aws_instance.a` alone, you must used a `moved`
block to specify which instance key that object will take in the new
configuration:

```hcl
locals {
  instances = tomap({
    big = {
      instance_type = "m3.large"
    }
    small = {
      instance_type = "t2.medium"
    }
  })
}

resource "aws_instance" "a" {
  for_each = local.instances

  instance_type = each.value.instance_type
  # (other resource-type-specific configuration)
}

moved {
  from = aws_instance.a
  to   = aws_instance.a["small"]
}
```

The above will avoid Terraform planning to destroy any existing object at
`aws_instance.a`, treating it instead as if it were originally created
as `aws_instance.a["small"]`.

When at least one of the two addresses includes an instance key, like
`["small"]` in the above example, Terraform will understand both addresses
as referring to specific _instances_ of a resource rather than the resource
as a whole. That means you can use `moved` to switch between keys and to
add and remove keys as you switch between `count`, `for_each`, or neither.

The following are some other examples of valid `moved` blocks that record
changes to resource instance keys in a similar way:

```hcl
# Both old and new configuration used "for_each", but the
# "small" element was renamed to "tiny".
moved {
  from = aws_instance.b["small"]
  to   = aws_instance.b["tiny"]
}

# The old configuration used "count" and the new configuration
# uses "for_each", with the following mappings from
# index to key:
moved {
  from = aws_instance.c[0]
  to   = aws_instance.c["small"]
}
moved {
  from = aws_instance.c[1]
  to   = aws_instance.c["tiny"]
}

# The old configuration used "count" and the new configuration
# uses either "count" nor "for_each", and we want to keep
# only the object at index 2.
moved {
  from = aws_instance.d[2]
  to   = aws_instance.d
}
```

## Renaming a Module Call

You can rename a call to a module in a similar way as renaming a resource.
Consider the following original module version:

```hcl
module "a" {
  source = "../modules/example"

  # (module arguments)
}
```

When applying this configuration, Terraform would prefix the addresses for
any resources declared in this module with the module path `module.a`.
For example, a resource `aws_instance.example` would have the full address
`module.a.aws_instance.example`.

If you later choose a better name for this module call, then you can change the
name label in the `module` block and record the old name inside a `moved` block:

```hcl
module "b" {
  source = "../modules/example"

  # (module arguments)
}
```

When creating the next plan for each configuration using this module, Terraform
will treat any existing object addresses beginning with `module.a` as if
they had instead been created in `module.b`. `module.a.aws_instance.example`
would be treated as `module.b.aws_instance.example`.

Both of the addresses in this example referred to a module call as a whole, and
so Terraform recognizes the move for all instances of the call. If this
module call used `count` or `for_each` then it would apply to all of the
instances, without the need to specify each one separately.

## Enabling `count` and `for_each` For a Module Call

Imagine that the initial version of your module contains a single-instance
module:

```hcl
module "a" {
  source = "../modules/example"

  # (module arguments)
}
```

Applying this configuration would cause Terraform to create objects whose
addresses begin with `module.a`.

Later requirements might lead to you needing to use
[`count`](../meta-arguments/count.html) with this resource, in order
to systematically declare multiple instances. To preserve an object that
was previously associated with `aws_instance.a` alone, you can used a `moved`
block to specify which instance key that object will take in the new
configuration:

```hcl
module "a" {
  source = "../modules/example"
  count  = 3

  # (module arguments)
}

moved {
  from = module.a
  to   = module.a[2]
}
```

The above will cause Terraform to treat all objects in `module.a` as if they
were originally created in `module.a[2]`, and thus it will plan to create
new objects only for `module.a[0]` and `module.a[1]`.

When at least one of the two addresses includes an instance key, like
`[2]` in the above example, Terraform will understand both addresses as
referring to specific _instances_ of a module call rather than the module
call as a whole. That means you can use `moved` to switch between keys and to
add and remove keys as you switch between `count`, `for_each`, or neither.

For more examples of recording moves associated with instances, refer to
the similar section
[Enabling `count` and `for_each` For a Resource](#enabling-count-and-for_each-for-a-resource).

# Splitting One Module into Multiple

As a module grows to support new requirements, it might eventually grow big
enough to warrant splitting into two separate modules.

Consider the following initial version of a module:

```hcl
resource "aws_instance" "a" {
  # (other resource-type-specific configuration)
}

resource "aws_instance" "b" {
  # (other resource-type-specific configuration)
}

resource "aws_instance" "c" {
  # (other resource-type-specific configuration)
}
```

Imagine that we intend to split this into two modules as follows:

* `aws_instance.a` will now belong to module "x".
* `aws_instance.b` will also belong to module "x".
* `aws_instance.c`, though, will belong to module "y".

To achieve this refactoring without replacing existing objects bound to the
old resource addresses, you must:

1. Write module "x", copying over the two resources it should contain.
2. Write module "y", copying over the one resource it should contain.
3. Edit the original module to no longer include any of these resources, and
   instead to contain only shim configuration to migrate existing users.

The new modules "x" and "y" will therefore contain only `resource` blocks,
as normal:

```hcl
# module "x"

resource "aws_instance" "a" {
  # (other resource-type-specific configuration)
}

resource "aws_instance" "b" {
  # (other resource-type-specific configuration)
}
```

```hcl
# module "y"

resource "aws_instance" "c" {
  # (other resource-type-specific configuration)
}
```

The original module, now only a shim for backward-compatibility, calls the
two new modules and indicates that the resources moved into them:

```hcl
module "x" {
  source = "../modules/x"

  # ...
}

module "y" {
  source = "../modules/y"

  # ...
}

moved {
  from = aws_instance.a
  to   = module.x.aws_instance.a
}

moved {
  from = aws_instance.b
  to   = module.x.aws_instance.b
}

moved {
  from = aws_instance.c
  to   = module.y.aws_instance.c
}
```

When an existing user of the original module upgrades to the new "shim"
version, Terraform will notice these three `moved` blocks and thus behave
as if the objects associated with the three old resource addresses were
originally created inside the two new modules.

New users of this family of modules may use either the combined shim module
_or_ the two new modules separately. You may wish to communicate to your
existing users that the old module is now deprecated and so they should use
the two separate modules for any new needs.

The multi-module refactoring situation is unusual in that it violates the
typical rule that a parent module sees its child module as a "closed box",
unaware of exactly which resources are declared inside it. This compromise
assumes that all three of these modules are maintained by the same people
and distributed together in a single
[module package](sources.html#modules-in-package-sub-directories).

To avoid strong [coupling](https://en.wikipedia.org/wiki/Coupling_(computer_programming))
between separately-packaged modules, Terraform will only allow declarations of
moves between modules in the same package. In other words, Terraform would
not have allowed moving into `module.x` above if the `source` address of
that call had not been a [local path](sources.html#local-paths).

Module references in `moved` blocks are resolved relative to the module
instance they are defined in. For example, if our original module above were
already a child module named `module.original`, the reference to
`module.x.aws_instance.a` would resolve as
`module.original.module.x.aws_instance.a`. A module may only make `moved`
statements about its own objects and objects of its child modules.

If you need to refer to resources within a module that was called using
`count` or `for_each` meta-arguments, you must specify a specific instance
key to use in order to match with the new location of the resource
configuration:

```hcl
moved {
  from = aws_instance.example
  to   = module.new[2].aws_instance.example
}
```

## Removing `moved` Blocks

Over time, a long-lasting module may accumulate many `moved` blocks.

If you are maintaining private modules within an organization and you know
for certain that all users of your module have successfully run
`terraform apply` with your new module version already then it can be safe
to remove `moved` blocks in later versions of your module.

However, in the general case removing a `moved` block is a breaking change
to your module, because any configurations whose state refers to the old
address will then plan to delete that existing object instead of move it.

We recommend that by default module authors retain all historical `moved`
blocks from earlier versions of their modules, in order to preserve the
upgrade path for users of any old version. If later maintence causes you
to rename or move the same object twice, you can document that full history
using _chained_ `moved` blocks, where the new block refers to the existing
block:

```hcl
moved {
  from = aws_instance.a
  to   = aws_instance.b
}

moved {
  from = aws_instance.b
  to   = aws_instance.c
}
```

When you record a sequence of moves in this way you will allow for successful
upgrades both for configurations with objects at `aws_instance.a` _and_
configurations with objects at `aws_instance.b`. In both cases, Terraform will
treat the existing object as if it had been originally created as
`aws_instance.c`.
