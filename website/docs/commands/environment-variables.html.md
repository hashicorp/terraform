---
layout: "docs"
page_title: "Environment Variables"
sidebar_current: "docs-commands-environment-variables"
description: |-
  Terraform uses environment variables to configure various aspects of its behavior.
---

# Environment Variables

Terraform refers to a number of environment variables to customize various
aspects of its behavior. None of these environment variables are required
when using Terraform, but they can be used to change some of Terraform's
default behaviors in unusual situations, or to increase output verbosity
for debugging.

## TF_LOG

If set to any value, enables detailed logs to appear on stderr which is useful for debugging. For example:

```shell
export TF_LOG=TRACE
```

To disable, either unset it or set it to empty. When unset, logging will default to stderr. For example:

```shell
export TF_LOG=
```

For more on debugging Terraform, check out the section on [Debugging](/docs/internals/debugging.html).

## TF_LOG_PATH

This specifies where the log should persist its output to. Note that even when `TF_LOG_PATH` is set, `TF_LOG` must be set in order for any logging to be enabled. For example, to always write the log to the directory you're currently running terraform from:

```shell
export TF_LOG_PATH=./terraform.log
```

For more on debugging Terraform, check out the section on [Debugging](/docs/internals/debugging.html).

## TF_INPUT

If set to "false" or "0", causes terraform commands to behave as if the `-input=false` flag was specified. This is used when you want to disable prompts for variables that haven't had their values specified. For example:

```shell
export TF_INPUT=0
```

## TF_MODULE_DEPTH

When given a value, causes terraform commands to behave as if the `-module-depth=VALUE` flag was specified. By setting this to 0, for example, you enable commands such as [plan](/docs/commands/plan.html) and [graph](/docs/commands/graph.html) to display more compressed information.

```shell
export TF_MODULE_DEPTH=0
```

For more information regarding modules, check out the section on [Using Modules](/docs/modules/usage.html).

## TF_VAR_name

Environment variables can be used to set variables. The environment variables must be in the format `TF_VAR_name` and this will be checked last for a value. For example:

```shell
export TF_VAR_region=us-west-1
export TF_VAR_ami=ami-049d8641
export TF_VAR_alist='[1,2,3]'
export TF_VAR_amap='{ foo = "bar", baz = "qux" }'
```

For more on how to use `TF_VAR_name` in context, check out the section on [Variable Configuration](/docs/configuration/variables.html).

## TF_CLI_ARGS and TF_CLI_ARGS_name

The value of `TF_CLI_ARGS` will specify additional arguments to the
command-line. This allows easier automation in CI environments as well as
modifying default behavior of Terraform on your own system.

These arguments are inserted directly _after_ the subcommand
(such as `plan`) and _before_ any flags specified directly on the command-line.
This behavior ensures that flags on the command-line take precedence over
environment variables.

For example, the following command: `TF_CLI_ARGS="-input=false" terraform apply -force`
is the equivalent to manually typing: `terraform apply -input=false -force`.

The flag `TF_CLI_ARGS` affects all Terraform commands. If you specify a
named command in the form of `TF_CLI_ARGS_name` then it will only affect
that command. As an example, to specify that only plans never refresh,
you can set `TF_CLI_ARGS_plan="-refresh=false"`.

The value of the flag is parsed as if you typed it directly to the shell.
Double and single quotes are allowed to capture strings and arguments will
be separated by spaces otherwise.

## TF_DATA_DIR

`TF_DATA_DIR` changes the location where Terraform keeps its
per-working-directory data, such as the current remote backend configuration.

By default this data is written into a `.terraform` subdirectory of the
current directory, but the path given in `TF_DATA_DIR` will be used instead
if non-empty.

In most cases it should not be necessary to set this variable, but it may
be useful to do so if e.g. the working directory is not writable.

The data directory is used to retain data that must persist from one command
to the next, so it's important to have this variable set consistently throughout
all of the Terraform workflow commands (starting with `terraform init`) or else
Terraform may be unable to find providers, modules, and other artifacts.
