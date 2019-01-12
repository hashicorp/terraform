---
layout: "docs"
page_title: "Terraform Settings - Configuration Language"
sidebar_current: "docs-config-terraform"
description: |-
  The "terraform" configuration section is used to configure some behaviors
  of Terraform itself.
---

# Terraform Settings

The special `terraform` configuration block type is used to configure some
behaviors of Terraform itself, such as requiring a minimum Terraform version to
apply your configuration.

## Terraform Block Syntax

Terraform-specific settings are gathered together into `terraform` blocks:

```hcl
terraform {
  # ...
}
```

Each `terraform` block can contain a number of settings related to Terraform's
behavior. Within a `terraform` block, only constant values can be used;
arguments may not refer to named objects such as resources, input variables,
etc, and may not use any of the Terraform language built-in functions.

The various options supported within a `terraform` block are described in the
following sections.

## Configuring a Terraform Backend

The selected _backend_ for a Terraform configuration defines exactly where
and how operations are performed, where [state](/docs/state/index.html) is
stored, etc. Most non-trivial Terraform configurations will have a backend
configuration that configures a remote backend to allow collaboration within
a team.

A backend configuration is given in a nested `backend` block within a
`terraform` block:

```hcl
terraform {
  backend "s3" {
    # (backend-specific settings...)
  }
}
```

More information on backend configuration can be found in
[the _Backends_ section](/docs/backends/index.html).

## Specifying a Required Terraform Version

The `required_version` setting can be used to constrain which versions of
the Terraform CLI can be used with your configuration. If the running version of
Terraform doesn't match the constraints specified, Terraform will produce
an error and exit without taking any further actions.

When you use [child modules](./modules.html), each module
can specify its own version requirements. The requirements of all modules
in the tree must be satisfied.

Use Terraform version constraints in a collaborative environment to
ensure that everyone is using a spceific Terraform version, or using at least
a minimum Terraform version that has behavior expected by the configuration.

The `required_version` setting applies only to the version of Terraform CLI.
Various behaviors of Terraform are actually implemented by Terraform Providers,
which are released on a cycle independent of Terraform CLI and of each other.
Use [provider version constraints](./providers.html#provider-versions)
to make similar constraints on which provider versions may be used.

The value for `required_version` is a string containing a comma-separated
list of constraints. Each constraint is an operator followed by a version
number, such as `> 0.12.0`. The following constraint operators are allowed:

* `=` (or no operator): exact version equality

* `!=`: version not equal

* `>`, `>=`, `<`, `<=`: version comparison, where "greater than" is a larger
  version number

* `~>`: pessimistic constraint operator, constraining both the oldest and
  newest version allowed. For example, `~> 0.9` is equivalent to
  `>= 0.9, < 1.0`, and `~> 0.8.4`, is equivalent to `>= 0.8.4, < 0.9`

Re-usable modules should constrain only the minimum allowed version, such
as `>= 0.12.0`. This specifies the earliest version that the module is
compatible with while leaving the user of the module flexibility to upgrade
to newer versions of Terraform without altering the module.

## Specifying Required Provider Versions

The `required_providers` setting is a map specifying a version constraint for
each provider required by your configuration.

This is one of several ways to define
[provider version constraints](./providers.html#provider-versions),
and is particularly suited to re-usable modules that expect a provider
configuration to be provided by their caller but still need to impose a
minimum version for that provider.

```hcl
terraform {
  required_providers = {
    aws = ">= 1.0.0"
  }
}
```

Re-usable modules should constrain only the minimum allowed version, such
as `>= 1.0.0`. This specifies the earliest version that the module is
compatible with while leaving the user of the module flexibility to upgrade
to newer versions of the provider without altering the module.
