---
layout: "docs"
page_title: "Command: init"
sidebar_current: "docs-commands-init"
description: |-
  The `terraform init` command is used to initialize a Terraform configuration. This is the first command that should be run for any new or existing Terraform configuration. It is safe to run this command multiple times.
---

# Command: init

The `terraform init` command is used to initialize a Terraform configuration.
This is the first command that should be run for any new or existing
Terraform configuration. It is safe to run this command multiple times.

## Usage

Usage: `terraform init [options] [SOURCE] [PATH]`

Initialize a new or existing Terraform environment by creating
initial files, loading any remote state, downloading modules, etc.

This is the first command that should be run for any new or existing
Terraform configuration per machine. This sets up all the local data
necessary to run Terraform that is typically not committed to version
control.

This command is always safe to run multiple times. Though subsequent runs
may give errors, this command will never blow away your environment or state.
Even so, if you have important information, please back it up prior to
running this command just in case.

If no arguments are given, the configuration in this working directory
is initialized.

If one or two arguments are given, the first is a SOURCE of a module to
download to the second argument PATH. After downloading the module to PATH,
the configuration will be initialized as if this command were called pointing
only to that PATH. PATH must be empty of any Terraform files. Any
conflicting non-Terraform files will be overwritten. The module download
is a copy. If you're downloading a module from Git, it will not preserve
Git history.

The command-line flags are all optional. The list of available flags are:

* `-backend=true` - Initialize the [backend](/docs/backends) for this environment.

* `-backend-config=value` - Value can be a path to an HCL file or a string
  in the format of 'key=value'. This specifies additional configuration to merge
  for the backend. This can be specified multiple times. Flags specified
  later in the line override those specified earlier if they conflict.

* `-force-copy` -  Suppress prompts about copying state data. This is equivalent
  to providing a "yes" to all confirmation prompts.

* `-get=true` - Download any modules for this configuration.

* `-input=true` - Ask for input interactively if necessary. If this is false
  and input is required, `init` will error.

* `-lock=true` - Lock the state file when locking is supported.

* `-lock-timeout=0s` - Duration to retry a state lock.

* `-no-color` - If specified, output won't contain any color.

* `-reconfigure` - Reconfigure the backend, ignoring any saved configuration.

## Backend Config

The `-backend-config` can take a path or `key=value` pair to specify additional
backend configuration when [initializing a backend](/docs/backends/init.html).

This is particularly useful for
[partial configuration of backends](/docs/backends/config.html). Partial
configuration lets you keep sensitive information out of your Terraform
configuration.

For path values, the backend configuration file is a basic HCL file with key/value pairs.
The keys are configuration keys for your backend. You do not need to wrap it
in a `terraform` block. For example, the following file is a valid backend
configuration file for the Consul backend type:

```hcl
address = "demo.consul.io"
path    = "newpath"
```

If the value contains an equal sign (`=`), it is parsed as a `key=value` pair.
The format of this flag is identical to the `-var` flag for plan, apply,
etc. but applies to configuration keys for backends. For example:

```shell
$ terraform init \
  -backend-config 'address=demo.consul.io' \
  -backend-config 'path=newpath'
```

These two formats can be mixed. In this case, the values will be merged by
key with keys specified later in the command-line overriding conflicting
keys specified earlier.
