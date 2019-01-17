---
layout: "docs"
page_title: "Override Files - 0.11 Configuration Language"
sidebar_current: "docs-conf-old-override"
description: |-
  Terraform loads all configuration files within a directory and appends them together. Terraform also has a concept of overrides, a way to create files that are loaded last and merged into your configuration, rather than appended.
---

# Override Files

Terraform loads all configuration files within a directory and
appends them together. Terraform also has a concept of _overrides_,
a way to create files that are loaded last and _merged_ into your
configuration, rather than appended.

Overrides have a few use cases:

  * Machines (tools) can create overrides to modify Terraform
    behavior without having to edit the Terraform configuration
    tailored to human readability.

  * Temporary modifications can be made to Terraform configurations
    without having to modify the configuration itself.

Overrides names must be `override` or end in `_override`, excluding
the extension. Examples of valid override files are `override.tf`,
`override.tf.json`, `temp_override.tf`.

Override files are loaded last in alphabetical order.

Override files can be in Terraform syntax or JSON, just like non-override
Terraform configurations.

## Example

If you have a Terraform configuration `example.tf` with the contents:

```hcl
resource "aws_instance" "web" {
  ami = "ami-408c7f28"
}
```

And you created a file `override.tf` with the contents:

```hcl
resource "aws_instance" "web" {
  ami = "foo"
}
```

Then the AMI for the one resource will be replaced with "foo". Note
that the override syntax can be Terraform syntax or JSON. You can
mix and match syntaxes without issue.
