---
layout: "random"
page_title: "Random: random_shuffle"
sidebar_current: "docs-random-resource-shuffle"
description: |-
  Produces a random permutation of a given list.
---

# random\_shuffle

The resource `random_shuffle` generates a random permutation of a list
of strings given as an argument.

## Example Usage

```hcl
resource "random_shuffle" "az" {
  input = ["us-west-1a", "us-west-1c", "us-west-1d", "us-west-1e"]
  result_count = 2
}

resource "aws_elb" "example" {
  # Place the ELB in any two of the given availability zones, selected
  # at random.
  availability_zones = ["${random_shuffle.az.result}"]

  # ... and other aws_elb arguments ...
}
```

## Argument Reference

The following arguments are supported:

* `input` - (Required) The list of strings to shuffle.

* `result_count` - (Optional) The number of results to return. Defaults to
  the number of items in the `input` list. If fewer items are requested,
  some elements will be excluded from the result. If more items are requested,
  items will be repeated in the result but not more frequently than the number
  of items in the input list.

* `keepers` - (Optional) Arbitrary map of values that, when changed, will
  trigger a new id to be generated. See
  [the main provider documentation](../index.html) for more information.

* `seed` - (Optional) Arbitrary string with which to seed the random number
  generator, in order to produce less-volatile permutations of the list.
  **Important:** Even with an identical seed, it is not guaranteed that the
  same permutation will be produced across different versions of Terraform.
  This argument causes the result to be *less volatile*, but not fixed for
  all time.

## Attributes Reference

The following attributes are exported:

* `result` - Random permutation of the list of strings given in `input`.

