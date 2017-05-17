---
layout: "docs"
page_title: "Command: console"
sidebar_current: "docs-commands-console"
description: |-
  The `terraform console` command creates an interactive console for using [interpolations](/docs/configuration/interpolation.html).
---

# Command: console

The `terraform console` command creates an interactive console for
using [interpolations](/docs/configuration/interpolation.html).

## Usage

Usage: `terraform console [options] [dir]`

This opens an interactive console for experimenting with interpolations.
This is useful for testing interpolations before using them in configurations
as well as interacting with an existing [state](/docs/state/index.html).

If a state file doesn't exist, the console still works and can be used
to experiment with supported interpolation functions. Try entering some basic
math such as `1 + 5` to see.

The `dir` argument can be used to open a console for a specific Terraform
configuration directory. This will load any state from that directory as
well as the configuration. This defaults to the current working directory.
The `console` command does not require Terraform state or configuration
to function.

The command-line flags are all optional. The list of available flags are:

* `-state=path` - Path to the state file. Defaults to `terraform.tfstate`.
  A state file doesn't need to exist.

You can close the console with the `exit` command or by using Control-C
or Control-D.

## Scripting

The `terraform console` command can be used in non-interactive scripts
by piping newline-separated commands to it. Only the output from the
final command is outputted unless an error occurs earlier.

An example is shown below:

```shell
$ echo "1 + 5" | terraform console
6
```

## Remote State

The `terraform console` command will read configured state even if it
is [remote](/docs/state/remote.html). This is great for scripting
state reading in CI environments or other remote scenarios.

After configuring remote state, run a `terraform remote pull` command
to sync state locally. The `terraform console` command will use this
state for operations.

Because the console currently isn't able to modify state in any way,
this is a one way operation and you don't need to worry about remote
state conflicts in any way.
