---
layout: "docs"
page_title: "Environment Variables"
sidebar_current: "docs-config-environment-variables"
description: |-
  Terraform uses different environment variables that can be used to configure various aspects of how Terraform behaves. this section documents those variables, their potential values, and how to use them.
---

# Environment Variables

## TF_LOG

If set to any value, enables detailed logs to appear on stderr which is useful for debugging. For example:

```
export TF_LOG=TRACE
```

To disable, either unset it or set it to empty. When unset, logging will default to stderr. For example:

```
export TF_LOG=
```

For more on debugging Terraform, check out the section on [Debugging](/docs/internals/debugging.html).

## TF_LOG_PATH

This specifies where the log should persist its output to. Note that even when `TF_LOG_PATH` is set, `TF_LOG` must be set in order for any logging to be enabled. For example, to always write the log to the directory you're currently running terraform from:

```
export TF_LOG_PATH=./terraform.log
```

For more on debugging Terraform, check out the section on [Debugging](/docs/internals/debugging.html).

## TF_INPUT

If set to "false" or "0", causes terraform commands to behave as if the `-input=false` flag was specified. This is used when you want to disable prompts for variables that haven't had their values specified. For example:

```
export TF_INPUT=0
```

## TF_MODULE_DEPTH

When given a value, causes terraform commands to behave as if the `-module=depth=VALUE` flag was specified. Modules are treated like a black box and terraform commands do not show what resources within the module will be created. By setting this to -1, for example, you enable commands such as [plan](/docs/commands/plan.html) and [graph](/docs/commands/graph.html) to display more detailed information.

```
export TF_MODULE_DEPTH=-1
```

For more information regarding modules, check out the section on [Using Modules](/docs/modules/usage.html).

## TF_VAR_name

Environment variables can be used to set variables. The environment variables must be in the format `TF_VAR_name` and this will be checked last for a value. For example:

```
export TF_VAR_region=us-west-1
export TF_VAR_ami=ami-049d8641
```

For more on how to use `TF_VAR_name` in context, check out the section on [Variable Configuration](/docs/configuration/variables.html).
