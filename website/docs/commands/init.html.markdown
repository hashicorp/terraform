---
layout: "docs"
page_title: "Command: init"
sidebar_current: "docs-commands-init"
description: |-
  The `terraform init` command is used to initialize a Terraform configuration. This is the first command that should be run for any new or existing Terraform configuration. It is safe to run this command multiple times.
---

# Command: init

The `terraform init` command is used to initialize a working directory
containing Terraform configuration files. This is the first command that should
be run after writing a new Terraform configuration or cloning an existing one
from version control. It is safe to run this command multiple times.

## Usage

Usage: `terraform init [options] [DIR]`

Initialize a new or existing Terraform working directory by creating
initial files, loading any remote state, downloading modules, etc.

This is the first command that should be run for any new or existing
Terraform configuration per machine. This sets up all the local data
necessary to run Terraform that is typically not committed to version
control.

This command is always safe to run multiple times. Though subsequent runs
may give errors, this command will never delete your configuration or
state. Even so, if you have important information, please back it up prior
to running this command, just in case.

If no arguments are given, the configuration in this working directory
is initialized.

The command-line flags are all optional. The list of available flags are:

* `-backend=true` - Initialize the [backend](/docs/backends) for this configuration.

* `-backend-config=path` This can be either a path to an HCL file with key/value
  assignments (same format as terraform.tfvars) or a 'key=value' format. This is
  merged with what is in the configuration file. This can be specified multiple
  times. The backend type must be in the configuration itself.

* `-force-copy`          Suppress prompts about copying state data. This is
  equivalent to providing a "yes" to all confirmation prompts.

* `-get=true`            Download any modules for this configuration.

* `-get-plugins=true`    Download any missing plugins for this configuration.

* `-input=true`          Ask for input if necessary. If false, will error if
  input was required.

* `-lock=true`           Lock the state file when locking is supported.

* `-lock-timeout=0s`     Duration to retry a state lock.

* `-no-color`            If specified, output won't contain any color.

* `-plugin-dir`          Directory containing plugin binaries. This overrides all
  default search paths for plugins, and prevents the automatic installation of
  plugins. This flag can be used multiple times.

* `-reconfigure`         Reconfigure the backend, ignoring any saved configuration.

* `-upgrade=false`       If installing modules (-get) or plugins (-get-plugins),
  ignore previously-downloaded objects and install the latest version allowed
  within configured constraints.

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
