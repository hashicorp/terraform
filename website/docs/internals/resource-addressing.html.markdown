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
resource_type.resource_name[resource index]
```

 * `resource_type` - Type of the resource being addressed.
 * `resource_name` - User-defined name of the resource.
 * `[resource index]` - an optional index into a resource with multiple
   instances, surrounded by square brace characters (`[` and `]`).

-> In Terraform v0.12 and later, a resource spec without a module path prefix
matches only resources in the root module. In earlier versions, a resource spec
without a module path prefix will match resources with the same type and name
in any descendent module.

__Resource index__:

 * `[N]` where `N` is a `0`-based numerical index into a resource with multiple
   instances specified by the `count` meta-argument. Omitting an index when
   addressing a resource where `count > 1` means that the address references
   all instances.
 * `["INDEX"]` where `INDEX` is a alphanumerical key index into a resource with
   multiple instances specified by the `for_each` meta-argument.

## Examples

### count Example

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

### for_each Example

Given a Terraform config that includes:

```hcl
resource "aws_instance" "web" {
  # ...
  for_each = {
    "terraform": "value1",
    "resource":  "value2",
    "indexing":  "value3",
    "example":   "value4",
  }
}
```

An address like this:

```
aws_instance.web["example"]
```

Refers to only the "example" instance in the config, and an address like this:

```
aws_instance.web[*]
```

Refers to all four "web" instances.
