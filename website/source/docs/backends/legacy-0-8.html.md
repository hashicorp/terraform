---
layout: "docs"
page_title: "Backends: Migrating From 0.8.x and Earlier"
sidebar_current: "docs-backends-migrate"
description: |-
  A "backend" in Terraform determines how state is loaded and how an operation such as `apply` is executed. This abstraction enables non-local file state storage, remote execution, etc.
---

# Backend & Legacy Remote State

Prior to Terraform 0.9.0 backends didn't exist and remote state management
was done in a completely different way. This page documents how you can
migrate to the new backend system and any considerations along the way.

Migrating to the new backends system is extremely simple. The only complex
case is if you had automation around configuring remote state. An existing
environment can be configured to use the new backend system after just
a few minutes of reading.

For the remainder of this document, the remote state system prior to
Terraform 0.9.0 will be called "legacy remote state."

-> **Note:** This page is targeted at users who used remote state prior
to version 0.9.0 and need to upgrade their environments. If you didn't
use remote state, you can ignore this document.

## Backwards Compatibility

In version 0.9.0, Terraform knows how to load and continue working with
legacy remote state. A warning is shown guiding you to this page, but
otherwise everything continues to work without changing any configuration.

Backwards compatibility with legacy remote state environments will be
removed in Terraform 0.11.0, or two major releases after 0.10.0. Starting
in 0.10.0, detection will remain but users will be _required_ to update
their configurations to use backends. In Terraform 0.11.0, detection and
loading will be completely removed.

For the short term, you may continue using Terraform with version 0.9.0
as you have been. However, you should begin planning to update your configuration
very soon. As you'll see, this process is very easy.

## Migrating to Backends

You should begin by reading the [complete backend documentation](/docs/backends)
section. This section covers in detail how you use and configure backends.

Next, perform the following steps to migrate. These steps will also guide
you through backing up your existing remote state just in case things don't
go as planned.

1. **With the older Terraform version (version 0.8.x),** run `terraform remote pull`. This
will cache the latest legacy remote state data locally. We'll use this for
a backup in case things go wrong.

1. Backup your `.terraform/terraform.tfstate` file. This contains the
cache we just pulled. Please copy this file to a location outside of your
Terraform module.

1. [Configure your backend](/docs/backends/config.html) in your Terraform
configuration. The backend type is the same backend type as you used with
your legacy remote state. The configuration should be setup to match the
same configuration you used with remote state.

1. [Run the init command](/docs/backends/init.html). This is an interactive
process that will guide you through migrating your existing remote state
to the new backend system. During this step, Terraform may ask if you want
to copy your old remote state into the newly configured backend. If you
configured the identical backend location, you may say no since it should
already be there.

1. Verify your state looks good by running `terraform plan` and seeing if
it detects your infrastructure. Advanced users may run `terraform state pull`
which will output the raw contents of your state file to your console. You
can compare this with the file you saved. There may be slight differences in
the serial number and version data, but the raw data should be almost identical.

After the above steps, you're good to go! Everyone who uses the same
Terraform state should copy the same steps above. The only difference is they
may be able to skip the configuration step if you're sharing the configuration.

At this point, **older Terraform versions will stop working.** Terraform
will prevent itself from working with state written with a higher version
of Terraform. This means that even other users using an older version of
Terraform with the same configured remote state location will no longer
be able to work with the environment. Everyone must upgrade.

## Rolling Back

If the migration fails for any reason: your states look different, your
plan isn't what you expect, you're getting errors, etc. then you may roll back.

After rolling back, please [report an issue](https://github.com/hashicorp/terraform)
so that we may resolve anything that may have gone wrong for you.

To roll back, follow the steps below using Terraform 0.8.x or earlier:

1. Remove the backend configuration from your Terraform configuration file.

2. Remove any "terraform.tfstate" files (including backups). If you believe
these may contain important data, you may back them up. Going with the assumption
that you started this migration guide with working remote state, these files
shouldn't contain anything of value.

3. Copy the `.terraform/terraform.tfstate` file you backed up back into
the same location.

And you're rolled back. If your backend migration worked properly and was
able to update your remote state, **then this will not work**. Terraform
prevents writing state that was written with a higher Terraform version
or a later serial number.

**If you're absolutely certain you want to restore your state backup**,
then you can use `terraform remote push -force`. This is extremely dangerous
and you will lose any changes that were in the remote location.

## Configuration Automation

The `terraform remote config` command has been replaced with
`terraform init`. The new command is better in many ways by allowing file-based
configuration, automatic state migration, and more.

You should be able to very easily migrate `terraform remote config`
scripting to the new `terraform init` command.

The new `terraform init` command takes a `-backend-config` flag which is
either an HCL file or a string in the format of `key=value`. This configuration
is merged with the backend configuration in your Terraform files.
This lets you keep secrets out of your actual configuration.
We call this "partial configuration" and you can learn more in the
docs on [configuring backends](/docs/backends/config.html).

This does introduce an extra step: your automation must generate a
JSON file (presumably JSON is easier to generate from a script than HCL
and HCL is compatible) to pass into `-backend-config`.
