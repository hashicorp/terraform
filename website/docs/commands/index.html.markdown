---
layout: "docs"
page_title: "Commands"
sidebar_current: "docs-commands"
description: |-
  Terraform is controlled via a very easy to use command-line interface (CLI). Terraform is only a single command-line application: terraform. This application then takes a subcommand such as "apply" or "plan". The complete list of subcommands is in the navigation to the left.
---

# Terraform Commands (CLI)

Terraform is controlled via a very easy to use command-line interface (CLI).
Terraform is only a single command-line application: terraform. This application
then takes a subcommand such as "apply" or "plan". The complete list of subcommands
is in the navigation to the left.

The terraform CLI is a well-behaved command line application. In erroneous cases,
a non-zero exit status will be returned. It also responds to -h and --help as you'd
most likely expect.

To view a list of the available commands at any time, just run terraform with no arguments:

```text
$ terraform
Usage: terraform [--version] [--help] <command> [args]

The available commands for execution are listed below.
The most common, useful commands are shown first, followed by
less common or more advanced commands. If you're just getting
started with Terraform, stick with the common commands. For the
other commands, please read the help and docs before usage.

Common commands:
    apply              Builds or changes infrastructure
    console            Interactive console for Terraform interpolations
    destroy            Destroy Terraform-managed infrastructure
    fmt                Rewrites config files to canonical format
    get                Download and install modules for the configuration
    graph              Create a visual graph of Terraform resources
    import             Import existing infrastructure into Terraform
    init               Initialize a new or existing Terraform configuration
    output             Read an output from a state file
    plan               Generate and show an execution plan
    providers          Prints a tree of the providers used in the configuration
    push               Upload this Terraform module to Terraform Enterprise to run
    refresh            Update local state file against real resources
    show               Inspect Terraform state or plan
    taint              Manually mark a resource for recreation
    untaint            Manually unmark a resource as tainted
    validate           Validates the Terraform files
    version            Prints the Terraform version
    workspace          Workspace management

All other commands:
    debug              Debug output management (experimental)
    force-unlock       Manually unlock the terraform state
    state              Advanced state management
```

To get help for any specific command, pass the -h flag to the relevant subcommand. For example,
to see help about the graph subcommand:

```text
$ terraform graph -h
Usage: terraform graph [options] PATH

  Outputs the visual graph of Terraform resources. If the path given is
  the path to a configuration, the dependency graph of the resources are
  shown. If the path is a plan file, then the dependency graph of the
  plan itself is shown.

  The graph is outputted in DOT format. The typical program that can
  read this format is GraphViz, but many web services are also available
  to read this format.
```

## Shell Tab-completion

If you use either `bash` or `zsh` as your command shell, Terraform can provide
tab-completion support for all command names and (at this time) _some_ command
arguments.

To add the necessary commands to your shell profile, run the following command:

```bash
terraform -install-autocomplete
```

After installation, it is necessary to restart your shell or to re-read its
profile script before completion will be activated.

To uninstall the completion hook, assuming that it has not been modified
manually in the shell profile, run the following command:

```bash
terraform -uninstall-autocomplete
```

Currently not all of Terraform's subcommands have full tab-completion support
for all arguments. We plan to improve tab-completion coverage over time.

## Upgrade and Security Bulletin Checks

The Terraform CLI commands interact with the HashiCorp service
[Checkpoint](https://checkpoint.hashicorp.com/) to check for the availability
of new versions and for critical security bulletins about the current version.

One place where the effect of this can be seen is in `terraform version`, where
it is used by default to indicate in the output when a newer version is
available.

Only anonymous information, which cannot be used to identify the user or host,
is sent to Checkpoint. An anonymous ID is sent which helps de-duplicate warning
messages. Both the anonymous id and the use of checkpoint itself are completely
optional and can be disabled.

Checkpoint itself can be entirely disabled for all HashiCorp products by
setting the environment variable `CHECKPOINT_DISABLE` to any non-empty value.

Alternatively, settings in Terraform's global configuration file can be used
to disable checkpoint features. On Unix systems this file is named
`.terraformrc` and is placed within the home directory of the user running
Terraform. On Windows, this file is named `terraform.rc` and is and is placed
in the current user's _Application Data_ folder.

The following checkpoint-related settings are supported in this file:

* `disable_checkpoint` - set to `true` to disable checkpoint calls
  entirely. This is similar to the `CHECKPOINT_DISABLE` environment variable
  described above.

* `disable_checkpoint_signature` - set to `true` to disable the use of an
  anonymous signature in checkpoint requests. This allows Terraform to check
  for security bulletins but does not send the anonymous signature in these
  requests.

[The Checkpoint client code](https://github.com/hashicorp/go-checkpoint) used
by Terraform is available for review by any interested party.
