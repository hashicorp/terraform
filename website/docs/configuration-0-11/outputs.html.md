---
layout: "docs"
page_title: "Output Values - 0.11 Configuration Language"
sidebar_current: "docs-conf-old-outputs"
description: |-
  Outputs define values that will be highlighted to the user when Terraform applies, and can be queried easily using the output command. Output usage is covered in more detail in the getting started guide. This page covers configuration syntax for outputs.
---

# Output Values

Outputs define values that will be highlighted to the user
when Terraform applies, and can be queried easily using the
[output command](/docs/commands/output.html). Output usage
is covered in more detail in the
[getting started guide](/intro/getting-started/outputs.html).
This page covers configuration syntax for outputs.

Terraform knows a lot about the infrastructure it manages.
Most resources have attributes associated with them, and
outputs are a way to easily extract and query that information.

This page assumes you are familiar with the
[configuration syntax](/docs/configuration/syntax.html)
already.

## Example

A simple output configuration looks like the following:

```hcl
output "address" {
  value = "${aws_instance.db.public_dns}"
}
```

This will output a string value corresponding to the public
DNS address of the Terraform-defined AWS instance named "db". It
is possible to export complex data types like maps and lists as
well:

```hcl
output "addresses" {
  value = ["${aws_instance.web.*.public_dns}"]
}
```

## Description

The `output` block configures a single output variable. Multiple
output variables can be configured with multiple output blocks.
The `NAME` given to the output block is the name used to reference
the output variable. It must conform to Terraform variable naming
conventions if it is to be used as an input to other modules.

Within the block (the `{ }`) is configuration for the output.
These are the parameters that can be set:

- `value` (required) - The value of the output. This can be a string, list, or
  map. This usually includes an interpolation since outputs that are static
  aren't usually useful.

- `description` (optional) - A human-friendly description for the output. This
  is primarily for documentation for users using your Terraform configuration. A
  future version of Terraform will expose these descriptions as part of some
  Terraform CLI command.

- `depends_on` (list of strings) - Explicit dependencies that this output has.
  These dependencies will be created before this output value is processed. The
  dependencies are in the format of `TYPE.NAME`, for example `aws_instance.web`.

- `sensitive` (optional, boolean) - See below.

## Syntax

The full syntax is:

```text
output NAME {
  value = VALUE
}
```

## Sensitive Outputs

Outputs can be marked as containing sensitive material by setting the
`sensitive` attribute to `true`, like this:

```hcl
output "sensitive" {
  sensitive = true
  value     = VALUE
}
```

When outputs are displayed on-screen following a `terraform apply` or
`terraform refresh`, sensitive outputs are redacted, with `<sensitive>`
displayed in place of their value.

### Limitations of Sensitive Outputs

- The values of sensitive outputs are still stored in the Terraform state, and
  available using the `terraform output` command, so cannot be relied on as a
  sole means of protecting values.

- Sensitivity is not tracked internally, so if the output is interpolated in
  another module into a resource, the value will be displayed.
