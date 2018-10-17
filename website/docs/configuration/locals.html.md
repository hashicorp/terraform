---
layout: "docs"
page_title: "Configuring Local Values"
sidebar_current: "docs-config-locals"
description: |-
  Local values assign a name to an expression that can then be used multiple times
  within a module.
---

# Local Values

A local value assigns a name to an [expression](/docs/configuration/expressions.html),
allowing it to then be used multiple times within a module without repeating
it.

Comparing modules to functions in a traditional programming language,
if [input variables](./variables.html) are analogous to function arguments and
[outputs values](./outputs.html) are analogous to function return values then
_local values_ are comparable to a function's local temporary symbols.

## Declaring a Local Value

A set of related local values can be delared together in a single `locals`
block:

```hcl
locals {
  service_name = "forum"
  owner        = "Community Team"
}
```

For brevity, local values are often referred to just as "locals", when the
meaning is clear from context.

The expressions assigned to local value names can either be simple constants
like the above, allowing these values to be defined only once but used many
times, or they can be more complex expressions that transform or combine
values from elsewhere in the module:

```hcl
locals {
  # Ids for multiple sets of EC2 instances, merged together
  instance_ids = "${concat(aws_instance.blue.*.id, aws_instance.green.*.id)}"
}

locals {
  # Common tags to be assigned to all resources
  common_tags = {
    Service = local.service_name
    Owner   = local.owner
  }
}
```

As shown above, local values can be referenced from elsewhere in the module
with an expression like `local.common_tags`, and output values can reference
each other in order to build more complex values from simpler ones.

```
resource "aws_instance" "example" {
  # ...

  tags = local.common_tags
}
```

## When To Use Local Values

Local Values can be helpful to avoid repeating the same values or expressions
multiple times in a configuration, but if overused they can also make a
configuration hard to read by future maintainers by hiding the actual values
used.

Use Local Values only in moderation, in situations where a single value or
result is used in many places _and_ that value is likely to be changed in
future. The ability to easily change the value in a central place is the key
advantage of local values.
