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

```
$ terraform
usage: terraform [--version] [--help] <command> [<args>]

Available commands are:
    apply      Builds or changes infrastructure
    destroy    Destroy Terraform-managed infrastructure
    get        Download and install modules for the configuration
    graph      Create a visual graph of Terraform resources
    init       Initializes Terraform configuration from a module
    output     Read an output from a state file
    plan       Generate and show an execution plan
    refresh    Update local state file against real resources
    remote     Configure remote state storage
    show       Inspect Terraform state or plan
    taint      Manually mark a resource for recreation
    version    Prints the Terraform version
```

To get help for any specific command, pass the -h flag to the relevant subcommand. For example,
to see help about the members subcommand:

```
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

