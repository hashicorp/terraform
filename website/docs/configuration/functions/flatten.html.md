---
layout: "functions"
page_title: "flatten - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-flatten"
description: |-
  The flatten function eliminates nested lists from a list.
---

# `flatten` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`flatten` takes a list and replaces any elements that are lists with a
flattened sequence of the list contents.

## Examples

```
> flatten([["a", "b"], [], ["c"]])
["a", "b", "c"]
```

If any of the nested lists also contain directly-nested lists, these too are
flattened recursively:

```
> flatten([[["a", "b"], []], ["c"]])
["a", "b", "c"]
```

Indirectly-nested lists, such as those in maps, are _not_ flattened.

## Flattening nested structures for `for_each`

The
[resource `for_each`](/docs/configuration/resources.html#for_each-multiple-resource-instances-defined-by-a-map-or-set-of-strings)
and
[`dynamic` block](/docs/configuration/expressions.html#dynamic-blocks)
language features both require a collection value that has one element for
each repetition.

Sometimes your input data structure isn't naturally in a suitable shape for
use in a `for_each` argument, and `flatten` can be a useful helper function
when reducing a nested data structure into a flat one.

For example, consider a module that declares a variable like the following:

```hcl
variable "networks" {
  type = map(object({
    cidr_block = string
    subnets = map(object({
      cidr_block = string
    })
  })
}
```

The above is a reasonable way to model objects that naturally form a tree,
such as top-level networks and their subnets. The repetition for the top-level
networks can use this variable directly, because it's already in a form
where the resulting instances match one-to-one with map elements:

```hcl
resource "aws_vpc" "example" {
  for_each = var.networks

  cidr_block = each.value.cidr_block
}
```

However, in order to declare all of the _subnets_ with a single `resource`
block, we must first flatten the structure to produce a collection where each
top-level element represents a single subnet:

```hcl
locals {
  # flatten ensures that this local value is a flat list of objects, rather
  # than a list of lists of objects.
  network_subnets = flatten([
    for network_key, network in var.networks : [
      for subnet_key, subnet in network.subnets : {
        network_key = network_key
        subnet_key  = subnet_key
        network_id  = aws_vpc.example[network_key].id
        cidr_block  = subnet.cidr_block
      }
    ]
  ])
}

resource "aws_subnet" "example" {
  # local.network_subnets is a list, so we must now project it into a map
  # where each key is unique. We'll combine the network and subnet keys to
  # produce a single unique key per instance.
  for_each = {
    for subnet in local.network_subnets : "${subnet.network_key}.${subnet.subnet_key}" => subnet
  }

  vpc_id            = each.value.network_id
  availability_zone = each.value.subnet_key
  cidr_block        = each.value_cidr_block
}
```

The above results in one subnet instance per subnet object, while retaining
the associations between the subnets and their containing networks.

## Related Functions

* [`setproduct`](./setproduct.html) finds all of the combinations of multiple
  lists or sets of values, which can also be useful when preparing collections
  for use with `for_each` constructs.
