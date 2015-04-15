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
larger infrastructure. The syntax of a resource address is:

```
<resource_type>.<resource_name>[optional fields]
```

Required fields:

 * `resource_type` - Type of the resource being addressed.
 * `resource_name` - User-defined name of the resource.

Optional fields may include:

 * `[N]` - where `N` is a `0`-based index into a resource with multiple
   instances specified by the `count` meta-parameter. Omitting an index when
   addressing a resource where `count > 1` means that the address references
   all instances.


## Examples

Given a Terraform config that includes:

```
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
