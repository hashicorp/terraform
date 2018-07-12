---
layout: "guides"
page_title: "Running Terraform in Automation - Guides"
sidebar_current: "guides-running-terraform-in-automation"
description: |-
  Terraform can, with some caveats, be run in automated processes such as
  continuous delivery pipelines. Ths guide describes some techniques for
  doing so and some gotchas to watch out for.
---

# Running Terraform in Automation

~> **This is an advanced guide!** When getting started with Terraform, it's
recommended to use it locally from the command line. Automation can become
valuable once Terraform is being used regularly in production, or by a larger
team, but this guide assumes familiarity with the the normal, local CLI
workflow.

For teams that use Terraform as a key part of a change management and
deployment pipeline, it can be desirable to orchestrate Terraform runs in some
sort of automation in order to ensure consistency between runs, and provide
other interesting features such as integration with version control hooks.

Automation of Terraform can come in various forms, and to varying degrees.
Some teams continue to run Terraform locally but use _wrapper scripts_ to
prepare a consistent working directory for Terraform to run in, while other
teams run Terraform entirely within an orchestration tool such as Jenkins.

This guide covers some things that should be considered when implementing
such automation, both to ensure safe operation of Terraform and to accommodate
some current limitations in Terraform's workflow that require careful
attention in automation.

The guide assumes that Terraform will be running in an _non-interactive_
environment, where it is not possible to prompt for input at the terminal.
This is not necessarily true for wrapper scripts, but is often true when
running in orchestration tools.

This is a general guide, giving an overview of things to consider when
implementing orchestration of Terraform. Due to its general nature, it is not
possible to go into specifics about any particular tools, though other
tool-specific guides may be produced later if best practices emerge around
such a tool.

## Automated Workflow Overview

When running Terraform in automation, the focus is usually on the core
plan/apply cycle. The main path, then, is broadly the same as for CLI
usage:

1. Initialize the Terraform working directory.
2. Produce a plan for changing resources to match the current configuration.
3. Have a human operator review that plan, to ensure it is acceptable.
4. Apply the changes described by the plan.

Steps 1, 2 and 4 can be carried out using the familiar Terraform CLI commands,
with some additional options:

* `terraform init -input=false` to initialize the working directory.
* `terraform plan -out=tfplan -input=false` to create a plan and save it to the local file `tfplan`.
* `terraform apply -input=false tfplan` to apply the plan stored in the file `tfplan`.

The `-input=false` option indicates that Terraform should not attempt to
prompt for input, and instead expect all necessary values to be provided by
either configuration files or the command line. It may therefore be necessary
to use the `-var` and `-var-file` options on `terraform plan` to specify any
variable values that would traditionally have been manually-entered under
interactive usage.

It is strongly recommended to use a backend that supports
[remote state](/docs/state/remote.html), since that allows Terraform to
automatically save the state in a persistent location where it can be found
and updated by subsequent runs. Selecting a backend that supports
[state locking](/docs/state/locking.html) will additionally provide safety
against race conditions that can be caused by concurrent Terraform runs.

## Controlling Terraform Output in Automation

By default, some Terraform commands conclude by presenting a description
of a possible next step to the user, often including a specific command
to run next.

An automation tool will often abstract away the details of exactly which
commands are being run, causing these messages to be confusing and
un-actionable, and possibly harmful if they inadvertently encourage a user to
bypass the automation tool entirely.

When the environment variable `TF_IN_AUTOMATION` is set to any non-empty
value, Terraform makes some minor adjustments to its output to de-emphasize
specific commands to run. The specific changes made will vary over time,
but generally-speaking Terraform will consider this variable to indicate that
there is some wrapping application that will help the user with the next
step.

To reduce complexity, this feature is implemented primarily for the main
workflow commands described above. Other ancillary commands may still produce
command line suggestions, regardless of this setting.

## Plan and Apply on different machines

When running in an orchestration tool, it can be difficult or impossible to
ensure that the `plan` and `apply` subcommands are run on the same machine,
in the same directory, with all of the same files present.

Running `plan` and `apply` on different machines requires some additional
steps to ensure correct behavior. A robust strategy is as follows:

* After `plan` completes, archive the entire working directory, including the
  `.terraform` subdirectory created during `init`, and save it somewhere
  where it will be available to the apply step. A common choice is as a
  "build artifact" within the chosen orchestration tool.
