---
layout: "docs"
page_title: "Command: console"
sidebar_current: "docs-commands-console"
description: |-
  The `terraform console` command provides an interactive console for
  evaluting expressions.
---

# Command: console

The `terraform console` command provides an interactive console for
evaluating [expressions](/docs/configuration/expressions.html).

## Usage

Usage: `terraform console [options] [dir]`

This command provides an interative command-line console for evaluating and
experimenting with [expressions](/docs/configuration/expressions.html).
This is useful for testing interpolations before using them in configurations,
and for interacting with any values currently saved in
[state](/docs/state/index.html).

If the current state is empty or has not yet been created, the console can be
used to experiment with the expression syntax and
[built-in functions](/docs/configuration/functions.html).

The `dir` argument specifies the directory of the root module to use.
If a path is not specified, the current working directory is used.

The supported options are:

* `-state=path` - Path to a local state file. Expressions will be evaluated
  using values from this state file. If not specified, the state associated
  with the current [workspace](/docs/state/workspaces.html) is used.

You can close the console with the `exit` command or by pressing Control-C
or Control-D.

## Scripting

The `terraform console` command can be used in non-interactive scripts
by piping newline-separated commands to it. Only the output from the
final command is printed unless an error occurs earlier.

For example:

```shell
$ echo "1 + 5" | terraform console
6
```

## Remote State

If [remote state](/docs/state/remote.html) is used by the current backend,
Terraform will read the state for the current workspace from the backend
before evaluating any expressions.
