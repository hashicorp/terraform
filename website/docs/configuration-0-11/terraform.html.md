---
layout: "docs"
page_title: "Terraform Settings - 0.11 Configuration Language"
sidebar_current: "docs-conf-old-terraform"
description: |-
  The `terraform` configuration section is used to configure Terraform itself, such as requiring a minimum Terraform version to execute a configuration.
---

# Terraform Settings

-> **Note:** This page is about Terraform 0.11 and earlier. For Terraform 0.12
and later, see
[Configuration Language: Terraform Settings](../configuration/terraform.html).

The `terraform` configuration section is used to configure Terraform itself,
such as requiring a minimum Terraform version to execute a configuration.

This page assumes you're familiar with the
[configuration syntax](./syntax.html)
already.

## Example

Terraform configuration looks like the following:

```hcl
terraform {
  required_version = "> 0.7.0"
}
```

## Description

The `terraform` block configures the behavior of Terraform itself.

The currently only allowed configurations within this block are
`required_version` and `backend`.

`required_version` specifies a set of version constraints
that must be met to perform operations on this configuration. If the
running Terraform version doesn't meet these constraints, an error
is shown. See the section below dedicated to this option.

See [backends](/docs/backends/index.html) for more detail on the `backend`
configuration.

**No value within the `terraform` block can use interpolations.** The
`terraform` block is loaded very early in the execution of Terraform
and interpolations are not yet available.

## Specifying a Required Terraform Version

The `required_version` setting can be used to require a specific version
of Terraform. If the running version of Terraform doesn't match the
constraints specified, Terraform will show an error and exit.

When [modules](./modules.html) are used, all Terraform
version requirements specified by the complete module tree must be
satisfied. This means that the `required_version` setting can be used
by a module to require that all consumers of a module also use a specific
version.

The value of this configuration is a comma-separated list of constraints.
A constraint is an operator followed by a version, such as `> 0.7.0`.
Constraints support the following operations:

- `=` (or no operator): exact version equality

- `!=`: version not equal

- `>`, `>=`, `<`, `<=`: version comparison, where "greater than" is a larger
  version number

- `~>`: pessimistic constraint operator. Example: for `~> 0.9`, this means
  `>= 0.9, < 1.0`. Example: for `~> 0.8.4`, this means `>= 0.8.4, < 0.9`

For modules, a minimum version is recommended, such as `> 0.8.0`. This
minimum version ensures that a module operates as expected, but gives
the consumer flexibility to use newer versions.

## Syntax

The full syntax is:

```text
terraform {
  required_version = VALUE
}
```
