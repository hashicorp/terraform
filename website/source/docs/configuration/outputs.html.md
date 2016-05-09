---
layout: "docs"
page_title: "Configuring Outputs"
sidebar_current: "docs-config-outputs"
description: |-
  Outputs define values that will be highlighted to the user when Terraform applies, and can be queried easily using the output command. Output usage is covered in more detail in the getting started guide. This page covers configuration syntax for outputs.
---

# Output Configuration

Outputs define values that will be highlighted to the user
when Terraform applies, and can be queried easily using the
[output command](/docs/commands/output.html). Output usage
is covered in more detail in the
[getting started guide](/intro/getting-started/outputs.html).
This page covers configuration syntax for outputs.

Terraform knows a lot about the infrastructure it manages.
Most resources have a handful or even a dozen or more attributes
associated with it. Outputs are a way to easily extract
information.

This page assumes you're familiar with the
[configuration syntax](/docs/configuration/syntax.html)
already.

## Example

An output configuration looks like the following:

```
output "address" {
	value = "${aws_instance.web.public_dns}"
}
```

## Description

The `output` block configures a single output variable. Multiple
output variables can be configured with multiple output blocks.
The `NAME` given to the output block is the name used to reference
the output variable.

Within the block (the `{ }`) is configuration for the output.
These are the parameters that can be set:

  * `value` (required, string) - The value of the output. This must
    be a string. This usually includes an interpolation since outputs
    that are static aren't usually useful.

## Syntax

The full syntax is:

```
output NAME {
	value = VALUE
}
```

## Sensitive Outputs

Outputs can be marked as containing sensitive material by setting the
`sensitive` attribute to `true`, like this:

```
output "sensitive" {
    sensitive = true
    value = VALUE 
}
```

When outputs are displayed on-screen following a `terraform apply` or
`terraform refresh`, sensitive outputs are redacted, with `<sensitive>`
displayed in place of their value.

### Limitations of Sensitive Outputs

* The values of sensitive outputs are still stored in the Terraform
  state, and available using the `terraform output` command, so cannot be
  relied on as a sole means of protecting values.
* Sensitivity is not tracked internally, so if the output is interpolated in
  another module into a resource, the value will be displayed. 
