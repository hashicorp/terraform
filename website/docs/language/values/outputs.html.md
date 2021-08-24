---
layout: "language"
page_title: "Output Values - Configuration Language"
sidebar_current: "docs-config-outputs"
description: |-
  Output values are the return values of a Terraform module.
---

# Output Values

Output values make information about your infrastructure available on the
command line, and can expose information for other Terraform configurations to
use. Output values are similar to return values in programming languages.

> **Hands-on:** Try the [Output Data From
Terraform](https://learn.hashicorp.com/tutorials/terraform/outputs)
tutorial on HashiCorp Learn.

Output values have several uses:

- A child module can use outputs to expose a subset of its resource attributes
  to a parent module.
- A root module can use outputs to print certain values in the CLI output after
  running `terraform apply`.
- When using [remote state](/docs/language/state/remote.html), root module outputs can be
  accessed by other configurations via a
  [`terraform_remote_state` data source](/docs/language/state/remote-state-data.html).

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

The label immediately after the `output` keyword is the name, which must be a
valid [identifier](/docs/language/syntax/configuration.html#identifiers). In a root module, this name is
displayed to the user; in a child module, it can be used to access the output's
value.

The `value` argument takes an [expression](/docs/language/expressions/index.html)
whose result is to be returned to the user. In this example, the expression
refers to the `private_ip` attribute exposed by an `aws_instance` resource
defined elsewhere in this module (not shown). Any valid expression is allowed
as an output value.

-> **Note:** Outputs are only rendered when Terraform applies your plan. Running
`terraform plan` will not render outputs.

## Accessing Child Module Outputs

In a parent module, outputs of child modules are available in expressions as
`module.<MODULE NAME>.<OUTPUT NAME>`. For example, if a child module named
`web_server` declared an output named `instance_ip_addr`, you could access that
value as `module.web_server.instance_ip_addr`.

## Optional Arguments

`output` blocks can optionally include `description`, `sensitive`, and `depends_on` arguments, which are described in the following sections.

<a id="description"></a>

### `description` — Output Value Documentation

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

<a id="sensitive"></a>

### `sensitive` — Suppressing Values in CLI Output

An output can be marked as containing sensitive material using the optional
`sensitive` argument:

```hcl
output "db_password" {
  value       = aws_db_instance.db.password
  description = "The password for logging in to the database."
  sensitive   = true
}
```

Terraform will hide values marked as sensitive in the messages from
`terraform plan` and `terraform apply`. In the following scenario, our root
module has an output declared as sensitive and a module call with a
sensitive output, which we then use in a resource attribute.

```hcl
# main.tf

module "foo" {
  source = "./mod"
}

resource "test_instance" "x" {
  some_attribute = module.mod.a # resource attribute references a sensitive output
}

output "out" {
  value     = "xyz"
  sensitive = true
}

# mod/main.tf, our module containing a sensitive output

output "a" {
  value     = "secret"
  sensitive = true
}
```

When we run a plan or apply, the sensitive value is redacted from output:

```
Terraform will perform the following actions:

  # test_instance.x will be created
  + resource "test_instance" "x" {
      + some_attribute    = (sensitive)
    }

Plan: 1 to add, 0 to change, 0 to destroy.

Changes to Outputs:
  + out = (sensitive value)
```

-> **Note:** In Terraform versions prior to Terraform 0.14, setting an output
value in the root module as sensitive would prevent Terraform from showing its
value in the list of outputs at the end of `terraform apply`. However, the
value could still display in the CLI output for other reasons, like if the
value is referenced in an expression for a resource argument.

Terraform will still record sensitive values in the [state](/docs/language/state/index.html),
and so anyone who can access the state data will have access to the sensitive
values in cleartext. For more information, see
[_Sensitive Data in State_](/docs/language/state/sensitive-data.html).

<a id="depends_on"></a>

### `depends_on` — Explicit Output Dependencies

Since output values are just a means for passing data out of a module, it is
usually not necessary to worry about their relationships with other nodes in
the dependency graph.

However, when a parent module accesses an output value exported by one of its
child modules, the dependencies of that output value allow Terraform to
correctly determine the dependencies between resources defined in different
modules.

Just as with
[resource dependencies](/docs/language/resources/behavior.html#resource-dependencies),
Terraform analyzes the `value` expression for an output value and automatically
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
