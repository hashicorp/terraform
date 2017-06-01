---
layout: "docs"
page_title: "State: Environments"
sidebar_current: "docs-state-env"
description: |-
  Terraform stores state which caches the known state of the world the last time Terraform ran.
---

# State Environments

An environment is a state namespace, allowing a single folder of Terraform
configurations to manage multiple distinct infrastructure resources.

Terraform state determines what resources it manages based on what
exists in the state. This is how `terraform plan` determines what isn't
created, what needs to be updated, etc. The full details of state can be
found on the [purpose page](/docs/state/purpose.html).

Environments are a way to create multiple states that contain
their own data so a single set of Terraform configurations can manage
multiple distinct sets of resources.

Environments are currently supported by the following backends:

 * [Consul](/docs/backends/types/consul.html)
 * [S3](/docs/backends/types/s3.html)

## Using Environments

Terraform starts with a single environment named "default". This
environment is special both because it is the default and also because
it cannot ever be deleted. If you've never explicitly used environments, then
you've only ever worked on the "default" environment.

Environments are managed with the `terraform env` set of commands. To
create a new environment and switch to it, you can use `terraform env new`,
to switch environments you can use `terraform env select`, etc.

For example, creating an environment:

```text
$ terraform env new bar
Created and switched to environment "bar"!

You're now on a new, empty environment. Environments isolate their state,
so if you run "terraform plan" Terraform will not see any existing state
for this configuration.
```

As the command says, if you run `terraform plan`, Terraform will not see
any existing resources that existed on the default (or any other) environment.
**These resources still physically exist,** but are managed by another
Terraform environment.

## Current Environment Interpolation

Within your Terraform configuration, you may reference the current environment
using the `${terraform.env}` interpolation variable. This can be used anywhere
interpolations are allowed.

Referencing the current environment is useful for changing behavior based
on the environment. For example, for non-default environments, it may be useful
to spin up smaller cluster sizes. You can do this:

```hcl
resource "aws_instance" "example" {
  count = "${terraform.env == "default" ? 5 : 1}"

  # ... other fields
}
```

Another popular use case is using the environment as part of naming or
tagging behavior:

```hcl
resource "aws_instance" "example" {
  tags { Name = "web - ${terraform.env}" }

  # ... other fields
}
```

## Best Practices

An environment can be used to manage the difference between development,
staging, and production, but it **should not** be treated as the only isolation
mechanism. As Terraform configurations get larger, it's much more
manageable and safer to split one large configuration into many
smaller ones linked together with `terraform_remote_state` data sources. This
allows teams to delegate ownership and reduce the blast radius of changes.
For **each smaller configuration**, you can use environments to model the
differences between development, staging, and production. However, if you have
one large Terraform configuration, it is riskier and not recommended to use
environments to model those differences.

The [terraform_remote_state](/docs/providers/terraform/d/remote_state.html)
resource accepts an `environment` name to target. Therefore, you can link
together multiple independently managed Terraform configurations with the same
environment easily. But, they can also have different environments.

While environments are available to all,
[Terraform Enterprise](https://www.hashicorp.com/products/terraform/)
provides an interface and API for managing sets of configurations linked
with `terraform_remote_state` and viewing them all as a single environment.

Environments alone are useful for isolating a set of resources to test
changes during development. For example, it is common to associate a
branch in a VCS with an environment so new features can be developed
without affecting the default environment.

Future Terraform versions and environment enhancements will enable
Terraform to track VCS branches with an environment to help verify only certain
branches can make changes to a Terraform environment.

## Environments Internals

Environments are technically equivalent to renaming your state file. They
aren't any more complex than that. Terraform wraps this simple notion with
a set of protections and support for remote state.

For local state, Terraform stores the state environments in a folder
`terraform.tfstate.d`. This folder should be committed to version control
(just like local-only `terraform.tfstate`).

For [remote state](/docs/state/remote.html), the environments are stored
directly in the configured [backend](/docs/backends). For example, if you
use [Consul](/docs/backends/types/consul.html), the environments are stored
by suffixing the state path with the environment name. To ensure that
environment names are stored correctly and safely in all backends, the name
must be valid to use in a URL path segment without escaping.

The important thing about environment internals is that environments are
meant to be a shared resource. They aren't a private, local-only notion
(unless you're using purely local state and not committing it).

The "current environment" name is stored only locally in the ignored
`.terraform` directory. This allows multiple team members to work on
different environments concurrently.