* Before running `apply`, obtain the archive created in the previous step
  and extract it _at the same absolute path_. This re-creates everything
  that was present after plan, avoiding strange issues where local files
  were created during the plan step.

Terraform currently makes some assumptions which must be accommodated by
such an automation setup:

* The saved plan file can contain absolute paths to child modules and other
  data files referred to by configuration. Therefore it is necessary to ensure
  that the archived configuration is extracted at an identical absolute path.
  This is most commonly achieved by running Terraform in some sort of isolation,
  such as a Docker container, where the filesystem layout can be controlled.
* Terraform assumes that the plan will be applied on the same operating system
  and CPU architecture as where it was created. For example, this means that
  it is not possible to create a plan on a Windows computer and then apply it
  on a Linux server.
* Terraform expects the provider plugins that were used to produce a
  plan to be available and identical when the plan is applied, to ensure
  that the plan is interpreted correctly. An error will be produced if
  Terraform or any plugins are upgraded between creating and applying a plan.
* Terraform can't automatically detect if the credentials used to create a
  plan grant access to the same resources used to apply that plan. If using
  different credentials for each (e.g. to generate the plan using read-only
  credentials) it is important to ensure that the two are consistent
  in which account on the corresponding service they belong to.

~> The plan file contains a full copy of the configuration, the state that
the plan applies to, and any variables passed to `terraform plan`. If any of
these contain sensitive data then the archived working directory containing
the plan file should be protected accordingly. For provider authentication
credentials, it is recommended to use environment variables instead where
possible since these are _not_ included in the plan or persisted to disk
by Terraform in any other way.

## Interactive Approval of Plans

Another challenge with automating the Terraform workflow is the desire for an
interactive approval step between plan and apply. To implement this robustly,
it is important to ensure that either only one plan can be outstanding at a
time or that the two steps are connected such that approving a plan passes
along  enough information to the apply step to ensure that the correct plan is
applied, as opposed to some later plan that also exists.

Different orchestration tools address this in different ways, but generally
this is implemented via a _build pipeline_ feature, where different steps
can be applied in sequence, with later steps having access to data produced
by earlier steps. 

The recommended approach is to allow only one plan to be outstanding at a
time. When a plan is applied, any other existing plans that were produced
against the same state are invalidated, since they must now be recomputed
relative to the new state. By forcing plans to be approved (or dismissed) in
sequence, this can be avoided.

## Auto-Approval of Plans

While manual review of plans is strongly recommended for production
use-cases, it is sometimes desirable to take a more automatic approach
when deploying in pre-production or development situations.

Where manual approval is not required, a simpler sequence of commands
can be used:

* `terraform init -input=false`
* `terraform apply -input=false -auto-approve`

This variant of the `apply` command implicitly creates a new plan and then
immediately applies it. The `-auto-approve` option tells Terraform not
to require interactive approval of the plan before applying it.

~> When Terraform is empowered to make destructive changes to infrastructure,
manual review of plans is always recommended unless downtime is tolerated
in the event of unintended changes. Use automatic approval **only** with
non-critical infrastructure.

## Testing Pull Requests with `terraform plan`

`terraform plan` can be used as a way to perform certain limited verification
of the validity of a Terraform configuration, without affecting real
infrastructure. Although the plan step updates the state to match real
resources, thus ensuring an accurate plan, the updated state is _not_
persisted, and so this command can safely be used to produce "throwaway" plans
that are created only to aid in code review.

When implementing such a workflow, hooks can be used within the code review
tool in question (for example, Github Pull Requests) to trigger an orchestration
tool for each new commit under review. Terraform can be run in this case
as follows:

* `terraform plan -input=false`

As in the "main" workflow, it may be necessary to provide `-var` or `-var-file`
as appropriate. The `-out` option is not used in this scenario because a
plan produced for code review purposes will never be applied. Instead, a
new plan can be created and applied from the primary version control branch
once the change is merged.

~> Beware that passing sensitive/secret data to Terraform via
variables or via environment variables will make it possible for anyone who
can submit a PR to discover those values, so this flow must be
used with care on an open source project, or on any private project where
some or all contributors should not have direct access to credentials, etc.

## Multi-environment Deployment

Automation of Terraform often goes hand-in-hand with creating the same
configuration multiple times to produce parallel environments for use-cases
such as pre-release testing or multi-tenant infrastructure. Automation
in such a situation can help ensure that the correct settings are used for
each environment, and that the working directory is properly configured
before each operation.

