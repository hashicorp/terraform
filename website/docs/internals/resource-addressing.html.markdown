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

## Module path

A module path addresses a module within the tree of modules. It takes the form:

```
module.module_name[module index]
```

 * `module` - Module keyword indicating a child module (non-root). Multiple `module`
   keywords in a path indicate nesting.
 * `module_name` - User-defined name of the module.
 * `[module index]` - (Optional) [Index](#index-values-for-modules-and-resources) into a
   module with multiple instances, surrounded by square brace characters (`[` and `]`).

An address without a resource spec, i.e. `module.foo` applies to every resource within
the module if a single module, or all instances of a module if a module has multiple instances.
To address all resources of a particular module instance, include the module index in the address,
such as `module.foo[0]`.

If the module path is omitted, the address applies to the root module.

An example of the `module` keyword delineating between two modules that have multiple instances:

```
module.foo[0].module.bar["a"]
```

-> Module index only applies to modules in Terraform v0.13 or later, as in earlier
versions of Terraform, a module could not have multiple instances.

## Resource spec

A resource spec addresses a specific resource in the config. It takes the form:

```
resource_type.resource_name[resource index]
```

 * `resource_type` - Type of the resource being addressed.
 * `resource_name` - User-defined name of the resource.
 * `[resource index]` - (Optional) [Index](#index-values-for-modules-and-resources)
   into a resource with multiple instances, surrounded by square brace characters (`[` and `]`).

-> In Terraform v0.12 and later, a resource spec without a module path prefix
matches only resources in the root module. In earlier versions, a resource spec
without a module path prefix will match resources with the same type and name
in any descendent module.

## Index values for Modules and Resources

The following specifications apply to index values on modules and resources with multiple instances:

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

Refers to only the "example" instance in the config.
