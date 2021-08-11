---
layout: "language"
page_title: "uuid - Functions - Configuration Language"
sidebar_current: "docs-funcs-crypto-uuid"
description: |-
  The uuid function generates a unique id.
---

# `uuid` Function

`uuid` generates a unique identifier string.

The id is a generated and formatted as required by
[RFC 4122 section 4.4](https://datatracker.ietf.org/doc/html/rfc4122#section-4.4),
producing a Version 4 UUID. The result is a UUID generated only from
pseudo-random numbers.

This function produces a new value each time it is called, and so using it
directly in resource arguments will result in spurious diffs. We do not
recommend using the `uuid` function in resource configurations, but it can
be used with care in conjunction with
[the `ignore_changes` lifecycle meta-argument](/docs/language/meta-arguments/lifecycle.html#ignore_changes). 

If this function produced a value during the plan step, it would cause the final configuration during the apply step not to match the actions shown in the plan (since the function is called again in the apply step, and would return a different value), which violates the Terraform execution model. For that reason, this function produces an unknown value result during the plan step, with the real result being decided only during the apply step.

In most cases we recommend using [the `random` provider](https://registry.terraform.io/providers/hashicorp/random/latest/docs)
instead, since it allows the one-time generation of random values that are
then retained in the Terraform [state](/docs/language/state/index.html) for use by
future operations. In particular,
[`random_id`](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/id) can generate results with
equivalent randomness to the `uuid` function.

## Examples

```
> uuid()
b5ee72a3-54dd-c4b8-551c-4bdc0204cedb
```

## Related Functions

* [`uuidv5`](./uuidv5.html), which generates name-based UUIDs.