The two most interesting commands for multi-environment orchestration are
`terraform init` and `terraform workspace`. The former can be used with
additional options to tailor the backend configuration for any differences
between environments, while the latter can be used to safely switch between
multiple states for the same config stored in a single backend.

Where possible, it's recommended to use a single backend configuration for
all environments and use the `terraform workspace` command to switch
between workspaces:

* `terraform init -input=false`
* `terraform workspace select QA`

In this usage model, a fixed naming scheme is used within the backend
storage to allow multiple states to exist without any further configuration.

Alternatively, the automation tool can set the environment variable
`TF_WORKSPACE` to an existing workspace name, which overrides any selection
made with the `terraform workspace select` command. Using this environment
variable is recommended only for non-interactive usage, since in a local shell
environment it can be easy to forget the variable is set and apply changes
to the wrong state.

In some more complex situations it is impossible to share the same
[backend configuration](/docs/backends/config.html) across environments. For
example, the environments may exist in entirely separate accounts within the
target service, and thus need to use different credentials or endpoints for the
backend itself. In such situations, backend configuration settings can be
overridden via
[the `-backend-config` option to `terraform init`](/docs/commands/init.html#backend-config).

## Pre-installed Plugins

In default usage, [`terraform init`](/docs/commands/init.html#backend-config)
downloads and installs the plugins for any providers used in the configuration
automatically, placing them in a subdirectory of the `.terraform` directory.
This affords a simpler workflow for straightforward cases, and allows each
configuration to potentially use different versions of plugins.

In automation environments, it can be desirable to disable this behavior
and instead provide a fixed set of plugins already installed on the system
where Terraform is running. This then avoids the overhead of re-downloading
the plugins on each execution, and allows the system administrator to control
which plugins are available.

To use this mechanism, create a directory somewhere on the system where
Terraform will run and place into it the plugin executable files. The
plugin release archives are available for download on
[releases.hashicorp.com](https://releases.hashicorp.com/). Be sure to
download the appropriate archive for the target operating system and
architecture.

After extracting the necessary plugins, the contents of the new plugin
directory will look something like this:

```
$ ls -lah /usr/lib/custom-terraform-plugins
-rwxrwxr-x 1 user user  84M Jun 13 15:13 terraform-provider-aws-v1.0.0-x3
-rwxrwxr-x 1 user user  84M Jun 13 15:15 terraform-provider-rundeck-v2.3.0-x3
-rwxrwxr-x 1 user user  84M Jun 13 15:15 terraform-provider-mysql-v1.2.0-x3
```

The version information at the end of the filenames is important so that
Terraform can infer the version number of each plugin. It is allowed to
concurrently install multiple versions of the same provider plugin,
which will then be used to satisfy
[provider version constraints](/docs/configuration/providers.html#provider-versions)
from Terraform configurations.

With this directory populated, the usual auto-download and
[plugin discovery](/docs/plugins/basics.html#installing-a-plugin)
behavior can be bypassed using the `-plugin-dir` option to `terraform init`:

* `terraform init -input=false -plugin-dir=/usr/lib/custom-terraform-plugins`

When this option is used, only the plugins in the given directory are
available for use. This gives the system administrator a high level of
control over the execution environment, but on the other hand it prevents
use of newer plugin versions that have not yet been installed into the
local plugin directory. Which approach is more appropriate will depend on
unique constraints within each organization.

Plugins can also be provided along with the configuration by creating a
`terraform.d/plugins/OS_ARCH` directory, which will be searched before
automatically downloading additional plugins. The `-get-plugins=false` flag can
be used to prevent Terraform from automatically downloading additional plugins. 

## Terraform Enterprise

As an alternative to home-grown automation solutions, Hashicorp offers
[Terraform Enterprise](https://www.hashicorp.com/products/terraform/).

Internally, Terraform Enterprise runs the same Terraform CLI commands
described above, using the same release binaries offered for download on this
site.

Terraform Enterprise builds on the core Terraform CLI functionality to add
additional features such as role-based access control, orchestration of the
plan and apply lifecycle, a user interface for reviewing and approving plans,
and much more.

It will always be possible to run Terraform via in-house automation, to
allow for usage in situations where Terraform Enterprise is not appropriate.
It is recommended to consider Terraform Enterprise as an alternative to
in-house solutions, since it provides an out-of-the-box solution that
already incorporates the best practices described in this guide and can thus
reduce time spent developing and maintaining an in-house alternative.
