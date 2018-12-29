---
layout: "docs"
page_title: "Output Values - Configuration Language"
sidebar_current: "docs-config-outputs"
description: |-
  Output values are the return values of a Terraform module.
---

# Output Values

Output values are like the return values of a Terraform module, and have several
uses:

- A child module can use outputs to expose a subset of its resource attributes
  to a parent module.
- A root module can use outputs to print certain values in the CLI output after
  running `terraform apply`.
- When using [remote state](/docs/state/remote.html), root module outputs can be
  accessed by other configurations via a
  [`terraform_remote_state` data source](/docs/providers/terraform/d/remote_state.html).

Resource instances managed by Terraform each export attributes whose values
can be used elsewhere in configuration. Output values are a way to expose some
of that information to the user of your module.

-> **Note:** For brevity, output values are often referred to as just "outputs"
when the meaning is clear from context.

## Declaring an Output Value

Each output value exported by a module must be declared using an `output`
block:

```hcl
output "instance_ip_addr" {
  value = aws_instance.server.private_ip
}
```

The label immediately after the `output` keyword is the name that can be used
to access this output in the parent module, if any, or the name that will be
displayed to the user for output values in the root module.

The `value` argument takes an [expression](./expressions.html)
whose result is to be returned to the user. In this example, the expression
refers to the `private_ip` attribute exposed by an `aws_instance` resource
defined elsewhere in this module (not shown). Any valid expression is allowed
as an output value.

Several other optional arguments are allowed within `output` blocks. These
will be described in the following sections.

## Output Value Documentation

Because the output values of a module are part of its user interface, you can
briefly describe the purpose of each value using the optional `description`
argument:

```hcl
output "instance_ip_addr" {
  value       = aws_instance.server.private_ip
  description = "The private IP address of the main server instance."
}
```

The description should concisely explain the
purpose of the output and what kind of value is expected. This description
string might be included in documentation about the module, and so it should be
written from the perspective of the user of the module rather than its
maintainer. For commentary for module maintainers, use comments.

## Sensitive Output Values

An output can be marked as containing sensitive material using the optional
`sensitive` argument:

```hcl
output "db_password" {
  value       = aws_db_instance.db.password
  description = "The password for logging in to the database."
  sensitive   = true
}
```

Setting an output value in the root module as sensitive prevents Terraform
from showing its value in the list of outputs at the end of `terraform apply`.
It might still be shown in the CLI output for other reasons, like if the
value is referenced in an expression for a resource argument.

Sensitive output values are still recorded in the
[state](/docs/state/index.html), and so will be visible to anyone who is able
to access the state data. For more information, see
[_Sensitive Data in State_](/docs/state/sensitive-data.html).

## Output Dependencies

Since output values are just a means for passing data out of a module, it is
usually not necessary to worry about their relationships with other nodes in
the dependency graph.

However, when a parent module accesses an output value exported by one of its
child modules, the dependencies of that output value allow Terraform to
correctly determine the dependencies between resources defined in different
modules.

Just as with
[resource dependencies](./resources.html#resource-dependencies),
Terraform analyzes the `value` expression for an output value and autmatically
determines a set of dependencies, but in less-common cases there are
dependencies that cannot be recognized implicitly. In these rare cases, the
`depends_on` argument can be used to create additional explicit dependencies:

```hcl
output "instance_ip_addr" {
  value       = aws_instance.server.private_ip
  description = "The private IP address of the main server instance."

  depends_on = [
    # Security group rule must be created before this IP address could
    # actually be used, otherwise the services will be unreachable.
    aws_security_group_rule.local_access,
  ]
}
```

The `depends_on` argument should be used only as a last resort. When using it,
always include a comment explaining why it is being used, to help future
maintainers understand the purpose of the additional dependency.
