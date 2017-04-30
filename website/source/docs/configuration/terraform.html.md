---
layout: "docs"
page_title: "Configuring Terraform"
sidebar_current: "docs-config-terraform"
description: |-
  The `terraform` configuration section is used to configure Terraform itself, such as requiring a minimum Terraform version to execute a configuration.
---

# Terraform Configuration

The `terraform` configuration section is used to configure Terraform itself,
such as requiring a minimum Terraform version to execute a configuration.

This page assumes you're familiar with the
[configuration syntax](/docs/configuration/syntax.html)
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

Prior to
[terraform-0.9.0](https://github.com/hashicorp/terraform/blob/v0.9.0/CHANGELOG.md)
(released 2017-03-15), the only allowed configuration within the `terraform`
block was `required_version` (documented below). Since `terraform-0.9.0`,
`terraform` blocks may also be used to configure a specific Terraform
[backend](/docs/backends/index.html). Use of a `backend` stanza within a
`terraform` block is documented in
["Backend Configuration"](/docs/backends/config.html).

**No value within the `terraform` block can use interpolations.** The
`terraform` block is loaded very early in the execution of Terraform
and interpolations are not yet available.

## Specifying a Required Terraform Version

The `required_version` setting specifies a set of Terraform version number
constraints that must be met in order for Terraform to perform operations on
the current configuration. The setting can be used to require a specific
version of Terraform. If the running Terraform version does not meet these
constraints, an error message is printed to `stderr`, `terraform` stops
processing, and then exits with an error status.

For example, a `required_version` setting with a value of `>= 0.9.4` would
have the following effect when attempting to process the Terraform config with
an older `terraform` version:

```sh
    $ terraform --version
    Terraform v0.8.7

    $ terraform plan
    The currently running version of Terraform doesn't meet the
    version requirements explicitly specified by the configuration.
    Please use the required version or update the configuration.
    Note that version requirements are usually set for a reason, so
    we recommend verifying with whoever set the version requirements
    prior to making any manual changes.

      Module: root
      Required version: 0.9.4
      Current version: 0.8.7
```

When [modules](/docs/configuration/modules.html) are used, all Terraform
version requirements specified by the complete module tree must be
satisified. This means that the `required_version` setting can be used
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
