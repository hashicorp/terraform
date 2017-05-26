---
layout: "docs"
page_title: "Internals: Resource Address"
sidebar_current: "docs-internals-resource-addressing"
description: |-
  Resource addressing is used to target specific resources in a larger
  infrastructure.
---

# Resource Addressing

A __Resource Address__ is a string that references a specific resource in a
larger infrastructure. An address is made up of two parts:

```
[module path][resource spec]
```

__Module path__:

A module path addresses a module within the tree of modules. It takes the form:

```
module.A.module.B.module.C...
```

Multiple modules in a path indicate nesting. If a module path is specified
without a resource spec, the address applies to every resource within the
module. If the module path is omitted, this addresses the root module.

__Resource spec__:

A resource spec addresses a specific resource in the config. It takes the form:

```
resource_type.resource_name[N]
```

 * `resource_type` - Type of the resource being addressed.
 * `resource_name` - User-defined name of the resource.
 * `[N]` - where `N` is a `0`-based index into a resource with multiple
   instances specified by the `count` meta-parameter. Omitting an index when
   addressing a resource where `count > 1` means that the address references
   all instances.


## Examples

Given a Terraform config that includes:

```hcl
resource "aws_instance" "web" {
  # ...
  count = 4
}
```

An address like this:

```
aws_instance.web[3]
```

Refers to only the last instance in the config, and an address like this:

```
aws_instance.web
```

Refers to all four "web" instances.
