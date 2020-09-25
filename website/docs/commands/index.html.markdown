---
layout: "docs"
page_title: "Commands"
sidebar_current: "docs-commands"
description: |-
  Terraform is controlled via a very easy to use command-line interface (CLI). Terraform is only a single command-line application: terraform. This application then takes a subcommand such as "apply" or "plan". The complete list of subcommands is in the navigation to the left.
---

# Terraform Commands (CLI)

> For a hands-on tutorial, try the [Get Started](https://learn.hashicorp.com/terraform/getting-started/intro?utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS) track on HashiCorp Learn.

Terraform is controlled via a very easy to use command-line interface (CLI).
Terraform is only a single command-line application: terraform. This application
then takes a subcommand such as "apply" or "plan". The complete list of subcommands
is in the navigation to the left.

The terraform CLI is a well-behaved command line application. In erroneous cases,
a non-zero exit status will be returned. It also responds to -h and --help as you'd
most likely expect.

To view a list of the available commands at any time, just run terraform with no arguments:

```text
Usage: terraform [global options] <subcommand> [args]

The available commands for execution are listed below.
The most common, useful commands are shown first, followed by
less common or more advanced commands. If you're just getting
started with Terraform, stick with the common commands. For the
other commands, please read the help and docs before usage.

Common commands:
    apply              Builds or changes infrastructure
    console            Interactive console for Terraform interpolations
    destroy            Destroy Terraform-managed infrastructure
    env                Workspace management
    fmt                Rewrites config files to canonical format
    get                Download and install modules for the configuration
    graph              Create a visual graph of Terraform resources
    import             Import existing infrastructure into Terraform
    init               Initialize a Terraform working directory
    login              Obtain and save credentials for a remote host
    logout             Remove locally-stored credentials for a remote host
    output             Read an output from a state file
    plan               Generate and show an execution plan
    providers          Prints a tree of the providers used in the configuration
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


Global options (use these before the subcommand, if any):
    -chdir=DIR         Switch to a different working directory before executing
                       the given subcommand.
    -help              Show this help output, or the help for a specified
                       subcommand.
    -version           An alias for the "version" subcommand.
```

To get help for any specific command, use the -help option to the relevant
subcommand. For example, to see help about the graph subcommand:

```text
$ terraform graph -help
Usage: terraform graph [options] PATH

  Outputs the visual graph of Terraform resources. If the path given is
  the path to a configuration, the dependency graph of the resources are
  shown. If the path is a plan file, then the dependency graph of the
  plan itself is shown.

  The graph is outputted in DOT format. The typical program that can
  read this format is GraphViz, but many web services are also available
  to read this format.
```

## Switching working directory with `-chdir`

The usual way to run Terraform is to first switch to the directory containing
the `.tf` files for your root module (for example, using the `cd` command), so
that Terraform will find those files automatically without any extra arguments.

In some cases though — particularly when wrapping Terraform in automation
scripts — it can be convenient to run Terraform from a different directory than
the root module directory. To allow that, Terraform supports a global option
`-chdir=...` which you can include before the name of the subcommand you intend
to run:

```
terraform -chdir=environments/production apply
```

The `chdir` option instructs Terraform to change its working directory to the
given directory before running the given subcommand. This means that any files
that Terraform would normally read or write in the current working directory
will be read or written in the given directory instead.

There are two exceptions where Terraform will use the original working directory
even when you specify `-chdir=...`:

* Settings in the [CLI Configuration](cli-config.html) are not for a specific
  subcommand and Terraform processes them before acting on the `-chdir`
  option.

* In case you need to use files from the original working directory as part
  of your configuration, a reference to `path.cwd` in the configuration will
  produce the original working directory instead of the overridden working
  directory. Use `path.root` to get the root module directory.

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

Alternatively, settings in
[the CLI configuration file](/docs/commands/cli-config.html) can be used to
disable checkpoint features. The following checkpoint-related settings are
supported in this file:

* `disable_checkpoint` - set to `true` to disable checkpoint calls
  entirely. This is similar to the `CHECKPOINT_DISABLE` environment variable
  described above.

* `disable_checkpoint_signature` - set to `true` to disable the use of an
  anonymous signature in checkpoint requests. This allows Terraform to check
  for security bulletins but does not send the anonymous signature in these
  requests.

[The Checkpoint client code](https://github.com/hashicorp/go-checkpoint) used
by Terraform is available for review by any interested party.
